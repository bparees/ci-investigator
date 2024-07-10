package api

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/pkg/errors"

	bqclient "github.com/openshift/sippy/pkg/bigquery"
	"github.com/openshift/sippy/pkg/db"
	"github.com/openshift/sippy/pkg/db/query"
	log "github.com/sirupsen/logrus"

	"github.com/openshift/sippy/pkg/apis/cache"
)

var (
	defaultCacheDuration = 8 * time.Hour
)

const releasePresubmits = "Presubmits"

type CacheData struct {
	cacheKey func() ([]byte, error)
}

func (c *CacheData) GetCacheKey() ([]byte, error) {
	if c.cacheKey == nil {
		panic("cache key is nil")
	}
	return c.cacheKey()
}

func GetPrefixedCacheKey(prefix string, cacheKey interface{}) CacheData {
	if isStructWithNoPublicFields(cacheKey) {
		panic(fmt.Sprintf("you cannot use struct %s with no exported fields as a cache key", reflect.TypeOf(cacheKey)))
	}

	return CacheData{cacheKey: func() ([]byte, error) {
		b, err := json.Marshal(cacheKey)
		if err != nil {
			return nil, err
		}
		if len(prefix) > 0 {
			return append([]byte(prefix), b...), nil
		}
		return b, nil
	}}
}

// getDataFromCacheOrGenerate attempts to find a cached record otherwise generates new data.
func getDataFromCacheOrGenerate[T any](c cache.Cache, cacheOptions cache.RequestOptions, cacheData CacheData, generateFn func() (T, []error), defaultVal T) (T, []error) {
	if c != nil {
		cacheKey, err := cacheData.GetCacheKey()
		if err != nil {
			return defaultVal, []error{err}
		}

		// If someone is giving us an uncacheable cacheKey, we should panic so it gets detected in testing
		if len(cacheKey) == 0 {
			panic(fmt.Sprintf("cache key is empty for %s", reflect.TypeOf(defaultVal)))
		}

		if !cacheOptions.ForceRefresh {
			if res, err := c.Get(string(cacheKey)); err == nil {
				log.WithFields(log.Fields{
					"key":  string(cacheKey),
					"type": reflect.TypeOf(defaultVal).String(),
				}).Debugf("cache hit")
				var cr T
				if err := json.Unmarshal(res, &cr); err != nil {
					return defaultVal, []error{errors.WithMessagef(err, "failed to unmarshal cached item.  cacheKey=%+v", cacheKey)}
				}
				return cr, nil
			}
			log.Infof("cache miss for cache key: %s", string(cacheKey))
		}
		result, errs := generateFn()
		if len(errs) == 0 {
			cr, err := json.Marshal(result)
			if err == nil {
				cacheDuration := defaultCacheDuration
				if cacheOptions.CRTimeRoundingFactor > 0 {
					now := time.Now().UTC()
					// Only cache until the next rounding duration
					cacheDuration = now.Truncate(cacheOptions.CRTimeRoundingFactor).Add(cacheOptions.CRTimeRoundingFactor).Sub(now)
				}
				if err := c.Set(string(cacheKey), cr, cacheDuration); err != nil {
					log.WithError(err).Warningf("couldn't persist new item to cache")
				} else {
					log.Debugf("cache set for cache key: %s", string(cacheKey))
				}
			}
		}
		return result, errs
	}

	return generateFn()
}

// isStructWithNoPublicFields checks if the given interface is a struct with no public fields.
func isStructWithNoPublicFields(v interface{}) bool {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < val.NumField(); i++ {
		if val.Type().Field(i).IsExported() {
			return false
		}
	}
	return true
}

// GetReleases gets all the releases defined in the BQ Releases table if bqc is defined.
// Otherwise, it falls back to get it from sippy DB
func GetReleases(dbc *db.DB, bqc *bqclient.Client) ([]query.Release, error) {
	if bqc != nil {
		releases, err := GetReleasesFromBigQuery(bqc)
		if err != nil {
			log.WithError(err).Error("error getting releases from bigquery")
			return releases, err
		}
		// Add special release Presubmits for prow jobs
		releases = append(releases, query.Release{Release: releasePresubmits})
		return releases, nil
	}
	return query.ReleasesFromDB(dbc)
}
