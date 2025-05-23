package sources

import (
	"context"
	"errors"
	"fmt"
	"github.com/glanceapp/glance/pkg/sources/changedetection"
	"github.com/glanceapp/glance/pkg/sources/common"
	"github.com/glanceapp/glance/pkg/sources/github"
	"github.com/glanceapp/glance/pkg/sources/hackernews"
	"github.com/glanceapp/glance/pkg/sources/lobsters"
	"github.com/glanceapp/glance/pkg/sources/mastodon"
	"github.com/glanceapp/glance/pkg/sources/reddit"
	"github.com/glanceapp/glance/pkg/sources/rss"
)

func NewSource(widgetType string) (Source, error) {
	if widgetType == "" {
		return nil, errors.New("widget 'type' property is empty or not specified")
	}

	var s Source

	switch widgetType {
	case "mastodon-account":
		s = mastodon.NewSourceAccount()
	case "mastodon-tag":
		s = mastodon.NewSourceTag()
	case "hacker-news":
		s = hackernews.NewHackerNewsSource()
	case "reddit":
		s = reddit.NewSourceSubreddit()
	case "lobsters-tag":
		s = lobsters.NewSourceTag()
	case "lobsters-feed":
		s = lobsters.NewSourceFeed()
	case "rss":
		s = rss.NewSourceFeed()
	case "releases":
		s = github.NewReleaseSource()
	case "issues":
		s = github.NewIssuesSource()
	case "change-detection":
		s = changedetection.NewSourceWebsiteChange()
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
	Stream(ctx context.Context, feed chan<- common.Activity, errs chan<- error)
}
