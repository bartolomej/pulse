package sources

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mattn/go-mastodon"
	"golang.org/x/net/html"
)

type mastodonSource struct {
	sourceBase     `yaml:",inline"`
	Posts          []*mastodonPost `yaml:"-"`
	InstanceURL    string          `yaml:"instance-url"`
	Accounts       []string        `yaml:"accounts"`
	Hashtags       []string        `yaml:"hashtags"`
	Limit          int             `yaml:"limit"`
	ShowThumbnails bool            `yaml:"-"`
	client         *mastodon.Client
}

type mastodonPost struct {
	raw *mastodon.Status
}

func (p *mastodonPost) UID() string {
	return string(p.raw.ID)
}

func (p *mastodonPost) Title() string {
	if p.raw.Card != nil {
		return p.raw.Card.Title
	}

	return oneLineTitle(p.Body(), 50)
}

func (p *mastodonPost) Body() string {
	return extractTextFromHTML(p.raw.Content)
}

func (p *mastodonPost) URL() string {
	return p.raw.URL
}

func (p *mastodonPost) ImageURL() string {
	if len(p.raw.MediaAttachments) > 0 {
		return p.raw.MediaAttachments[0].URL
	}
	return ""
}

func (p *mastodonPost) CreatedAt() time.Time {
	return p.raw.CreatedAt
}

func (s *mastodonSource) Feed() []Activity {
	activities := make([]Activity, len(s.Posts))
	for i, post := range s.Posts {
		activities[i] = post
	}
	return activities
}

func (s *mastodonSource) Initialize() error {
	if s.InstanceURL == "" {
		return fmt.Errorf("instance-url is required")
	}

	s.
		withTitle("Mastodon").
		withTitleURL(s.InstanceURL).
		withCacheDuration(30 * time.Minute)

	if s.Limit <= 0 {
		s.Limit = 15
	}

	s.client = mastodon.NewClient(&mastodon.Config{
		Server: s.InstanceURL,
	})

	return nil
}

func (s *mastodonSource) Update(ctx context.Context) {
	posts, err := fetchMastodonPosts(ctx, s.client, s.Accounts, s.Hashtags)

	if !s.canContinueUpdateAfterHandlingErr(err) {
		return
	}

	if s.Limit < len(posts) {
		posts = posts[:s.Limit]
	}

	s.Posts = posts
}

func fetchMastodonPosts(ctx context.Context, client *mastodon.Client, accounts []string, hashtags []string) ([]*mastodonPost, error) {
	var posts []*mastodonPost

	// Fetch posts from specified accounts
	for _, account := range accounts {
		accountPosts, err := client.GetAccountStatuses(ctx, mastodon.ID(account), nil)
		if err != nil {
			slog.Error("Failed to fetch Mastodon account posts", "error", err, "account", account)
			continue
		}

		for _, post := range accountPosts {
			posts = append(posts, &mastodonPost{raw: post})
		}
	}

	// Fetch posts from specified hashtags
	for _, hashtag := range hashtags {
		hashtagPosts, err := client.GetTimelineHashtag(ctx, hashtag, false, nil)
		if err != nil {
			slog.Error("Failed to fetch Mastodon hashtag posts", "error", err, "hashtag", hashtag)
			continue
		}

		for _, post := range hashtagPosts {
			posts = append(posts, &mastodonPost{raw: post})
		}
	}

	if len(posts) == 0 {
		return nil, errNoContent
	}

	return posts, nil
}

func extractTextFromHTML(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr
	}
	var b strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return strings.TrimSpace(b.String())
}

func oneLineTitle(text string, maxLen int) string {
	re := regexp.MustCompile(`\s+`)
	t := re.ReplaceAllString(text, " ")
	t = strings.TrimSpace(t)
	if utf8.RuneCountInString(t) > maxLen {
		runes := []rune(t)
		return string(runes[:maxLen-1]) + "…"
	}
	return t
}
