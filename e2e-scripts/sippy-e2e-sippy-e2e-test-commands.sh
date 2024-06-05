#!/bin/bash

# test
set -x

function cleanup() {
  echo "Cleaning up port forward"
  pf_job=$(jobs -p)
  kill ${pf_job} && wait
  echo "Port forward is cleaned up"
}
trap cleanup EXIT

# In Prow CI, SIPPY_IMAGE variable is defined in the sippy-e2e-ref.yaml file as a
# dependency so that the pipeline:sippy image (containing the sippy binary)
# will be available to start the sippy-load and sippy-server pods.
# When running locally, the user has to define SIPPY_IMAGE.
echo "The sippy CI image: ${SIPPY_IMAGE}"

# The GCS_CRED allows us to pull artifacts from GCS when importing prow jobs.
# Redefine GCS_CRED to use your own.
GCS_CRED="${GCS_CRED:=/var/run/sippy-bigquery-job-importer/gcs-sa}"
echo "The GCS cred is: ${GCS_CRED}"

# If you're using Openshift, we use oc, if you're using plain Kubernetes,
# we use kubectl.
#
KUBECTL_CMD="${KUBECTL_CMD:=oc}"
echo "The kubectl command is: ${KUBECTL_CMD}"

# Get the gcs credentials out to the cluster-pool cluster.
# These credentials are in vault and maintained by the TRT team (e.g. for updates and rotations).
# See https://vault.ci.openshift.org/ui/vault/secrets/kv/show/selfservice/technical-release-team/sippy-ci-gcs-read-sa
#
${KUBECTL_CMD} create secret generic gcs-cred --from-file gcs-cred=$GCS_CRED -n sippy-e2e

# Launch the sippy api server pod.
cat << END | ${KUBECTL_CMD} apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: sippy-server
  namespace: sippy-e2e
  labels:
    app: sippy-server
spec:
  containers:
  - name: sippy-server
    image: ${SIPPY_IMAGE}
    imagePullPolicy: ${SIPPY_IMAGE_PULL_POLICY:-Always}
    ports:
    - name: www
      containerPort: 8080
      protocol: TCP
    - name: metrics
      containerPort: 12112
      protocol: TCP
    readinessProbe:
      exec:
        command:
        - echo
        - "Wait for a short time"
      initialDelaySeconds: 10
    resources:
      limits:
        memory: 2Gi
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    command:
    - /bin/sippy
    args:
    - serve
    - --listen
    - ":8080"
    - --listen-metrics
    -  ":12112"
    - --database-dsn=postgresql://postgres:password@postgres.sippy-e2e.svc.cluster.local:5432/postgres
    - --google-service-account-credential-file
    - /tmp/secrets/gcs-cred
    - --log-level
    - debug
    - --mode
    - ocp
    env:
    - name: GCS_SA_JSON_PATH
      value: /tmp/secrets/gcs-cred
    volumeMounts:
    - mountPath: /tmp/secrets
      name: gcs-cred
      readOnly: true
  imagePullSecrets:
  - name: regcred
  volumes:
    - name: gcs-cred
      secret:
        secretName: gcs-cred
  dnsPolicy: ClusterFirst
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  terminationGracePeriodSeconds: 30
END

# The basic readiness probe will give us at least 10 seconds before declaring the pod as ready.
echo "Waiting for sippy api server pod to be Ready ..."
${KUBECTL_CMD} -n sippy-e2e wait --for=condition=Ready pod/sippy-server --timeout=30s

${KUBECTL_CMD} -n sippy-e2e get pod -o wide
${KUBECTL_CMD} -n sippy-e2e logs sippy-server > ${ARTIFACT_DIR}/sippy-server.log

echo "Setup services and port forwarding for the sippy api server ..."

# Create the Kubernetes service for the sippy-server pod
# Setup port forward for port 18080 to get to the sippy-server pod
${KUBECTL_CMD} -n sippy-e2e expose pod sippy-server
${KUBECTL_CMD} -n sippy-e2e port-forward pod/sippy-server 8080:8080 &
SIPPY_API_PORT=8080
export SIPPY_API_PORT

${KUBECTL_CMD} -n sippy-e2e get svc,ep

${KUBECTL_CMD} -n sippy-e2e delete secret regcred

go test ./test/e2e/ -v
