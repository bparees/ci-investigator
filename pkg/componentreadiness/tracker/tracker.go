package tracker

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/google/uuid"
	"github.com/openshift/sippy/pkg/apis/api"
	sippybigquery "github.com/openshift/sippy/pkg/bigquery"
	"github.com/openshift/sippy/pkg/componentreadiness/resolvedissues"
	"github.com/openshift/sippy/pkg/variantregistry"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
)

const (
	testRegressionsTable = "test_regressions"
)

// RegressionStore is an underlying interface for where we store/load data on open test regressions.
type RegressionStore interface {
	ListCurrentRegressions(release string) ([]api.TestRegression, error)
	OpenRegression(release string, newRegressedTest api.ComponentReportTestSummary) (*api.TestRegression, error)
	ReOpenRegression(regressionID string) error
	CloseRegression(regressionID string, closedAt time.Time) error
}

// BigQueryRegressionStore is the primary implementation for real world usage, storing when regressions appear/disappear in BigQuery.
type BigQueryRegressionStore struct {
	client *sippybigquery.Client
}

func NewBigQueryRegressionStore(client *sippybigquery.Client) RegressionStore {
	return &BigQueryRegressionStore{client: client}
}

func (bq *BigQueryRegressionStore) ListCurrentRegressions(release string) ([]api.TestRegression, error) {
	// List open regressions (no closed date), or those that closed within the last two days. This is to prevent flapping
	// and return more accurate opened dates when a test is falling in / out of the report.
	queryString := fmt.Sprintf("SELECT * FROM %s.%s WHERE release = @SampleRelease AND (closed IS NULL or closed > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 2 DAY))",
		bq.client.Dataset, testRegressionsTable)

	params := make([]bigquery.QueryParameter, 0)
	params = append(params, []bigquery.QueryParameter{
		{
			Name:  "SampleRelease",
			Value: release,
		},
	}...)

	sampleQuery := bq.client.BQ.Query(queryString)
	sampleQuery.Parameters = append(sampleQuery.Parameters, params...)

	regressions := make([]api.TestRegression, 0)
	log.Infof("Fetching current test regressions with:\n%s\nParameters:\n%+v\n",
		sampleQuery.Q, sampleQuery.Parameters)

	it, err := sampleQuery.Read(context.TODO())
	if err != nil {
		log.WithError(err).Error("error querying triaged incidents from bigquery")
		return regressions, err
	}

	for {
		var regression api.TestRegression
		err := it.Next(&regression)
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.WithError(err).Error("error parsing triaged incident from bigquery")
			return nil, errors.Wrap(err, "error parsing triaged incident from bigquery")
		}
		regressions = append(regressions, regression)
	}
	return regressions, nil

}
func (bq *BigQueryRegressionStore) OpenRegression(release string, newRegressedTest api.ComponentReportTestSummary) (*api.TestRegression, error) {
	id := uuid.New()
	newRegression := &api.TestRegression{
		Release:      release,
		TestID:       newRegressedTest.TestID,
		TestName:     newRegressedTest.TestName,
		RegressionID: id.String(),
		Opened:       time.Now(),
		Variants: []api.ComponentReportVariant{
			{
				Key:   variantregistry.VariantNetwork,
				Value: newRegressedTest.Network,
			},
			{
				Key:   variantregistry.VariantUpgrade,
				Value: newRegressedTest.Upgrade,
			},
			{
				Key:   variantregistry.VariantPlatform,
				Value: newRegressedTest.Platform,
			},
			{
				Key:   variantregistry.VariantArch,
				Value: newRegressedTest.Arch,
			},
			{
				Key:   resolvedissues.VariantVariant,
				Value: newRegressedTest.Variant,
			},
		},
	}
	inserter := bq.client.BQ.Dataset(bq.client.Dataset).Table(testRegressionsTable).Inserter()
	items := []*api.TestRegression{
		newRegression,
	}
	if err := inserter.Put(context.TODO(), items); err != nil {
		return nil, err
	}
	return newRegression, nil

}

func (bq *BigQueryRegressionStore) ReOpenRegression(regressionID string) error {
	return bq.updateClosed(regressionID, "NULL")
}

func (bq *BigQueryRegressionStore) CloseRegression(regressionID string, closedAt time.Time) error {
	return bq.updateClosed(regressionID,
		fmt.Sprintf("'%s'", closedAt.Format("2006-01-02 15:04:05.999999")))
}

func (bq *BigQueryRegressionStore) updateClosed(regressionID, closed string) error {
	queryString := fmt.Sprintf("UPDATE %s.%s SET closed = %s WHERE regression_id = '%s'",
		bq.client.Dataset, testRegressionsTable, closed, regressionID)

	query := bq.client.BQ.Query(queryString)

	job, err := query.Run(context.TODO())
	if err != nil {
		return err
	}

	status, err := job.Wait(context.TODO())
	if err != nil {
		return err
	}

	err = status.Err()
	return err
}

func NewRegressionTracker(backend RegressionStore, dryRun bool) *RegressionTracker {
	return &RegressionTracker{
		backend: backend,
		dryRun:  dryRun,
	}
}

// RegressionTracker is the primary object for managing regression tracking logic.
type RegressionTracker struct {
	backend RegressionStore
	dryRun  bool
}

func (rt *RegressionTracker) SyncComponentReport(release string, report *api.ComponentReport) error {
	regressions, err := rt.backend.ListCurrentRegressions(release)
	if err != nil {
		return err
	}
	rLog := log.WithField("func", "SyncComponentReport").WithField("dryRun", rt.dryRun)
	rLog.Infof("loaded %d regressions from db", len(regressions))

	// All regressions, both triaged and not:
	allRegressedTests := []api.ComponentReportTestSummary{}
	for _, row := range report.Rows {
		for _, col := range row.Columns {
			allRegressedTests = append(allRegressedTests, col.RegressedTests...)
			// Once triaged, regressions move to this list, we want to still consider them an open regression until
			// the report says they're cleared and they disappear fully. Triaged does not imply fixed or no longer
			// a regression.
			for _, triaged := range col.TriagedIncidents {
				allRegressedTests = append(allRegressedTests, triaged.ComponentReportTestSummary)
			}
		}
	}

	matchedOpenRegressions := []api.TestRegression{} // all the matches we found, used to determine what had no match
	for _, regTest := range allRegressedTests {
		if openReg := FindOpenRegression(release, regTest, regressions); openReg != nil {
			if openReg.Closed.Valid {
				// if the regression returned has a closed date, we found a recently closed
				// regression for this test. We'll re-use it to limit churn as sometimes tests may drop
				// in / out of the report depending on the data available in the sample/basis.
				rLog.Infof("re-opening existing regression: %v", openReg)
				if !rt.dryRun {
					err := rt.backend.ReOpenRegression(openReg.RegressionID)
					if err != nil {
						rLog.WithError(err).Errorf("error re-opening regression: %v", openReg)
						return errors.Wrapf(err, "error re-opening regression: %v", openReg)
					}
				}
			} else {
				rLog.WithFields(log.Fields{
					"test": regTest.TestName,
				}).Infof("reusing already opened regression: %v", openReg)

			}
			matchedOpenRegressions = append(matchedOpenRegressions, *openReg)
		} else {
			rLog.Infof("opening new regression: %v", regTest)
			if !rt.dryRun {
				// Open a new regression:
				newReg, err := rt.backend.OpenRegression(release, regTest)
				if err != nil {
					rLog.WithError(err).Errorf("error opening new regression for: %v", regTest)
					return errors.Wrapf(err, "error opening new regression: %v", regTest)
				}
				rLog.Infof("new regression opened with id: %s", newReg.RegressionID)
			}
		}
	}

	// Now we want to close any open regressions that are not appearing in the latest report:
	now := time.Now()
	for _, regression := range regressions {
		var matched bool
		for _, m := range matchedOpenRegressions {
			if reflect.DeepEqual(m, regression) {
				matched = true
				break
			}
		}
		// If we didn't match to an active test regression, and this record isn't already closed, close it.
		if !matched && !regression.Closed.Valid {
			rLog.Infof("found a regression no longer appearing in the report which should be closed: %v", regression)
			if !rt.dryRun {
				err := rt.backend.CloseRegression(regression.RegressionID, now)
				if err != nil {
					rLog.WithError(err).Errorf("error closing regression: %v", regression)
					return errors.Wrap(err, "error closing regression")
				}
			}
		}

	}

	return nil
}

// FindOpenRegression scans the list of open regressions for any that match the given test summary.
func FindOpenRegression(release string,
	regTest api.ComponentReportTestSummary,
	regressions []api.TestRegression) *api.TestRegression {

	for _, tr := range regressions {
		if tr.Release != release {
			continue
		}
		// We compare test ID not name, as names can change.
		if tr.TestID != regTest.TestID {
			continue
		}
		// TODO: needs updating when we're ready for dynamic variants
		if regTest.Network != findVariant(variantregistry.VariantNetwork, tr) {
			continue
		}
		if regTest.Upgrade != findVariant(variantregistry.VariantUpgrade, tr) {
			continue
		}
		if regTest.Platform != findVariant(variantregistry.VariantPlatform, tr) {
			continue
		}
		if regTest.Variant != findVariant(resolvedissues.VariantVariant, tr) {
			continue
		}
		if regTest.Arch != findVariant(variantregistry.VariantArch, tr) {
			continue
		}
		// If we made it this far, this appears to be a match:
		return &tr
	}
	return nil
}

func findVariant(variantName string, testReg api.TestRegression) string {
	for _, v := range testReg.Variants {
		if v.Key == variantName {
			return v.Value
		}
	}
	return ""
}
