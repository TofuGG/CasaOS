package service

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS/model"
	"go.uber.org/zap"
)

const (
	// githubReleasesURL is the TofuGG fork's GitHub releases feed (NOT the
	// upstream api.casaos.io endpoint).
	githubReleasesURL = "https://api.github.com/repos/TofuGG/CasaOS/releases?per_page=5"
)

type CasaService interface {
	GetCasaosVersion() model.Version
}

type casaService struct{}

/**
 * @description: get remote version (latest non-draft TofuGG/CasaOS GitHub release)
 * @return {model.Version}
 */
func (o *casaService) GetCasaosVersion() model.Version {
	// v2 key: the old "casa_version" key held the raw upstream API envelope;
	// the v2 shape is a plain normalized version string.
	keyName := "casa_version:v2"
	var version model.Version

	if result, ok := Cache.Get(keyName); ok {
		if cached, ok := result.(string); ok && len(cached) > 0 {
			version.Version = cached
			return version
		}
	}

	version = fetchLatestReleaseVersion()

	if len(version.Version) > 0 {
		Cache.Set(keyName, version.Version, time.Minute*20)
	}

	return version
}

// fetchLatestReleaseVersion queries the GitHub releases API for the newest
// non-draft release of TofuGG/CasaOS, normalizing the tag to a plain version
// (e.g. "v0.4.18-fork.1" -> "0.4.18").
//
// It FAILS CLOSED: on any error (rate limit, network failure, empty list,
// malformed JSON) it returns an empty model.Version, so callers never see a
// bogus "update available" badge.
func fetchLatestReleaseVersion() model.Version {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, githubReleasesURL, nil)
	if err != nil {
		logger.Error("failed to create GitHub releases request", zap.Error(err))
		return model.Version{}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "casaos-fork")

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("failed to fetch TofuGG/CasaOS releases", zap.Error(err))
		return model.Version{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("GitHub releases API returned unexpected status", zap.Int("status", resp.StatusCode))
		return model.Version{}
	}

	var releases []struct {
		TagName string `json:"tag_name"`
		Draft   bool   `json:"draft"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		logger.Error("failed to decode TofuGG/CasaOS releases", zap.Error(err))
		return model.Version{}
	}

	for _, release := range releases {
		if release.Draft {
			continue // prereleases are fine, drafts are not
		}

		tag := strings.TrimPrefix(release.TagName, "v")
		if idx := strings.IndexByte(tag, '-'); idx >= 0 {
			tag = tag[:idx]
		}

		if len(tag) > 0 {
			return model.Version{Version: tag}
		}
	}

	logger.Error("no published release found on TofuGG/CasaOS")
	return model.Version{}
}

func NewCasaService() CasaService {
	return &casaService{}
}
