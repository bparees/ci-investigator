package sippyserver

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/openshift/sippy/pkg/util"
)

func getISO8601Date(paramName string, req *http.Request) (*time.Time, error) {
	param := req.URL.Query().Get(paramName)
	if param == "" {
		return nil, nil
	}

	date, err := time.Parse("2006-01-02T15:04:05Z", param)
	if err != nil {
		return nil, err
	}

	return &date, nil
}

func getPeriodDates(defaultPeriod string, req *http.Request, timeNow time.Time) (start, boundary, end time.Time) {
	period := getPeriod(req, defaultPeriod)

	// If start, boundary, and end params are all specified, use those
	startp := getDateParam("start", req)
	boundaryp := getDateParam("boundary", req)
	endp := getDateParam("end", req)
	if startp != nil && boundaryp != nil && endp != nil {
		return *startp, *boundaryp, *endp
	}

	// Otherwise generate from the period name
	return util.PeriodToDates(period, timeNow)
}

func getDateParam(paramName string, req *http.Request) *time.Time {
	param := req.URL.Query().Get(paramName)
	if param != "" {
		t, err := time.Parse("2006-01-02", param)
		if err != nil {
			log.WithError(err).Warningf("error decoding %q param: %s", param, err.Error())
			return nil
		}
		return &t
	}

	return nil
}

func getPeriod(req *http.Request, defaultValue string) string {
	period := req.URL.Query().Get("period")
	if period == "" {
		return defaultValue
	}
	return period
}
