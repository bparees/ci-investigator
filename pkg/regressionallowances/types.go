package regressionallowances

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/openshift/sippy/pkg/apis/api"
)

type IntentionalRegression struct {
	JiraComponent             string
	TestID                    string
	TestName                  string
	Variant                   api.ComponentReportColumnIdentification
	PreviousPassPercentage    int
	PreviousSampleSize        int
	RegressedPassPercentage   int
	RegressedSampleSize       int
	ReasonToAllowInsteadOfFix string
}

func IntentionalRegressionFor(releaseString string, variant api.ComponentReportColumnIdentification, testID string) *IntentionalRegression {
	var targetMap map[regressionKey]IntentionalRegression
	switch release(releaseString) {
	case release415:
		targetMap = regressions_415
	case release416:
		targetMap = regressions_416
	default:
		return nil
	}

	inKey := keyFor(testID, variant)
	if t, ok := targetMap[inKey]; ok {
		log.Debugf("found approved regression: %+v", t)
		return &t
	}
	return nil
}

type release string

var (
	release415 release = "4.15"
	release416 release = "4.16"
)

var (
	regressions_415 = map[regressionKey]IntentionalRegression{}
	regressions_416 = map[regressionKey]IntentionalRegression{}
)

type regressionKey struct {
	testID  string
	variant api.ComponentReportColumnIdentification
}

func keyFor(testID string, variant api.ComponentReportColumnIdentification) regressionKey {
	return regressionKey{
		testID: testID,
		variant: api.ComponentReportColumnIdentification{
			Network:  variant.Network,
			Upgrade:  variant.Upgrade,
			Arch:     variant.Arch,
			Platform: variant.Platform,
		},
	}
}

func mustAddIntentionalRegression(release release, in IntentionalRegression) {
	if err := addIntentionalRegression(release, in); err != nil {
		panic(err)
	}
}

func addIntentionalRegression(release release, in IntentionalRegression) error {
	if len(in.JiraComponent) == 0 {
		return fmt.Errorf("jiraComponent must be specified")
	}
	if len(in.TestID) == 0 {
		return fmt.Errorf("testID must be specified")
	}
	if len(in.TestName) == 0 {
		return fmt.Errorf("testName must be specified")
	}
	if in.PreviousPassPercentage <= 0 {
		return fmt.Errorf("previousPassPercentage must be specified")
	}
	if in.RegressedPassPercentage <= 0 {
		return fmt.Errorf("regressedPassPercentage must be specified")
	}
	if in.PreviousSampleSize <= 0 {
		return fmt.Errorf("previousSampleSize must be specified")
	}
	if in.RegressedSampleSize <= 0 {
		return fmt.Errorf("regressedSampleSize must be specified")
	}
	if len(in.ReasonToAllowInsteadOfFix) == 0 {
		return fmt.Errorf("reasonToAllowInsteadOfFix must be specified")
	}
	if len(in.Variant.Network) == 0 {
		return fmt.Errorf("network must be specified")
	}
	if len(in.Variant.Arch) == 0 {
		return fmt.Errorf("arch must be specified")
	}
	if len(in.Variant.Platform) == 0 {
		return fmt.Errorf("platform must be specified")
	}
	if len(in.Variant.Upgrade) == 0 {
		return fmt.Errorf("upgrade must be specified")
	}

	var targetMap map[regressionKey]IntentionalRegression

	switch release {
	case release415:
		targetMap = regressions_415
	case release416:
		targetMap = regressions_416
	default:
		return fmt.Errorf("unknown release: %q", release)
	}

	inKey := keyFor(in.TestID, in.Variant)
	if _, ok := targetMap[inKey]; ok {
		return fmt.Errorf("test %q was already added", in.TestID)
	}

	targetMap[inKey] = in

	return nil
}
