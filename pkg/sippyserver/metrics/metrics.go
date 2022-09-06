package metrics

import (
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"

	"github.com/openshift/sippy/pkg/filter"
	"github.com/openshift/sippy/pkg/util"

	"github.com/openshift/sippy/pkg/api"
	apitype "github.com/openshift/sippy/pkg/apis/api"
	sippyprocessingv1 "github.com/openshift/sippy/pkg/apis/sippyprocessing/v1"
	"github.com/openshift/sippy/pkg/db"
	"github.com/openshift/sippy/pkg/db/query"
)

const (
	blockerScoreToAlertOn = 50
)

var (
	buildClusterHealthMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sippy_build_cluster_pass_ratio",
		Help: "Ratio of passed job runs for a build cluster in a period (2 day, 7 day, etc)",
	}, []string{"cluster", "period"})
	jobPassRatioMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sippy_job_pass_ratio",
		Help: "Ratio of passed job runs for the given job in a period (2 day, 7 day, etc)",
	}, []string{"release", "period", "name"})
	releaseWarningsMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sippy_release_warnings",
		Help: "Number of current warnings for a release, see overview page in UI for details",
	}, []string{"release"})
	payloadConsecutiveRejectionsMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sippy_payloads_consecutively_rejected",
		Help: "Number of consecutive rejected payloads in each release, stream and arch combo. Will be 0 if most recent payload accepted.",
	}, []string{"release", "stream", "architecture"})
	payloadHoursSinceLastAcceptedMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sippy_payloads_hours_since_last_accepted",
		Help: "Number of hours since last accepted payload in each release, stream and arch combo.",
	}, []string{"release", "stream", "architecture"})
	payloadHoursSinceLastOSUpgrade = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sippy_payloads_hours_since_last_os_upgrade",
		Help: "Number of hours since last OS upgrade.",
	}, []string{"release", "stream", "architecture"})
	payloadPossibleTestBlockersMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sippy_payloads_possible_test_blockers",
		Help: "Number of possible test blockers identified for a given payload stream.",
	}, []string{"release", "stream", "architecture"})
)

// presume in a historical context there won't be scraping of these metrics
// pinning the time just to be consistent
func RefreshMetricsDB(dbc *db.DB, reportEnd time.Time) error {
	releases, err := query.ReleasesFromDB(dbc)
	if err != nil {
		return err
	}

	promReportTypes := buildPromReportTypes(releases)
	if err != nil {
		return err
	}

	for _, pType := range promReportTypes {
		// start, boundary and end will just be defaults
		// the api will decide based on the period
		// and current day / time
		jobsResult, err := api.JobReportsFromDB(dbc, pType.release, pType.period, nil, time.Time{}, time.Time{}, time.Time{}, reportEnd)

		if err != nil {
			return errors.Wrapf(err, "error refreshing prom report type %s - %s", pType.period, pType.release)
		}
		for _, jobResult := range jobsResult {
			jobPassRatioMetric.WithLabelValues(pType.release, pType.period, jobResult.Name).Set(jobResult.CurrentPassPercentage / 100)
		}
	}

	// Add a metric for any warnings for each release. We can't convey exact details with prom, but we can
	// tell you x warnings are present and link you to the overview in the alert.
	for _, release := range releases {
		releaseWarnings := api.ScanForReleaseWarnings(dbc, release.Release, reportEnd)
		releaseWarningsMetric.WithLabelValues(release.Release).Set(float64(len(releaseWarnings)))
	}

	if err := refreshBuildClusterMetrics(dbc, reportEnd); err != nil {
		return errors.Wrapf(err, "error refreshing build cluster metrics")
	}

	refreshPayloadMetrics(dbc, reportEnd)

	return nil
}

func refreshBuildClusterMetrics(dbc *db.DB, reportEnd time.Time) error {
	for _, period := range []string{"default", "twoDay"} {
		start, boundary, end := util.PeriodToDates(period, reportEnd)
		result, err := query.BuildClusterHealth(dbc, start, boundary, end)
		if err != nil {
			return err
		}

		for _, cluster := range result {
			buildClusterHealthMetric.WithLabelValues(cluster.Cluster, period).Set(cluster.CurrentPassPercentage / 100)
		}
	}

	return nil
}

func refreshPayloadMetrics(dbc *db.DB, reportEnd time.Time) {
	releases, err := query.ReleasesFromDB(dbc)
	if err != nil {
		log.WithError(err).Error("error querying releases from db")
		return
	}
	for _, r := range releases {
		results, err := api.ReleaseHealthReports(dbc, r.Release, reportEnd)
		if err != nil {
			log.WithError(err).Error("error calling ReleaseHealthReports")
			return
		}

		for _, rhr := range results {
			count := 0
			if rhr.LastPhase == apitype.PayloadRejected {
				count = rhr.Count
			}
			payloadConsecutiveRejectionsMetric.WithLabelValues(r.Release, rhr.Stream, rhr.Architecture).Set(float64(count))

			// Piggy back the results here to use the list of arch+streams:
			if rhr.LastPhase == apitype.PayloadRejected {
				possibleTestBlockers, err := api.GetPayloadStreamTestFailures(dbc, r.Release, rhr.Stream,
					rhr.Architecture, &filter.FilterOptions{Filter: &filter.Filter{}}, reportEnd)
				if err != nil {
					log.WithError(err).Error("error getting payload stream test failures")
					return
				}
				blockersFound := 0
				for _, t := range possibleTestBlockers {
					if t.BlockerScore >= blockerScoreToAlertOn {
						blockersFound++
					}
				}
				payloadPossibleTestBlockersMetric.WithLabelValues(r.Release, rhr.Stream, rhr.Architecture).
					Set(float64(blockersFound))
			}
		}

		lastAcceptedReleaseTags, err := query.GetLastAcceptedByArchitectureAndStream(dbc.DB, r.Release)
		if err != nil {
			log.WithError(err).Error("error querying last accepted payloads")
			return
		}

		for _, archStream := range lastAcceptedReleaseTags {
			sinceLastAccepted := time.Since(archStream.ReleaseTime)
			payloadHoursSinceLastAcceptedMetric.WithLabelValues(r.Release, archStream.Stream, archStream.Architecture).Set(sinceLastAccepted.Hours())
		}

		lastOSUpgradeTags, err := query.GetLastOSUpgradeByArchitectureAndStream(dbc.DB, r.Release)
		if err != nil {
			log.WithError(err).Error("error querying last os upgrades")
			return
		}
		for _, archStream := range lastOSUpgradeTags {
			sinceLastOS := time.Since(archStream.ReleaseTime)
			payloadHoursSinceLastOSUpgrade.WithLabelValues(r.Release, archStream.Stream, archStream.Architecture).Set(sinceLastOS.Hours())
		}

	}
}

type promReportType struct {
	release string
	period  string
}

func buildPromReportTypes(releases []query.Release) []promReportType {
	var promReportTypes []promReportType

	for _, release := range releases {
		promReportTypes = append(promReportTypes, promReportType{release: release.Release, period: string(sippyprocessingv1.TwoDayReport)})
		promReportTypes = append(promReportTypes, promReportType{release: release.Release, period: string(sippyprocessingv1.CurrentReport)})
	}

	return promReportTypes
}
