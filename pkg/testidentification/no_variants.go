package testidentification

import (
	"github.com/openshift/sippy/pkg/db/models"
	"github.com/openshift/sippy/pkg/util/sets"
)

type noVariants struct{}

func NewEmptyVariantManager() VariantManager {
	return noVariants{}
}

func (noVariants) AllVariants() sets.String {
	return sets.String{}
}

func (v noVariants) IdentifyVariants(jobName, release string, jobType models.JobType) []string {
	return []string{}
}
func (noVariants) IsJobNeverStable(jobName string) bool {
	return false
}
