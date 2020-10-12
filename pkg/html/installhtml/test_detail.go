package installhtml

import (
	"fmt"
	"strings"

	"github.com/openshift/sippy/pkg/util/sets"

	sippyprocessingv1 "github.com/openshift/sippy/pkg/apis/sippyprocessing/v1"
)

func testDetailTests(curr, prev sippyprocessingv1.TestReport, testSubstrings []string) string {
	dataForTestsByPlatform := getDataForTestsByPlatform(
		curr, prev,
		isTestDetailRelatedTest(testSubstrings),
		neverMatch,
	)

	platforms := sets.String{}
	for _, byPlatform := range dataForTestsByPlatform.testNameToPlatformToTestResult {
		platforms.Insert(sets.StringKeySet(byPlatform).UnsortedList()...)
	}

	return dataForTestsByPlatform.getTableHTML("Details for Tests", "TestDetailByPlatform", "Test Details by Platform", platforms.List(), getOperatorFromTest)
}

func summaryTestDetailRelatedTests(curr, prev sippyprocessingv1.TestReport, testSubstrings []string, numDays int, release string) string {
	// test name | test | pass rate | higher/lower | pass rate
	s := fmt.Sprintf(`
	<table class="table">
		<tr>
			<th colspan=5 class="text-center"><a class="text-dark" title="Tests, sorted by passing rate.  The link will prepopulate a BZ template to be filled out and submitted to report a test against the test." id="Tests" href="#Tests">Tests</a></th>
		</tr>
		<tr>
			<th colspan=2/><th class="text-center">Latest %d Days</th><th/><th class="text-center">Previous 7 Days</th>
		</tr>
		<tr>
			<th>Test Name</th><th>File a Test</th><th>Pass Rate</th><th/><th>Pass Rate</th>
		</tr>
	`, numDays)

	s += failingTestsRows(curr.ByTest, prev.ByTest, release, isTestDetailRelatedTest(testSubstrings))

	s = s + "</table>"

	return s
}

func isTestDetailRelatedTest(testSubstrings []string) func(sippyprocessingv1.TestResult) bool {
	return func(testResult sippyprocessingv1.TestResult) bool {
		for _, testSubString := range testSubstrings {
			if strings.Contains(testResult.Name, testSubString) {
				return true
			}
		}

		return false
	}
}
