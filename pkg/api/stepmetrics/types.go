package stepmetrics

import (
	"math"

	sippyprocessingv1 "github.com/openshift/sippy/pkg/apis/sippyprocessing/v1"
)

type TrendTrajectory string

const (
	TrendTrajectoryUp   TrendTrajectory = "Increasing"
	TrendTrajectoryDown TrendTrajectory = "Decreasing"
	TrendTrajectoryFlat TrendTrajectory = "Flat"
)

const All string = "All"

type Trend struct {
	Trajectory TrendTrajectory               `json:"trajectory"`
	Delta      float64                       `json:"delta"`
	Current    sippyprocessingv1.StageResult `json:"current"`
	Previous   sippyprocessingv1.StageResult `json:"previous"`
}

type StepDetail struct {
	Name  string `json:"name"`
	Trend `json:"trend"`
}

type JobDetails struct {
	JobName string `json:"jobName"`
	MultistageDetails
}

type MultistageDetails struct {
	Name        string `json:"name"`
	Trend       `json:"trend"`
	StepDetails map[string]StepDetail `json:"stepDetails"`
}

type StepDetails struct {
	Name         string `json:"name"`
	Trend        `json:"trend"`
	ByMultistage map[string]StepDetail `json:"multistageDetails"`
}

type Response struct {
	JobDetails        map[string]JobDetails        `json:"jobDetails"`
	MultistageDetails map[string]MultistageDetails `json:"multistageDetails"`
	StepDetails       map[string]StepDetails       `json:"stepDetails"`
	Request           Request                      `json:"request"`
}

func newStepDetail(curr, prev sippyprocessingv1.StageResult) StepDetail {
	return StepDetail{
		Name:  curr.Name,
		Trend: newTrend(curr, prev),
	}
}

func newTrend(curr, prev sippyprocessingv1.StageResult) Trend {
	return Trend{
		Current:    curr,
		Previous:   prev,
		Trajectory: getTrajectory(curr, prev),
		Delta:      math.Abs(curr.PassPercentage - prev.PassPercentage),
	}
}

func getTrajectory(curr, prev sippyprocessingv1.StageResult) TrendTrajectory {
	if curr.PassPercentage == prev.PassPercentage {
		return TrendTrajectoryFlat
	} else if curr.PassPercentage > prev.PassPercentage {
		return TrendTrajectoryUp
	}

	return TrendTrajectoryDown
}
