package testgridconversion

import (
	"fmt"

	"github.com/openshift/sippy/pkg/testgridanalysis/testgridanalysisapi"
	"github.com/openshift/sippy/pkg/util/sets"
)

// createSyntheticTests takes the JobRunResult information and produces some pre-analysis by interpreting different types of failures
// and potentially producing synthentic test results and aggregations to better inform sippy.
// This needs to be called after all the JobDetails have been processed.
// returns warnings found in the data. Not failures to process it.
func createSyntheticTests(rawJobResults testgridanalysisapi.RawData) []string {
	warnings := []string{}

	// make a pass to fill in install, upgrade, and infra synthentic tests.
	type synthenticTestResult struct {
		name string
		pass int
		fail int
	}

	for jobName, jobResults := range rawJobResults.JobResults {
		numRunsWithoutSetup := 0
		for jrrKey, jrr := range jobResults.JobRunResults {
			if jrr.SetupStatus == "" {
				numRunsWithoutSetup++
			}

			syntheticTests := map[string]*synthenticTestResult{
				testgridanalysisapi.InstallTestName:             &synthenticTestResult{name: testgridanalysisapi.InstallTestName},
				testgridanalysisapi.InstallTimeoutTestName:      &synthenticTestResult{name: testgridanalysisapi.InstallTestName},
				testgridanalysisapi.InfrastructureTestName:      &synthenticTestResult{name: testgridanalysisapi.InfrastructureTestName},
				testgridanalysisapi.FinalOperatorHealthTestName: &synthenticTestResult{name: testgridanalysisapi.FinalOperatorHealthTestName},
			}
			// upgrades should only be indicated on jobs that run upgrades
			if jrr.UpgradeStarted {
				syntheticTests[testgridanalysisapi.UpgradeTestName] = &synthenticTestResult{name: testgridanalysisapi.UpgradeTestName}
			}

			hasFinalOperatorResults := len(jrr.FinalOperatorStates) > 0
			allOperatorsSuccessfulAtEndOfRun := true
			for _, operator := range jrr.FinalOperatorStates {
				if operator.State == testgridanalysisapi.Failure {
					allOperatorsSuccessfulAtEndOfRun = false
					break
				}
			}
			setupFailed := jrr.SetupStatus != testgridanalysisapi.Success
			setupSucceeded := jrr.SetupStatus == testgridanalysisapi.Success

			switch {
			case !hasFinalOperatorResults:
			// without results, there is no run for the tests
			case allOperatorsSuccessfulAtEndOfRun:
				syntheticTests[testgridanalysisapi.FinalOperatorHealthTestName].pass = 1
			default:
				syntheticTests[testgridanalysisapi.FinalOperatorHealthTestName].fail = 1
			}

			// set overall installed status
			switch {
			case setupSucceeded:
				// if setup succeeded, we are guaranteed that installation succeeded.
				syntheticTests[testgridanalysisapi.InstallTestName].pass = 1
				// if the test succeeded, then the operator install tests should all be passes
				for _, operatorState := range jrr.FinalOperatorStates {
					testName := testgridanalysisapi.OperatorInstallPrefix + " " + operatorState.Name
					syntheticTests[testName] = &synthenticTestResult{
						name: testName,
						pass: 1,
					}
				}

			case !hasFinalOperatorResults:
				// if we don't have any operator results, then don't count this an install one way or the other.  This was an infra failure

			default:
				// the setup failed and we have some operator results, which means the install started. This is a failure
				syntheticTests[testgridanalysisapi.InstallTestName].fail = 1

				// if the test failed, then the operator install tests should match the operator state
				for _, operatorState := range jrr.FinalOperatorStates {
					testName := testgridanalysisapi.OperatorInstallPrefix + " " + operatorState.Name
					syntheticTests[testName] = &synthenticTestResult{
						name: testName,
					}
					if operatorState.State == testgridanalysisapi.Success {
						syntheticTests[testName].pass = 1
					} else {
						syntheticTests[testName].fail = 1
					}
				}
			}

			// set overall install timeout status
			switch {
			case !setupSucceeded && hasFinalOperatorResults && allOperatorsSuccessfulAtEndOfRun:
				// the setup failed and yet all operators were successful in the end.  This means we had a weird problem.  Probably a timeout failure.
				syntheticTests[testgridanalysisapi.InstallTimeoutTestName].fail = 1

			default:
				syntheticTests[testgridanalysisapi.InstallTimeoutTestName].pass = 1

			}

			// set the infra status
			switch {
			case jobsWithKnownBadSetupContainer.Has(jobName):
				// do nothing.  If we don't have a setup container, we have no way of determining infrastructure

			case setupFailed && !hasFinalOperatorResults:
				// we only count failures as infra if we have no operator results.  If we got any operator working, then CI infra was working.
				syntheticTests[testgridanalysisapi.InfrastructureTestName].fail = 1

			default:
				syntheticTests[testgridanalysisapi.InfrastructureTestName].pass = 1
			}

			// set the update status
			switch {
			case setupFailed:
				// do nothing
			case !jrr.UpgradeStarted:
			// do nothing

			default:
				if jrr.UpgradeForOperatorsStatus == testgridanalysisapi.Success && jrr.UpgradeForMachineConfigPoolsStatus == testgridanalysisapi.Success {
					syntheticTests[testgridanalysisapi.UpgradeTestName].pass = 1
					// if the test succeeded, then the operator install tests should all be passes
					for _, operatorState := range jrr.FinalOperatorStates {
						testName := testgridanalysisapi.OperatorUpgradePrefix + " " + operatorState.Name
						syntheticTests[testName] = &synthenticTestResult{
							name: testName,
							pass: 1,
						}
					}

				} else {
					syntheticTests[testgridanalysisapi.UpgradeTestName].fail = 1
					// if the test failed, then the operator upgrade tests should match the operator state
					for _, operatorState := range jrr.FinalOperatorStates {
						testName := testgridanalysisapi.OperatorUpgradePrefix + " " + operatorState.Name
						syntheticTests[testName] = &synthenticTestResult{
							name: testName,
						}
						if operatorState.State == testgridanalysisapi.Success {
							syntheticTests[testName].pass = 1
						} else {
							syntheticTests[testName].fail = 1
						}
					}
				}
			}

			for testName, result := range syntheticTests {
				if result.fail > 0 {
					jrr.TestFailures += result.fail
					jrr.FailedTestNames = append(jrr.FailedTestNames, testName)
				}
				addTestResult(jobResults.TestResults, testName, result.pass, result.fail, 0)
			}

			jobResults.JobRunResults[jrrKey] = jrr
		}
		if float64(numRunsWithoutSetup)/float64(len(jobResults.JobRunResults)+1)*100 > 50 {
			if !jobsWithKnownBadSetupContainer.Has(jobName) {
				warnings = append(warnings, fmt.Sprintf("%q is missing a test setup job to indicate successful installs", jobName))
			}
		}

		rawJobResults.JobResults[jobName] = jobResults
	}
	return warnings
}

// this a list of jobs that either do not install the product (bug) or have never had a passing install.
// both should be fixed over time, but this reduces noise as we ratchet down.
var jobsWithKnownBadSetupContainer = sets.NewString(
	"promote-release-openshift-machine-os-content-e2e-aws-4.6",
	"promote-release-openshift-machine-os-content-e2e-aws-4.6-s390x",
	"promote-release-openshift-machine-os-content-e2e-aws-4.6-ppc64le",
	"release-openshift-origin-installer-e2e-aws-upgrade-rollback-4.5-to-4.6",
	"promote-release-openshift-machine-os-content-e2e-aws-4.7",
	"promote-release-openshift-machine-os-content-e2e-aws-4.7-s390x",
	"promote-release-openshift-machine-os-content-e2e-aws-4.7-ppc64le",
	"release-openshift-origin-installer-e2e-aws-upgrade-rollback-4.6-to-4.7",
	"periodic-ci-openshift-origin-release-3.11-e2e-gcp",
)
