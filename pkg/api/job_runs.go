package api

import (
	"fmt"
	"net/http"
	gosort "sort"
	"strconv"
	"time"

	"github.com/hashicorp/go-version"
	apitype "github.com/openshift/sippy/pkg/apis/api"
	"github.com/openshift/sippy/pkg/db"
	"github.com/openshift/sippy/pkg/db/models"
	"github.com/openshift/sippy/pkg/db/query"
	"github.com/openshift/sippy/pkg/filter"
	"github.com/openshift/sippy/pkg/testidentification"
	log "github.com/sirupsen/logrus"
)

const (
	// maxFailuresToFullyAnalyze is a limit to the number of failures we'll attempt to
	// individually analyze, if you exceed this the job failure is classified as high risk.
	maxFailuresToFullyAnalyze = 20
)

func (runs apiRunResults) sort(req *http.Request) apiRunResults {
	sortField := req.URL.Query().Get("sortField")
	sort := apitype.Sort(req.URL.Query().Get("sort"))

	if sortField == "" {
		sortField = "test_failures"
	}

	if sort == "" {
		sort = apitype.SortDescending
	}

	gosort.Slice(runs, func(i, j int) bool {
		if sort == apitype.SortAscending {
			return filter.Compare(runs[i], runs[j], sortField)
		}
		return filter.Compare(runs[j], runs[i], sortField)
	})

	return runs
}

func (runs apiRunResults) limit(req *http.Request) apiRunResults {
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	if limit > 0 && len(runs) >= limit {
		return runs[:limit]
	}

	return runs
}

type apiRunResults []apitype.JobRun

// JobsRunsReportFromDB renders a filtered summary of matching jobs.
func JobsRunsReportFromDB(dbc *db.DB, filterOpts *filter.FilterOptions, release string, pagination *apitype.Pagination, reportEnd time.Time) (*apitype.PaginationResult, error) {
	jobsResult := make([]apitype.JobRun, 0)
	table := "prow_job_runs_report_matview"
	q, err := filter.FilterableDBResult(dbc.DB.Table(table), filterOpts, apitype.JobRun{})
	if err != nil {
		return nil, err
	}

	if len(release) > 0 {
		q = q.Where("release = ?", release)
	}

	q = q.Where("timestamp < ?", reportEnd.UnixMilli())

	// Get the row count before pagination
	var rowCount int64
	q.Count(&rowCount)

	// Paginate the results:
	if pagination == nil {
		pagination = &apitype.Pagination{
			PerPage: int(rowCount),
			Page:    0,
		}
	} else {
		q = q.Limit(pagination.PerPage).Offset(pagination.Page * pagination.PerPage)
	}

	res := q.Scan(&jobsResult)
	return &apitype.PaginationResult{
		Rows:      jobsResult,
		TotalRows: rowCount,
		PageSize:  pagination.PerPage,
		Page:      pagination.Page,
	}, res.Error
}

func FetchJobRun(dbc *db.DB, jobRunID int64, logger *log.Entry) (*models.ProwJobRun, int, error) {

	jobRun := &models.ProwJobRun{}
	// Load the ProwJobRun, ProwJob, and failed tests:
	// TODO: we may want to expand to analyzing flakes here in the future
	res := dbc.DB.Joins("ProwJob").
		Preload("Tests", "status = 12").
		Preload("Tests.Test").
		Preload("Tests.Suite").First(jobRun, jobRunID)
	if res.Error != nil {
		return nil, -1, res.Error
	}

	jobRunTestCount, err := query.JobRunTestCount(dbc, jobRunID)
	if err != nil {
		logger.WithError(err).Error("Error getting job run test count")
		jobRunTestCount = -1
	}

	return jobRun, jobRunTestCount, nil
}

// JobRunRiskAnalysis checks the test failures and linked bugs for a job run, and reports back an estimated
// risk level for each failed test, and the job run overall.
func JobRunRiskAnalysis(dbc *db.DB, jobRun *models.ProwJobRun, jobRunTestCount int, logger *log.Entry) (apitype.ProwJobRunRiskAnalysis, error) {

	// If this job is a Presubmit, compare to test results from master, not presubmits, which may perform
	// worse due to dev code that hasn't merged. We do not presently track presubmits on branches other than
	// master, so it should be safe to assume the latest compareRelease in the db.
	compareRelease := jobRun.ProwJob.Release
	if compareRelease == "Presubmits" {
		// Get latest release from the DB:
		ar, err := query.ReleasesFromDB(dbc)
		if err != nil {
			return apitype.ProwJobRunRiskAnalysis{}, err
		}
		if len(ar) == 0 {
			return apitype.ProwJobRunRiskAnalysis{}, fmt.Errorf("no releases found in db")
		}

		compareRelease = ar[0].Release
	}

	historicalCount, err := query.ProwJobHistoricalTestCounts(dbc, jobRun.ProwJob.ID)

	// if we had an error we will continue the risk analysis and not elevate based on test counts
	if err != nil {
		logger.WithError(err).Error("Error comparing historical job run test count")
		historicalCount = 0
	}

	// -1 indicates an error getting the jobRunTest count we will log an error and skip this validation
	if jobRunTestCount < 0 {
		logger.Error("Unable to determine job run test count, initializing to historical count")
		jobRunTestCount = historicalCount
	} else if jobRunTestCount == 0 {
		// hack since we don't currently get the jobRunTestCount for 4.12 jobs.
		// If the jobRunTestCount is 0 and we are pre 4.13 set the jobRunTestCount to the historicalCount
		preSupportVersion, _ := version.NewVersion("4.12")
		currentVersion, _ := version.NewVersion(compareRelease)
		if preSupportVersion.GreaterThanOrEqual(currentVersion) {
			jobRunTestCount = historicalCount
		}
	}

	// NOTE: we are including bugs for all releases, may want to filter here in future to just those
	// with an AffectsVersions that seems to match our compareRelease?
	jobBugs, err := query.LoadBugsForJobs(dbc, []int{int(jobRun.ProwJob.ID)}, true)
	if err != nil {
		logger.WithError(err).Errorf("Error evaluating bugs for prow job: %d", jobRun.ProwJob.ID)
	} else {
		jobRun.ProwJob.Bugs = jobBugs
	}

	// Pre-load test bugs as well:
	if len(jobRun.Tests) <= maxFailuresToFullyAnalyze {
		for i, tr := range jobRun.Tests {
			bugs, err := query.LoadBugsForTest(dbc, tr.Test.Name, true)
			if err != nil {
				logger.WithError(err).Errorf("Error evaluating bugs for prow job: %d, test name: %s", jobRun.ProwJob.ID, tr.Test.Name)
			} else {
				logger.Infof("Found %d bugs for test %s", len(bugs), tr.Test.Name)
				tr.Test.Bugs = bugs
				jobRun.Tests[i] = tr
			}
		}
	}

	return runJobRunAnalysis(jobRun, compareRelease, jobRunTestCount, historicalCount, logger.WithField("func", "runJobRunAnalysis"),
		func(testName, release, suite string, variants []string) (*apitype.Test, error) {

			fil := &filter.Filter{
				Items: []filter.FilterItem{
					{
						Field:    "name",
						Not:      false,
						Operator: filter.OperatorEquals,
						Value:    testName,
					},
				},
				LinkOperator: "and",
			}
			trs, _, err := BuildTestsResults(dbc, release, "default", false, false,
				fil)
			if err != nil {
				return nil, err
			}
			gosort.Strings(variants)
			for _, tr := range trs {
				// this is a weird way to get the variant we want, but it allows re-use
				// of the existing code.
				gosort.Strings(tr.Variants)
				if stringSlicesEqual(variants, tr.Variants) && tr.SuiteName == suite {
					return &tr, nil
				}
			}

			// otherwise, what is our best match...
			// do something more expensive and check to see
			// which tr contains all the variants we have currently
			for _, tr := range trs {
				// we didn't find an exact variant match
				// next best guess is the first variant list that contains all of our known variants
				if stringSubSlicesEqual(variants, tr.Variants) && tr.SuiteName == suite {
					return &tr, nil
				}
			}

			return nil, nil
		})
}

// testResultsFunc is used for injecting db responses in unit tests.
type testResultsFunc func(testName string, release, suite string, variants []string) (*apitype.Test, error)

func runJobRunAnalysis(jobRun *models.ProwJobRun, compareRelease string, jobRunTestCount int, historicalRunTestCount int, logger *log.Entry,
	testResultsFunc testResultsFunc) (apitype.ProwJobRunRiskAnalysis, error) {

	logger.Info("loaded prow job run for analysis")
	logger.Infof("this job run has %d failed tests", len(jobRun.Tests))

	response := apitype.ProwJobRunRiskAnalysis{
		ProwJobRunID:   jobRun.ID,
		ProwJobName:    jobRun.ProwJob.Name,
		Release:        jobRun.ProwJob.Release,
		CompareRelease: compareRelease,
		Tests:          []apitype.ProwJobRunTestRiskAnalysis{},
		OverallRisk: apitype.FailureRisk{
			Level:   apitype.FailureRiskLevelNone,
			Reasons: []string{},
		},
		OpenBugs: jobRun.ProwJob.Bugs,
	}

	switch {

	// Return early if we see a large gap in the number of tests:
	// order matters, if we have 0 tests that ran && 0 tests that failed we
	// want to compare that here before the 'no test failures' case
	case jobRunTestCount < (int(float64(historicalRunTestCount) * .75)):
		response.OverallRisk.Level = apitype.FailureRiskLevelIncomplete
		response.OverallRisk.Reasons = append(response.OverallRisk.Reasons,
			fmt.Sprintf("Tests for this run (%d) are below the historical average (%d): Incomplete", jobRunTestCount, historicalRunTestCount))
		return response, nil

	// Return early if no tests failed in this run:
	case len(jobRun.Tests) == 0:
		response.OverallRisk.Level = apitype.FailureRiskLevelNone
		response.OverallRisk.Reasons = append(response.OverallRisk.Reasons,
			"No test failures found in this job run.")
		return response, nil

	// Return early if we see mass test failures:
	case len(jobRun.Tests) > maxFailuresToFullyAnalyze:
		response.OverallRisk.Level = apitype.FailureRiskLevelHigh
		response.OverallRisk.Reasons = append(response.OverallRisk.Reasons,
			fmt.Sprintf("%d tests failed in this run: High", len(jobRun.Tests)))
		return response, nil
	}

	var maxTestRiskReason string

	// Iterate each failed test, query it's pass rates by NURPs, find a matching variant combo, and
	// see how often we've passed in the last week.
	for _, ft := range jobRun.Tests {

		if ft.Test.Name == testidentification.OpenShiftTestsName || testidentification.IsIgnoredTest(ft.Test.Name) {
			continue
		}

		logger.WithFields(log.Fields{
			"name": ft.Test.Name,
		}).Debug("failed test")

		testResult, err := testResultsFunc(
			ft.Test.Name, compareRelease, ft.Suite.Name, jobRun.ProwJob.Variants)
		if err != nil {
			return response, err
		}
		// Watch out for tests that ran in previous period, but not current, no sense comparing to 0 runs:
		if testResult != nil && testResult.CurrentRuns > 0 {
			testRiskLvl := getSeverityLevelForPassRate(testResult.CurrentPassPercentage)
			if testRiskLvl.Level >= response.OverallRisk.Level.Level {
				response.OverallRisk.Level = testRiskLvl
				maxTestRiskReason = fmt.Sprintf("Maximum failed test risk: %s", testRiskLvl.Name)
			}
			response.Tests = append(response.Tests, apitype.ProwJobRunTestRiskAnalysis{
				Name: testResult.Name,
				Risk: apitype.FailureRisk{
					Level: testRiskLvl,
					Reasons: []string{
						fmt.Sprintf("This test has passed %.2f%% of %d runs on release %s %v in the last week.",
							testResult.CurrentPassPercentage, testResult.CurrentRuns, compareRelease, testResult.Variants),
					},
				},
				OpenBugs: ft.Test.Bugs,
			})
		} else {
			testRiskLvl := apitype.FailureRiskLevelUnknown
			if testRiskLvl.Level >= response.OverallRisk.Level.Level {
				response.OverallRisk.Level = testRiskLvl
				maxTestRiskReason = fmt.Sprintf("Maximum failed test risk: %s", testRiskLvl.Name)
			}
			response.Tests = append(response.Tests, apitype.ProwJobRunTestRiskAnalysis{
				Name: ft.Test.Name,
				Risk: apitype.FailureRisk{
					Level: testRiskLvl,
					Reasons: []string{
						fmt.Sprintf("Unable to find matching test results for variants: %v",
							jobRun.ProwJob.Variants),
					},
				},
				OpenBugs: ft.Test.Bugs,
			})
		}
	}

	response.OverallRisk.Reasons = append(response.OverallRisk.Reasons, maxTestRiskReason)

	return response, nil
}

func getSeverityLevelForPassRate(passPercentage float64) apitype.RiskLevel {
	switch {
	case passPercentage >= 98.0:
		return apitype.FailureRiskLevelHigh
	case passPercentage >= 80:
		return apitype.FailureRiskLevelMedium
	case passPercentage < 80:
		return apitype.FailureRiskLevelLow
	}
	return apitype.FailureRiskLevelUnknown
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func stringSubSlicesEqual(a, b []string) bool {
	// we are going to check if b contains all the values in a
	if len(a) > len(b) {
		return false
	}
	for _, v := range a {
		found := false
		for _, s := range b {
			if v == s {
				found = true
			}
		}

		if !found {
			return false
		}
	}
	return true
}
