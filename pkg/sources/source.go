package sources

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func NewSource(widgetType string) (Source, error) {
	if widgetType == "" {
		return nil, errors.New("widget 'type' property is empty or not specified")
	}

	var s Source

	switch widgetType {
	case "mastodon-account":
		s = NewMastodonAccountSource()
	case "mastodon-tag":
		s = NewMastodonTagSource()
	case "hacker-news":
		s = NewHackerNewsSource()
	case "reddit":
		s = NewRedditSource()
	case "lobsters-tag":
		s = NewLobstersTagSource()
	case "lobsters-feed":
		s = NewLobstersFeedSource()
	case "rss":
		s = NewRSSSource()
	case "releases":
		s = NewGithubReleasesSource()
	case "issues":
		s = NewGithubIssuesSource()
	case "change-detection":
		s = NewChangeDetectionSource()
	default:
		return nil, fmt.Errorf("unknown source type: %s", widgetType)
	}

	return s, nil
}

type Source interface {
	UID() string
	// Name is a human-readable UID.
	Name() string
	// URL is a web resource representation of UID.
	URL() string
	Initialize() error
	Stream(ctx context.Context, feed chan<- Activity, errs chan<- error)
}

// Activity TODO(pulse): Compute LLM summary
type Activity interface {
	UID() string
	Title() string
	Body() string
	URL() string
	ImageURL() string
	CreatedAt() time.Time
}
