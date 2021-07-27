package testgridconversion_test

import (
	"fmt"
	"testing"

	testgridv1 "github.com/openshift/sippy/pkg/apis/testgrid/v1"
	"github.com/openshift/sippy/pkg/testgridanalysis/testgridanalysisapi"
	"github.com/openshift/sippy/pkg/testgridanalysis/testgridconversion"
	"github.com/openshift/sippy/pkg/util/sets"
)

// The TestGrid job name; hardcoded for simplicity.
const jobName string = "periodic-ci-openshift-release-master-nightly-4.9-e2e-aws"

const overallTestName string = "Overall"

// The missing overall error text
const missingOverallErrText string = "missing Overall test in job " + jobName

// For simplicity, we assume a single test run on a single day.
// Only the RunLength test uses a different value
const numOfJobs int = 1

// Comprises test cases which test that a given set of input testgrid names
// cause a certain set of output tests to be present or omitted. This allows
// easy reuse of the main test loop and allows additional checks to be added
// onto it for specific edge-cases not covered by the main test loop.
type rawDataTestCase struct {
	// Test names as they appear in testgrid
	testGridTestNames []string
	// What test names we expect to find
	expectedTestNames []string
	// Additional (optional) tests / checks to run
	testFunc func(testFuncOpts)
	// What conversion options we should use
	options testgridconversion.ProcessingOptions
}

type testFuncOpts struct {
	// The resulting RawData
	rawData testgridanalysisapi.RawData
	// Any warnings generated by the conversion process
	warnings []string
	// Errors returned by the conversion process
	errs []error
	// The test case used to run the code under test
	testCase rawDataTestCase
	// The testgrid status of the generated input data
	testGridStatus testgridv1.TestStatus
}

func runRawDataTestCase(t *testing.T, rawDataTestCase rawDataTestCase) {
	// The testgrid test statuses we're concerned about
	statuses := []testgridv1.TestStatus{
		testgridv1.TestStatusSuccess,
		testgridv1.TestStatusFailure,
		testgridv1.TestStatusFlake,
	}

	for _, status := range statuses {
		// Create a sub-test for each testgrid test status we're interested in.
		// This is done to easily increase test coverage.
		t.Run(getStatusName(status), func(t *testing.T) {
			// Get our input data
			testGridJobDetails := getTestGridJobDetailsFromTestNames(rawDataTestCase.testGridTestNames, status)

			// Check our inputs
			assertTestGridJobDetailsOK(t, testGridJobDetails, numOfJobs)

			// Run the processing code (the code under test) using the provided options
			rawData, warnings, errs := rawDataTestCase.options.ProcessTestGridDataIntoRawJobResults(testGridJobDetails)

			// If the provided test names do not include Overall, we should expect errors.
			if !sets.NewString(rawDataTestCase.testGridTestNames...).Has(overallTestName) {
				assertMissingOverallErrors(t, errs)
			} else {
				assertNoErrors(t, errs)
			}

			// Get the job results of our sole job name
			jobResults := rawData.JobResults[jobName]

			// Check that we have the expected test names
			// Note: We don't verify their results as this is something best handled
			// by the optional verification function.
			assertRawTestResultsNamesEqual(t, jobResults, rawDataTestCase.expectedTestNames)

			// If there are no expected test names, there are no changelists.
			if len(rawDataTestCase.expectedTestNames) > 0 {
				assertChangelistsEqual(t, jobResults, testGridJobDetails[0].ChangeLists)
			} else {
				assertChangelistsEqual(t, jobResults, []string{})
			}

			// Run the optional verifications, if present.
			if rawDataTestCase.testFunc != nil {
				rawDataTestCase.testFunc(testFuncOpts{
					errs:           errs,
					rawData:        rawData,
					warnings:       warnings,
					testCase:       rawDataTestCase,
					testGridStatus: status,
				})
			}
		})
	}
}

func TestToRawDataNamesWithPrefixes(t *testing.T) {
	testCases := []struct {
		testGridName string
		expectedName string
	}{
		// openshift-tests.
		{
			testGridName: "openshift-tests.[sig-arch] Managed cluster should ensure platform components have system-* priority class associated [Suite:openshift/conformance/parallel]",
			expectedName: "[sig-arch] Managed cluster should ensure platform components have system-* priority class associated [Suite:openshift/conformance/parallel]",
		},
		// operator conditions
		{
			testGridName: "Operator results.operator conditions authentication",
			expectedName: "operator conditions authentication",
		},
		// OSD e2e suite
		{
			testGridName: "OSD e2e suite.[install] [Suite: operators] [OSD] Custom Domains Operator Should allow dedicated-admins to create domains Should be resolvable by external services",
			expectedName: "[install] [Suite: operators] [OSD] Custom Domains Operator Should allow dedicated-admins to create domains Should be resolvable by external services",
		},
		// Cluster upgrade suite
		{
			testGridName: "Cluster upgrade.[sig-apps] daemonset-upgrade",
			expectedName: "[sig-apps] daemonset-upgrade",
		},
	}

	testFunc := func(opts testFuncOpts) {
		// We should have no warnings
		assertNoWarnings(t, opts.warnings)

		// We only have a single job, so do a single lookup
		jobResult := opts.rawData.JobResults[jobName]

		// Check that we have the expected test name
		assertHasRawTestResults(t, jobResult, opts.testCase.expectedTestNames)

		// Check that we don't have the original test name
		assertNotHasRawTestResults(t, jobResult, opts.testCase.testGridTestNames[0:1])
	}

	for _, testCase := range testCases {
		t.Run(testCase.testGridName, func(t *testing.T) {
			runRawDataTestCase(t, rawDataTestCase{
				testGridTestNames: []string{testCase.testGridName, overallTestName},
				expectedTestNames: []string{testCase.expectedName, overallTestName},
				testFunc:          testFunc,
				options: testgridconversion.ProcessingOptions{
					StartDay:             0,
					NumDays:              numOfJobs,
					SyntheticTestManager: testgridconversion.NewEmptySyntheticTestManager(),
				},
			})
		})
	}
}

func TestToRawDataOverall(t *testing.T) {
	// The name "Overall" has special meaning within the testgrid conversion code.
	// Specifically, it sets certain properties on the jobrunresult that indicate whether the job succeeded or failed.
	testName := overallTestName

	testFunc := func(opts testFuncOpts) {
		// This test name should not be an ignored test.
		if !testgridconversion.IsIgnoredTest(testName) {
			t.Errorf("expected %s not to be ignored", testName)
		}

		rawJobRunResult := opts.rawData.JobResults[jobName].JobRunResults[getProwURL("0123456789")]

		if opts.testGridStatus == testgridv1.TestStatusSuccess || opts.testGridStatus == testgridv1.TestStatusFlake {
			// Check that the job was marked as a success.
			if !rawJobRunResult.Succeeded {
				t.Errorf("expected job to succeed")
			}
		}

		if opts.testGridStatus == testgridv1.TestStatusFailure {
			// Check that the job was marked as a failure.
			if !rawJobRunResult.Failed {
				t.Errorf("expected job to succeed")
			}
		}
	}

	runRawDataTestCase(t, rawDataTestCase{
		testGridTestNames: []string{testName},
		expectedTestNames: []string{testName},
		options: testgridconversion.ProcessingOptions{
			StartDay:             0,
			NumDays:              numOfJobs,
			SyntheticTestManager: testgridconversion.NewEmptySyntheticTestManager(),
		},
		testFunc: testFunc,
	})
}

func TestToRawDataStepRegistryItem(t *testing.T) {
	testCase := rawDataTestCase{
		testGridTestNames: []string{
			"operator.Run multi-stage test openshift-ipi-azure-arcconformance",
			"operator.Run multi-stage test openshift-ipi-azure-arcconformance - openshift-ipi-azure-arcconformance-ipi-install-rbac container test",
			"operator.Run multi-stage test openshift-ipi-azure-arcconformance - openshift-ipi-azure-arcconformance-ipi-install-times-collection container test",
		},
		expectedTestNames: []string{
			// TODO: Determine if these tests should still be expected to be here
			// since they were previously ignored.
			"operator.Run multi-stage test openshift-ipi-azure-arcconformance",
			"operator.Run multi-stage test openshift-ipi-azure-arcconformance - openshift-ipi-azure-arcconformance-ipi-install-rbac container test",
			"operator.Run multi-stage test openshift-ipi-azure-arcconformance - openshift-ipi-azure-arcconformance-ipi-install-times-collection container test",
		},
		options: testgridconversion.ProcessingOptions{
			StartDay:             0,
			NumDays:              numOfJobs,
			SyntheticTestManager: testgridconversion.NewEmptySyntheticTestManager(),
		},
		testFunc: func(testOpts testFuncOpts) {
			expectedState := ""

			// If a success or flake occurs, we expect the step registry item to be
			// in a successful state.
			if testOpts.testGridStatus == testgridv1.TestStatusSuccess || testOpts.testGridStatus == testgridv1.TestStatusFlake {
				expectedState = testgridanalysisapi.Success
			}

			// If a failure occurs, we expect the step registry item to be
			// in a failure state.
			if testOpts.testGridStatus == testgridv1.TestStatusFailure {
				expectedState = testgridanalysisapi.Failure
			}

			// Regardless of state, we expect the names to be equal.
			expectedStepRegistryItemStates := testgridanalysisapi.StepRegistryItemStates{
				MultistageName: "openshift-ipi-azure-arcconformance",
				MultistageState: testgridanalysisapi.StageState{
					Name:  "openshift-ipi-azure-arcconformance",
					State: expectedState,
				},
				States: []testgridanalysisapi.StageState{
					{
						Name:             "ipi-install-rbac",
						State:            expectedState,
						OriginalTestName: "operator.Run multi-stage test openshift-ipi-azure-arcconformance - openshift-ipi-azure-arcconformance-ipi-install-rbac container test",
					},
					{
						Name:             "ipi-install-times-collection",
						State:            expectedState,
						OriginalTestName: "operator.Run multi-stage test openshift-ipi-azure-arcconformance - openshift-ipi-azure-arcconformance-ipi-install-times-collection container test",
					},
				},
			}

			jrr := testOpts.rawData.JobResults[jobName].JobRunResults[getProwURL("0123456789")]

			assertStepRegistryItemStatesEqual(t, jrr.StepRegistryItemStates, expectedStepRegistryItemStates)
		},
	}

	runRawDataTestCase(t, testCase)
}

func TestToRawDataSkippedNames(t *testing.T) {
	// These testgrid test names should be ignored
	ignoredTestNames := []string{
		"job.initialize",
		"openshift-tests.[sig-arch] Monitor cluster while tests execute",
		"openshift-tests.[sig-arch][Feature:ClusterUpgrade] Cluster should remain functional during upgrade [Disruptive] [Serial]",
		"operator.Import a release payload",
		"operator.Import the release payload \"intermediate\" from an external source",
		"operator.Import the release payload",
	}

	// These test names should not be ignored
	unignoredTestNames := []string{
		"nonignored test name",
	}

	options := testgridconversion.ProcessingOptions{
		StartDay:             0,
		NumDays:              numOfJobs,
		SyntheticTestManager: testgridconversion.NewEmptySyntheticTestManager(),
	}

	testFunc := func(opts testFuncOpts) {
		// We should have no warnings.
		assertNoWarnings(t, opts.warnings)
	}

	for _, testName := range unignoredTestNames {
		// Check that unignored test names are, indeed, not ignored.
		t.Run("UnignoredRegex"+testName, func(t *testing.T) {
			if testgridconversion.IsIgnoredTest(testName) {
				t.Errorf("expected %s not to be ignored", testName)
			}
		})

		// Run the raw data test case to verify that non-ignored tests are found
		t.Run("Unignored", func(t *testing.T) {
			runRawDataTestCase(t, rawDataTestCase{
				testGridTestNames: []string{testName, overallTestName},
				expectedTestNames: []string{testName, overallTestName},
				options:           options,
				testFunc:          testFunc,
			})
		})
	}

	for _, testName := range ignoredTestNames {
		// Create a sub-test per ignored test name
		t.Run(testName, func(t *testing.T) {
			// Check that the regex matches the ignored test name
			t.Run("Regex", func(t *testing.T) {
				if !testgridconversion.IsIgnoredTest(testName) {
					t.Errorf("expected %s to be ignored", testName)
				}
			})

			// Run the raw data test case to verify that no unexpected tests are found
			runRawDataTestCase(t, rawDataTestCase{
				testGridTestNames: []string{testName, overallTestName},
				expectedTestNames: []string{overallTestName},
				options:           options,
				testFunc:          testFunc,
			})
		})
	}
}

func TestToRawDataSynthetics(t *testing.T) {
	// These test cases are intended to test the interaction of ToRawData with
	// the Openshift Synthetic Test Manager.

	// These will be appended to the expected test names for each test case below
	// where "expectSyntheticTestNames" is set to true.
	sippySyntheticTests := []string{
		"[sig-sippy] infrastructure should work",
		"[sig-sippy] install should not timeout",
		"[sig-sippy] install should work",
		"[sig-sippy] openshift-tests should work",
		"[sig-sippy] tests should finish with healthy operators",
	}

	missingSetupJobWarning := []string{
		fmt.Sprintf("\"%s\" is missing a test setup job to indicate successful installs", jobName),
	}

	testCases := []struct {
		name              string
		testGridTestNames []string
		expectedTestNames []string
		// These are grouped by test status because certain test statuses (success,
		// failure, flake) will cause the OpenshiftSyntheticTestManager emit a
		// warning under specific cases.
		expectedWarnings map[testgridv1.TestStatus][]string
		expectedError    error
		// When true, we expect the test names in sippySyntheticTests to be present
		// in addition to the expected test names within the testcase.
		expectSyntheticTestNames bool
	}{
		// A job with no synthetic tests should still have synthetic test outcomes
		// when the synthetic test manager is used.
		{
			name: "no overall test present",
			testGridTestNames: []string{
				"not a synthetic test",
			},
			// We expect no test names since we do not have
			expectedTestNames: []string{},
			// We expect no warnings since we do not have an "Overall" test in this
			// case, so the synthetic test manager does not run.
			expectedWarnings: map[testgridv1.TestStatus][]string{},
			// However, we expect an error because we do not have an "Overall"
			// test.
			expectedError:            fmt.Errorf(missingOverallErrText),
			expectSyntheticTestNames: false,
		},
		{
			name: "operator upgrade",
			testGridTestNames: []string{
				"container setup",
				"operator install authentication",
				"operator install machine-config-operator",
				"openshift-tests.[bz-Machine Config Operator] clusteroperator/machine-config should not change condition/Progressing",
				overallTestName,
			},
			expectedTestNames: []string{
				"[sig-sippy] tests should finish with healthy operators",
				"container setup",
				"operator conditions  authentication",
				"operator conditions  machine-config-operator",
				"operator install authentication",
				"operator install machine-config-operator",
				"[bz-Machine Config Operator] clusteroperator/machine-config should not change condition/Progressing",
				overallTestName,
			},
			expectSyntheticTestNames: true,
		},
		{
			name: "cluster upgrade",
			testGridTestNames: []string{
				"Cluster upgrade.[sig-cluster-lifecycle] Cluster completes upgrade",
				"Cluster upgrade.[sig-cluster-lifecycle] Cluster version operator acknowledges upgrade",
				"Cluster upgrade.[sig-mco] Machine config pools complete upgrade",
				"container setup",
				overallTestName,
			},
			expectedTestNames: []string{
				"[sig-cluster-lifecycle] Cluster completes upgrade",
				"[sig-cluster-lifecycle] Cluster version operator acknowledges upgrade",
				"[sig-mco] Machine config pools complete upgrade",
				"[sig-sippy] upgrade should work",
				"container setup",
				overallTestName,
			},
			expectSyntheticTestNames: true,
		},
		// Overall has special meaning within the OpenShift Synthetic Test Manager
		// as well as within the conversion code.
		{
			name: overallTestName,
			testGridTestNames: []string{
				overallTestName,
			},
			expectedTestNames: []string{overallTestName},
			expectedWarnings: map[testgridv1.TestStatus][]string{
				testgridv1.TestStatusFailure: missingSetupJobWarning,
			},
			expectSyntheticTestNames: true,
		},
	}

	options := testgridconversion.ProcessingOptions{
		StartDay:             0,
		NumDays:              numOfJobs,
		SyntheticTestManager: testgridconversion.NewOpenshiftSyntheticTestManager(),
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			expectedTestNames := testCase.expectedTestNames
			if testCase.expectSyntheticTestNames {
				// Add the synthetic test names onto our expected test names, if needed
				expectedTestNames = append(expectedTestNames, sippySyntheticTests...)
			}

			// Convert each test case into a RawDataTestCase and run it
			runRawDataTestCase(t, rawDataTestCase{
				testGridTestNames: testCase.testGridTestNames,
				expectedTestNames: expectedTestNames,
				options:           options,
				testFunc: func(opts testFuncOpts) {
					// This func references the original test case, so must be defined within the test loop.

					// Ensure the warnings are what we expect
					assertWarningsEqual(t, opts.warnings, testCase.expectedWarnings[opts.testGridStatus])

					// Check for errors
					if testCase.expectedError != nil {
						for _, err := range opts.errs {
							assertErrorEqual(t, err, testCase.expectedError)
							assertMissingOverallError(t, err)
						}
					} else {
						assertNoErrors(t, opts.errs)
					}
				},
			})
		})
	}
}

func TestToRawDataSkippedJob(t *testing.T) {
	testNames := []string{
		"test-1",
		"test-2",
	}

	testGridJobDetails := getTestGridJobDetailsForSkipped(testNames)

	skippedJobName := testGridJobDetails[1].Name

	// Check our inputs
	for _, jobDetail := range testGridJobDetails {
		// We examine each of these individually because assertTestGridJobDetailsOK
		// only assumes a single JobDetail entry.
		assertTestGridJobDetailsOK(t, []testgridv1.JobDetails{jobDetail}, len(jobDetail.ChangeLists))
	}

	options := testgridconversion.ProcessingOptions{
		StartDay:             0,
		NumDays:              numOfJobs,
		SyntheticTestManager: testgridconversion.NewEmptySyntheticTestManager(),
	}

	// Run the processing code (the code under test)
	rawData, warnings, errs := options.ProcessTestGridDataIntoRawJobResults(testGridJobDetails)

	// We expect to get errors back
	assertErrorEqual(t, errs[0], fmt.Errorf("missing Overall test in job %s", skippedJobName))

	// Check that we have the correct job names
	actualJobNames := sets.StringKeySet(rawData.JobResults)
	if actualJobNames.Has(skippedJobName) {
		t.Errorf("expected not to find job with name %s", skippedJobName)
	}

	if !actualJobNames.Has(jobName) {
		t.Errorf("expected to find job with name %s", jobName)
	}

	// We should have no warnings since we're not concerned with synthetic tests.
	assertNoWarnings(t, warnings)

	jobResult := rawData.JobResults[jobName]

	// Check that the failing test attached to the skipped job is not present
	assertNotHasRawTestResults(t, rawData.JobResults[jobName], []string{"failing-test"})
	assertNotHasRawTestResults(t, rawData.JobResults[skippedJobName], []string{"failing-test"})

	// Check that the test results are exactly what we expect
	assertRawTestResultsEqual(t, jobResult, []testgridanalysisapi.RawTestResult{
		{
			Name:       overallTestName,
			Successes:  1,
			Timestamps: testGridJobDetails[0].Timestamps,
		},
		{
			Name:       "test-1",
			Successes:  1,
			Timestamps: testGridJobDetails[0].Timestamps,
		},
		{
			Name:       "test-2",
			Successes:  1,
			Timestamps: testGridJobDetails[0].Timestamps,
		},
	})
}

func TestToRawDataRunLengthEncoding(t *testing.T) {
	// This test is intended to test the runlength encoding input from TestGrid
	// as well as the numdays mechanism.
	// Amongst other things, this test verifies that given a certain number of
	// testgrid test statuses, that a specific rawtestresult will be generated.
	testName := "test-1"

	// We define a set of changelists which we will use across all test cases.
	changelists := []string{
		"123",
		"456",
		"789",
		"987",
		"654",
		"321",
		"ABC",
		"DEF",
	}

	testCases := []struct {
		// The name of the testcase for output purposes
		name string
		// Processing options for conversion code
		options testgridconversion.ProcessingOptions
		// The expected raw test results
		expectedRawTestResults []testgridanalysisapi.RawTestResult
		// List of changelists which are expected to be failures.
		failingChangelists []string
		// List of changelists which are expected to be successes.
		passingChangelists []string
		// List of changelists which should not appear in the output given the
		// startday / numday configuration.
		excludedChangelists []string
		// Note: Changelist lists within the test-case are unordered and no order is assumed.
	}{
		{
			name: "minus one both ends",
			options: testgridconversion.ProcessingOptions{
				StartDay: -1,
				NumDays:  6,
			},
			failingChangelists: []string{
				"456",
				"789",
			},
			passingChangelists: []string{
				"987",
				"654",
				"321",
				"123",
			},
			excludedChangelists: []string{
				"ABC",
				"DEF",
			},
			expectedRawTestResults: []testgridanalysisapi.RawTestResult{
				{
					Name: testName,
					// Successes include the flake count
					Successes:  4,
					Failures:   2,
					Flakes:     2,
					Timestamps: []int{1},
				},
				{
					Name:       overallTestName,
					Successes:  4,
					Failures:   2,
					Flakes:     2,
					Timestamps: []int{1},
				},
			},
		},
		{
			name: "from zero",
			options: testgridconversion.ProcessingOptions{
				StartDay: 0,
				NumDays:  8,
			},
			failingChangelists: []string{
				"456",
				"789",
			},
			passingChangelists: []string{
				"123",
				"987",
				"654",
				"321",
				"ABC",
				"DEF",
			},
			excludedChangelists: []string{},
			expectedRawTestResults: []testgridanalysisapi.RawTestResult{
				{
					Name:      testName,
					Successes: 6,
					Failures:  2,
					Flakes:    3,
					Timestamps: []int{
						1,
					},
				},
				{
					Name:      overallTestName,
					Successes: 6,
					Failures:  2,
					Flakes:    3,
					Timestamps: []int{
						1,
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Create our input test with statuses run-length encoded with respect to
			// the timestamp and changelist.
			//
			// Assuming the day window accommodates all the test statuses (as it does
			// in the "from zero" test case above), the changelists should align thusly:
			// 123 - success
			// 456 - failure
			// 789 - failure
			// 987 - flake
			// 654 - flake
			// 321 - success
			// ABC - success
			// DEF - flake
			testGridTests := []testgridv1.Test{
				{
					Name: testName,
					Statuses: []testgridv1.TestResult{
						{
							Count: 1,
							Value: testgridv1.TestStatusSuccess,
						},
						{
							Count: 2,
							Value: testgridv1.TestStatusFailure,
						},
						{
							Count: 2,
							Value: testgridv1.TestStatusFlake,
						},
						{
							Count: 2,
							Value: testgridv1.TestStatusSuccess,
						},
						{
							Count: 1,
							Value: testgridv1.TestStatusFlake,
						},
					},
				},
				{
					Name: overallTestName,
					Statuses: []testgridv1.TestResult{
						{
							Count: 1,
							Value: testgridv1.TestStatusSuccess,
						},
						{
							Count: 2,
							Value: testgridv1.TestStatusFailure,
						},
						{
							Count: 2,
							Value: testgridv1.TestStatusFlake,
						},
						{
							Count: 2,
							Value: testgridv1.TestStatusSuccess,
						},
						{
							Count: 1,
							Value: testgridv1.TestStatusFlake,
						},
					},
				},
			}

			// Get our input data
			testGridJobDetails := getTestGridJobDetailsForRunLength(testGridTests, changelists)

			// Copy the timestamps onto the expected test results
			for i := range testCase.expectedRawTestResults {
				testCase.expectedRawTestResults[i].Timestamps = testGridJobDetails[0].Timestamps
			}

			// Check our inputs
			assertTestGridJobDetailsOK(t, testGridJobDetails, len(changelists))

			// This test isn't concerned about synthetics
			testCase.options.SyntheticTestManager = testgridconversion.NewEmptySyntheticTestManager()

			// Run the processing code (the code under test)
			rawData, warnings, errs := testCase.options.ProcessTestGridDataIntoRawJobResults(testGridJobDetails)

			// We should get no warnings since we're using the empty synthetic test manager.
			assertNoWarnings(t, warnings)

			// We should get no errors because we have an Overall test
			assertNoErrors(t, errs)

			// We have a singular job, so only look this up once.
			jobResults := rawData.JobResults[jobName]

			// Check that the test results match what we expect
			assertRawTestResultsEqual(t, jobResults, testCase.expectedRawTestResults)

			// Check our passing changelists
			assertHasChangelists(t, jobResults, testCase.passingChangelists)
			for _, changelist := range testCase.passingChangelists {
				jrr := jobResults.JobRunResults[getProwURL(changelist)]

				if hasTestFailures(jrr) {
					t.Errorf("expected no test failures for changelist %s", changelist)
				}

				if hasFailedTest(jrr, testName) {
					t.Errorf("expected not to find failing test: %s in changelist %s", testName, changelist)
				}
			}

			// Check our failing changelists
			assertHasChangelists(t, jobResults, testCase.failingChangelists)
			for _, changelist := range testCase.failingChangelists {
				jrr := jobResults.JobRunResults[getProwURL(changelist)]

				if !hasTestFailures(jrr) {
					t.Errorf("expected test failure for changelist %s", changelist)
				}

				if !hasFailedTest(jrr, testName) {
					t.Errorf("expected to find failing test: %s in changelist %s", testName, changelist)
				}
			}

			// Check our excluded changelists
			assertNotHasChangelists(t, jobResults, testCase.excludedChangelists)

			// Ensure we didn't miss anything
			assertChangelistsEqual(t, jobResults, append(testCase.passingChangelists, testCase.failingChangelists...))
		})
	}
}

func assertStepRegistryItemStatesEqual(t *testing.T, have, want testgridanalysisapi.StepRegistryItemStates) {
	if have.MultistageName != want.MultistageName {
		t.Errorf("expected names to be equal, want: %s, got: %s", want.MultistageName, have.MultistageName)
	}

	if have.MultistageState != want.MultistageState {
		t.Errorf("expected multistage state to be equal, want: %v, got: %v", want.MultistageState, have.MultistageState)
	}

	if len(have.States) != len(want.States) {
		t.Errorf("mismatched stepmetrics, want: %d, got: %v", len(want.States), len(have.States))
	}

	for i := range have.States {
		if have.States[i] != want.States[i] {
			t.Errorf("want stepmetric: %v, got: %v", want.States[i], have.States[i])
		}
	}
}
