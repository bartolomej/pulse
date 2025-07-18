package rss

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/glanceapp/glance/pkg/sources/activities/types"
	"github.com/glanceapp/glance/pkg/utils"

	"github.com/mmcdole/gofeed"
	gofeedext "github.com/mmcdole/gofeed/extensions"
)

const TypeRSSFeed = "rss-feed"

type customTransport struct {
	headers map[string]string
	base    http.RoundTripper
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range t.headers {
		req.Header.Set(key, value)
	}
	return t.base.RoundTrip(req)
}

type SourceFeed struct {
	FeedURL string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

func NewSourceFeed() *SourceFeed {
	return &SourceFeed{}
}

func (s *SourceFeed) UID() string {
	return fmt.Sprintf("%s/%s", s.Type(), s.FeedURL)
}

func (s *SourceFeed) Name() string {
	return fmt.Sprintf("RSS (%s)", s.FeedURL)
}

func (s *SourceFeed) URL() string {
	return s.FeedURL
}

func (s *SourceFeed) Type() string {
	return TypeRSSFeed
}

func (s *SourceFeed) Initialize() error {
	if s.FeedURL == "" {
		return fmt.Errorf("URL is required")
	}

	return nil
}

func (s *SourceFeed) Stream(ctx context.Context, feed chan<- types.Activity, errs chan<- error) {
	parser := gofeed.NewParser()
	parser.UserAgent = utils.PulseUserAgentString

	if s.Headers != nil {
		parser.Client = &http.Client{
			Transport: &customTransport{
				headers: s.Headers,
				base:    http.DefaultTransport,
			},
		}
	}

	rssFeed, err := parser.ParseURL(s.FeedURL)
	if err != nil {
		errs <- fmt.Errorf("failed to parse RSS feed: %w", err)
		return
	}

	if rssFeed == nil {
		errs <- fmt.Errorf("feed is nil")
		return
	}

	if len(rssFeed.Items) == 0 {
		errs <- fmt.Errorf("feed has no items")
		return
	}

	for _, item := range rssFeed.Items {
		feed <- &FeedItem{Item: item, FeedURL: s.FeedURL, SourceTyp: s.Type(), SourceID: s.UID()}
	}
}

type FeedItem struct {
	Item      *gofeed.Item `json:"item"`
	FeedURL   string       `json:"feed_url"`
	SourceID  string       `json:"source_id"`
	SourceTyp string       `json:"source_type"`
}

func NewFeedItem() *FeedItem {
	return &FeedItem{}
}

func (e *FeedItem) SourceType() string {
	return e.SourceTyp
}

func (e *FeedItem) MarshalJSON() ([]byte, error) {
	type Alias FeedItem
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	})
}

func (e *FeedItem) UnmarshalJSON(data []byte) error {
	type Alias FeedItem
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	return json.Unmarshal(data, &aux)
}

func (e *FeedItem) UID() string {
	if e.Item.GUID != "" {
		return e.Item.GUID
	}
	return e.URL()
}

func (e *FeedItem) SourceUID() string {
	return e.SourceID
}

func (e *FeedItem) Title() string {
	if e.Item.Title != "" {
		return html.UnescapeString(e.Item.Title)
	}
	return shortenFeedDescriptionLen(e.Item.Description, 100)
}

func (e *FeedItem) Body() string {
	if e.Item.Content != "" {
		return e.Item.Content
	}
	return e.Item.Description
}

func (e *FeedItem) URL() string {
	if strings.HasPrefix(e.Item.Link, "http://") || strings.HasPrefix(e.Item.Link, "https://") {
		return e.Item.Link
	}

	parsedUrl, err := url.Parse(e.FeedURL)
	if err == nil {
		link := e.Item.Link
		if !strings.HasPrefix(link, "/") {
			link = "/" + link
		}
		return parsedUrl.Scheme + "://" + parsedUrl.Host + link
	}
	return e.Item.Link
}

func (e *FeedItem) ImageURL() string {
	if e.Item.Image != nil && e.Item.Image.URL != "" {
		return e.Item.Image.URL
	}
	if url := findThumbnailInItemExtensions(e.Item); url != "" {
		return url
	}
	return ""
}

func (e *FeedItem) CreatedAt() time.Time {
	if e.Item.PublishedParsed != nil {
		return *e.Item.PublishedParsed
	}
	if e.Item.UpdatedParsed != nil {
		return *e.Item.UpdatedParsed
	}
	return time.Now()
}

func (e *FeedItem) Categories() []string {
	categories := make([]string, 0, len(e.Item.Categories))
	for _, category := range e.Item.Categories {
		if category != "" {
			categories = append(categories, category)
		}
	}
	return categories
}

func findThumbnailInItemExtensions(item *gofeed.Item) string {
	media, ok := item.Extensions["media"]

	if !ok {
		return ""
	}

	return recursiveFindThumbnailInExtensions(media)
}

func recursiveFindThumbnailInExtensions(extensions map[string][]gofeedext.Extension) string {
	for _, exts := range extensions {
		for _, ext := range exts {
			if ext.Name == "thumbnail" || ext.Name == "image" {
				if url, ok := ext.Attrs["url"]; ok {
					return url
				}
			}

			if ext.Children != nil {
				if url := recursiveFindThumbnailInExtensions(ext.Children); url != "" {
					return url
				}
			}
		}
	}

	return ""
}

var htmlTagsWithAttributesPattern = regexp.MustCompile(`<\/?[a-zA-Z0-9-]+ *(?:[a-zA-Z-]+=(?:"|').*?(?:"|') ?)* *\/?>`)
var sequentialWhitespacePattern = regexp.MustCompile(`\s+`)

func sanitizeFeedDescription(description string) string {
	if description == "" {
		return ""
	}

	description = strings.ReplaceAll(description, "\n", " ")
	description = htmlTagsWithAttributesPattern.ReplaceAllString(description, "")
	description = sequentialWhitespacePattern.ReplaceAllString(description, " ")
	description = strings.TrimSpace(description)
	description = html.UnescapeString(description)

	return description
}

func shortenFeedDescriptionLen(description string, maxLen int) string {
	description, _ = utils.LimitStringLength(description, 1000)
	description = sanitizeFeedDescription(description)
	description, limited := utils.LimitStringLength(description, maxLen)

	if limited {
		description += "…"
	}

	return description
}

func (s *SourceFeed) MarshalJSON() ([]byte, error) {
	type Alias SourceFeed
	return json.Marshal(&struct {
		*Alias
		Type string `json:"type"`
	}{
		Alias: (*Alias)(s),
		Type:  s.Type(),
	})
}

func (s *SourceFeed) UnmarshalJSON(data []byte) error {
	type Alias SourceFeed
	aux := &struct {
		*Alias
		Type string `json:"type"`
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	return nil
}
