package reddit

import (
	"context"
	"errors"
	"fmt"
	"github.com/glanceapp/glance/pkg/sources/common"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/go-shiori/go-readability"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type SourceSubreddit struct {
	Subreddit          string `yaml:"subreddit"`
	SortBy             string `yaml:"sort-by"`
	TopPeriod          string `yaml:"top-period"`
	Search             string `yaml:"search"`
	Limit              int    `yaml:"limit"`
	RequestURLTemplate string `yaml:"request-url-template"`
	client             *reddit.Client
	AppAuth            struct {
		Name   string `yaml:"name"`
		ID     string `yaml:"ID"`
		Secret string `yaml:"secret"`
	} `yaml:"app-auth"`
}

func NewSourceSubreddit() *SourceSubreddit {
	return &SourceSubreddit{}
}

func (s *SourceSubreddit) UID() string {
	return fmt.Sprintf("reddit/%s/%s/%s/%s", s.Subreddit, s.SortBy, s.TopPeriod, s.Search)
}

func (s *SourceSubreddit) Name() string {
	return fmt.Sprintf("Reddit (%s, %s, %s)", s.Subreddit, s.SortBy, s.TopPeriod)
}

func (s *SourceSubreddit) URL() string {
	return fmt.Sprintf("https://reddit.com/r/%s/%s", s.Subreddit, s.SortBy)
}

type redditPost struct {
	raw *reddit.Post
}

func (p *redditPost) UID() string {
	return p.raw.ID
}

func (p *redditPost) Title() string {
	return html.UnescapeString(p.raw.Title)
}

func (p *redditPost) Body() string {
	body := p.raw.Body
	if p.raw.URL != "" && !p.raw.IsSelfPost {
		article, err := readability.FromURL(p.raw.URL, 5*time.Second)
		if err == nil {
			body += fmt.Sprintf("\n\nReferenced article: \n%s", article.TextContent)
		} else {
			slog.Error("Failed to fetch reddit article", "error", err, "url", p.raw.URL)
		}
	}
	return body
}

func (p *redditPost) URL() string {
	// TODO(pulse): Test format
	return "https://www.reddit.com" + p.raw.Permalink
}

func (p *redditPost) ImageURL() string {
	// TODO(pulse): Fetch thumbnail URL
	// The go-reddit library doesn't provide direct access to thumbnail URLs
	// We'll need to fetch this information separately if needed
	return ""
}

func (p *redditPost) CreatedAt() time.Time {
	return p.raw.Created.Time
}

func (s *SourceSubreddit) Initialize() error {
	if s.Subreddit == "" {
		return errors.New("subreddit is required")
	}

	sort := s.SortBy
	if sort != "hot" && sort != "new" && sort != "top" && sort != "rising" {
		s.SortBy = "hot"
	}

	p := s.TopPeriod
	if p != "hour" && p != "day" && p != "week" && p != "month" && p != "year" && p != "all" {
		s.TopPeriod = "day"
	}

	if s.RequestURLTemplate != "" {
		if !strings.Contains(s.RequestURLTemplate, "{REQUEST-URL}") {
			return errors.New("no `{REQUEST-URL}` placeholder specified")
		}
	}

	var client *reddit.Client
	var err error

	if s.AppAuth.ID != "" && s.AppAuth.Secret != "" {
		client, err = reddit.NewClient(reddit.Credentials{
			ID:     s.AppAuth.ID,
			Secret: s.AppAuth.Secret,
		})
	} else {
		client, err = reddit.NewReadonlyClient()
	}

	if err != nil {
		return fmt.Errorf("creating reddit client: %v", err)
	}

	s.client = client

	return nil
}

func (s *SourceSubreddit) Stream(ctx context.Context, feed chan<- common.Activity, errs chan<- error) {
	posts, err := s.fetchSubredditPosts(ctx)

	if err != nil {
		errs <- fmt.Errorf("fetching posts: %v", err)
		return
	}

	for _, post := range posts {
		feed <- post
	}
}

func (s *SourceSubreddit) fetchSubredditPosts(ctx context.Context) ([]*redditPost, error) {
	var posts []*reddit.Post
	var err error

	opts := &reddit.ListOptions{
		Limit: s.Limit,
	}

	if s.Search != "" {
		searchOpts := &reddit.ListPostSearchOptions{
			ListPostOptions: reddit.ListPostOptions{
				ListOptions: reddit.ListOptions{
					Limit: s.Limit,
				},
			},
			Sort: s.SortBy,
		}
		posts, _, err = s.client.Subreddit.SearchPosts(ctx, s.Subreddit, s.Search, searchOpts)
	} else {
		switch s.SortBy {
		case "hot":
			posts, _, err = s.client.Subreddit.HotPosts(ctx, s.Subreddit, opts)
		case "new":
			posts, _, err = s.client.Subreddit.NewPosts(ctx, s.Subreddit, opts)
		case "top":
			topOpts := &reddit.ListPostOptions{
				ListOptions: reddit.ListOptions{
					Limit: s.Limit,
				},
				Time: s.TopPeriod,
			}
			posts, _, err = s.client.Subreddit.TopPosts(ctx, s.Subreddit, topOpts)
		case "rising":
			posts, _, err = s.client.Subreddit.RisingPosts(ctx, s.Subreddit, opts)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("fetching posts: %v", err)
	}

	if len(posts) == 0 {
		return nil, fmt.Errorf("no posts found")
	}

	redditPosts := make([]*redditPost, 0, len(posts))
	for _, post := range posts {
		if post.Stickied {
			continue
		}

		redditPosts = append(redditPosts, &redditPost{raw: post})
	}

	return redditPosts, nil
}
