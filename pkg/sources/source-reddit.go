package sources

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/go-shiori/go-readability"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type redditSource struct {
	sourceBase          `yaml:",inline"`
	Posts               []*redditPost     `yaml:"-"`
	Subreddit           string            `yaml:"subreddit"`
	Proxy               proxyOptionsField `yaml:"proxy"`
	Style               string            `yaml:"style"`
	ShowThumbnails      bool              `yaml:"show-thumbnails"`
	ShowFlairs          bool              `yaml:"show-flairs"`
	SortBy              string            `yaml:"sort-by"`
	TopPeriod           string            `yaml:"top-period"`
	Search              string            `yaml:"search"`
	ExtraSortBy         string            `yaml:"extra-sort-by"`
	CommentsURLTemplate string            `yaml:"comments-url-template"`
	Limit               int               `yaml:"limit"`
	RequestURLTemplate  string            `yaml:"request-url-template"`
	client              *reddit.Client
	AppAuth             struct {
		Name   string `yaml:"name"`
		ID     string `yaml:"ID"`
		Secret string `yaml:"secret"`
	} `yaml:"app-auth"`
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

func (s *redditSource) Feed() []Activity {
	activities := make([]Activity, len(s.Posts))
	for i, post := range s.Posts {
		activities[i] = post
	}
	return activities
}

func (s *redditSource) Initialize() error {
	if s.Subreddit == "" {
		return errors.New("subreddit is required")
	}

	if s.Limit <= 0 {
		s.Limit = 15
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

	s.
		withTitle("r/" + s.Subreddit).
		withTitleURL("https://www.reddit.com/r/" + s.Subreddit + "/").
		withCacheDuration(30 * time.Minute)

	return nil
}

func (s *redditSource) Update(ctx context.Context) {
	posts, err := s.fetchSubredditPosts(ctx)
	if !s.canContinueUpdateAfterHandlingErr(err) {
		return
	}

	if len(posts) > s.Limit {
		posts = posts[:s.Limit]
	}

	s.Posts = posts
}

func (s *redditSource) fetchSubredditPosts(ctx context.Context) ([]*redditPost, error) {
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
