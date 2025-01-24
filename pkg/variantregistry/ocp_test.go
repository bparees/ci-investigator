package variantregistry

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	v1 "github.com/openshift/sippy/pkg/apis/config/v1"
)

func TestVariantSyncer(t *testing.T) {
	variantSyncer := OCPVariantLoader{config: &v1.SippyConfig{}}
	tests := []struct {
		job          string
		variantsFile map[string]string
		expected     map[string]string
	}{
		{
			job:          "periodic-ci-redhat-chaos-prow-scripts-main-cr-4.19-nightly-krkn-hub-aws",
			variantsFile: map[string]string{},
			expected: map[string]string{
				VariantRelease:          "4.19",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "19",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "chaos",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantAggregation:      VariantNoValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantContainerRuntime: "crun",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-osde2e-main-nightly-4.17-conformance-rosa-classic-sts",
			variantsFile: map[string]string{
				"Platform": "aws", // should be ignored
			},
			expected: map[string]string{
				VariantRelease:          "4.17",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "17",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "rosa",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "service-delivery",
				VariantTopology:         "ha",
				VariantSuite:            "parallel",
				VariantUpgrade:          VariantNoValue,
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantAggregation:      VariantNoValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.16-e2e-gcp-ovn-fips",
			variantsFile: map[string]string{
				"Foo":         "bar",          // should be added
				"CloudRegion": "us-central-1", // should be ignored
			},
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "gcp",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "ha",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantSecurityMode:     "fips",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				"Foo":                   "bar",
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.19-e2e-osd-ccs-gcp",
			variantsFile: map[string]string{
				"Platform": "gcp", // should be ignored
			},
			expected: map[string]string{
				VariantRelease:          "4.19",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "19",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "osd-gcp",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "service-delivery",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "crun",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.19-e2e-rosa-sts-ovn",
			variantsFile: map[string]string{
				"Platform": "aws", // should be ignored
			},
			expected: map[string]string{
				VariantRelease:          "4.19",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "19",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "rosa",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "service-delivery",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "crun",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-hypershift-release-4.16-periodics-e2e-aws-ovn-conformance",
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantArch:             "amd64",
				VariantInstaller:        "hypershift", // hypershift uses it's own installer
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "external",
				VariantSuite:            "parallel",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.16-e2e-aws-ovn-single-node-serial",
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "single",
				VariantSuite:            "serial",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.16-e2e-aws-ovn-sno-serial",
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "single",
				VariantSuite:            "serial",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.16-e2e-vsphere-ovn-upi-serial",
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantReleaseMajor:     "4",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantReleaseMinor:     "16",
				VariantArch:             "amd64",
				VariantInstaller:        "upi",
				VariantPlatform:         "vsphere",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "ha",
				VariantSuite:            "serial",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.16-e2e-aws-ovn-proxy",
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantNetwork:          "ovn",
				VariantNetworkAccess:    "proxy",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "ha",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-multiarch-master-nightly-4.17-ocp-e2e-aws-ovn-multi-x-ax",
			variantsFile: map[string]string{
				"Architecture": "amd64", // should be overruled by the job parsing.
			},
			expected: map[string]string{
				VariantRelease:          "4.17",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "17",
				VariantArch:             "heterogeneous",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-multiarch-master-nightly-4.17-ocp-e2e-aws-ovn-multi-a-a",
			variantsFile: map[string]string{
				"Architecture": "amd64", // should be overruled by the job parsing.
			},
			expected: map[string]string{
				VariantRelease:          "4.17",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "17",
				VariantArch:             "arm64",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-cluster-control-plane-machine-set-operator-release-4.17-periodics-e2e-aws-arm",
			variantsFile: map[string]string{
				"Architecture": "amd64", // should be overruled by the job parsing.
			},
			expected: map[string]string{
				VariantRelease:          "4.17",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "17",
				VariantArch:             "arm64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-multiarch-master-nightly-4.17-ocp-e2e-ovn-powervs-capi-multi-p-p",
			variantsFile: map[string]string{
				"Architecture": "amd64", // should be overruled by the job parsing.
			},
			expected: map[string]string{
				VariantRelease:          "4.17",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "17",
				VariantArch:             "ppc64le",
				VariantInstaller:        "ipi",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-multiarch-master-nightly-4.17-ocp-e2e-ovn-remote-libvirt-multi-z-z",
			variantsFile: map[string]string{
				"Architecture": "amd64", // should be overruled by the job parsing.
			},
			expected: map[string]string{
				VariantRelease:          "4.17",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "17",
				VariantArch:             "s390x",
				VariantInstaller:        "ipi",
				VariantPlatform:         "libvirt",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-multiarch-master-nightly-4.16-upgrade-from-nightly-4.15-ocp-e2e-upgrade-gcp-ovn-heterogeneous",
			variantsFile: map[string]string{
				"Architecture": "amd64", // should be overruled by the job parsing.
			},
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantFromRelease:      "4.15",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantFromReleaseMajor: "4",
				VariantFromReleaseMinor: "15",
				VariantArch:             "heterogeneous",
				VariantInstaller:        "ipi",
				VariantPlatform:         "gcp",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          "minor",
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.16-e2e-metal-ipi-sdn-bm-upgrade",
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantFromRelease:      "4.16",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantFromReleaseMajor: "4",
				VariantFromReleaseMinor: "16",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "metal",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "sdn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          "micro",
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.16-e2e-metal-ovn-assisted",
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantArch:             "amd64",
				VariantInstaller:        "assisted",
				VariantPlatform:         "metal",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.16-e2e-no-network-plugin-no-variant-file",
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.11-e2e-no-network-plugin-no-variant-file",
			expected: map[string]string{
				VariantRelease:          "4.11",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "11",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantNetwork:          "sdn", // should default to sdn prior to 4.12
				VariantNetworkStack:     "ipv4",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "release-openshift-origin-installer-e2e-aws-upgrade-4.13-to-4.14-to-4.15-to-4.16-ci",
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantFromRelease:      "4.13",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantFromReleaseMajor: "4",
				VariantFromReleaseMinor: "13",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          "multi",
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-ci-4.15-upgrade-from-stable-4.14-from-stable-4.13-e2e-aws-sdn-upgrade",
			expected: map[string]string{
				VariantRelease:          "4.15",
				VariantFromRelease:      "4.13",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "15",
				VariantFromReleaseMajor: "4",
				VariantFromReleaseMinor: "13",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "sdn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          "multi",
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-with-no-release-info",
			expected: map[string]string{
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.16-e2e-metal-ipi-ovn-dualstack",
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "metal",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "dual",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-nightly-4.16-upgrade-from-stable-4.15-e2e-metal-ipi-upgrade-ovn-ipv6",
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantFromRelease:      "4.15",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantFromReleaseMajor: "4",
				VariantFromReleaseMinor: "15",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "metal",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv6",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          "minor",
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job: "periodic-ci-openshift-release-master-ci-4.16-e2e-aws-ovn-upgrade-out-of-change",
			expected: map[string]string{
				VariantRelease:          "4.16",
				VariantFromRelease:      "4.16",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "16",
				VariantFromReleaseMajor: "4",
				VariantFromReleaseMinor: "16",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantSuite:            "unknown",
				VariantTopology:         "ha",
				VariantUpgrade:          "micro-downgrade",
				VariantAggregation:      VariantNoValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job:          "periodic-ci-openshift-release-master-nightly-4.17-e2e-telco5g",
			variantsFile: map[string]string{},
			expected: map[string]string{
				VariantRelease:          "4.17",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "17",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "cnf",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job:          "periodic-ci-openshift-qe-ocp-qe-perfscale-ci-main-aws-4.17-nightly-x86-payload-control-plane-6nodes",
			variantsFile: map[string]string{},
			expected: map[string]string{
				VariantRelease:          "4.17",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "17",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "perfscale",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job:          "periodic-ci-openshift-release-master-nightly-4.18-e2e-aws-ovn-crun",
			variantsFile: map[string]string{},
			expected: map[string]string{
				VariantRelease:          "4.18",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "18",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantContainerRuntime: "crun",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job:          "periodic-ci-openshift-release-master-nightly-4.18-e2e-aws-ovn-cgroupsv1",
			variantsFile: map[string]string{},
			expected: map[string]string{
				VariantRelease:          "4.18",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "18",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantContainerRuntime: "crun",
				VariantCGroupMode:       "v1",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job:          "periodic-ci-openshift-release-master-nightly-4.18-e2e-aws-virt-ovn-techpreview",
			variantsFile: map[string]string{},
			expected: map[string]string{
				VariantRelease:          "4.18",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "18",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantFeatureSet:       "techpreview",
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantContainerRuntime: "crun",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   "virt",
			},
		},
		{
			job:          "periodic-ci-openshift-release-master-nightly-4.18-e2e-aws-virt-ovn-runc-techpreview",
			variantsFile: map[string]string{},
			expected: map[string]string{
				VariantRelease:          "4.18",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "18",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantFeatureSet:       "techpreview",
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   "virt",
			},
		},
		{
			job:          "periodic-ci-openshift-release-master-nightly-4.17-e2e-aws-ovn-cgroupsv1-crun",
			variantsFile: map[string]string{},
			expected: map[string]string{
				VariantRelease:          "4.17",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "17",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantContainerRuntime: "crun",
				VariantCGroupMode:       "v1",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job:          "periodic-ci-openshift-release-master-nightly-4.17-e2e-aws-ovn-cgroupsv1",
			variantsFile: map[string]string{},
			expected: map[string]string{
				VariantRelease:          "4.17",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "17",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantFeatureSet:       VariantDefaultValue,
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v1",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job:          "periodic-ci-openshift-openshift-tests-private-release-4.17-automated-release-aws-ipi-f999",
			variantsFile: map[string]string{},
			expected: map[string]string{
				VariantRelease:          "4.17",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "17",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "automated-release",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "qe",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantFeatureSet:       "default",
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   VariantNoValue,
			},
		},
		{
			job:          "periodic-ci-openshift-release-master-nightly-4.17-e2e-aws-virt-ovn-techpreview",
			variantsFile: map[string]string{},
			expected: map[string]string{
				VariantRelease:          "4.17",
				VariantReleaseMajor:     "4",
				VariantReleaseMinor:     "17",
				VariantArch:             "amd64",
				VariantInstaller:        "ipi",
				VariantPlatform:         "aws",
				VariantProcedure:        "none",
				VariantJobTier:          "standard",
				VariantNetwork:          "ovn",
				VariantNetworkStack:     "ipv4",
				VariantOwner:            "eng",
				VariantTopology:         "ha",
				VariantSuite:            "unknown",
				VariantUpgrade:          VariantNoValue,
				VariantAggregation:      VariantNoValue,
				VariantSecurityMode:     VariantDefaultValue,
				VariantFeatureSet:       "techpreview",
				VariantNetworkAccess:    VariantDefaultValue,
				VariantScheduler:        VariantDefaultValue,
				VariantContainerRuntime: "runc",
				VariantCGroupMode:       "v2",
				VariantLayeredProduct:   "virt",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.job, func(t *testing.T) {
			assert.Equal(t, test.expected,
				variantSyncer.CalculateVariantsForJob(
					logrus.WithField("source", "TestVariantSyncer"),
					test.job,
					test.variantsFile))
		})
	}
}
