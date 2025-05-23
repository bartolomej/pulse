package hackernews

import (
	"context"
	"fmt"
	"github.com/glanceapp/glance/pkg/sources/common"
	"log/slog"
	"time"

	"github.com/alexferrari88/gohn/pkg/gohn"
	"github.com/go-shiori/go-readability"
)

type SourcePosts struct {
	SortBy string `yaml:"sort-by"`
	client *gohn.Client
}

func NewHackerNewsSource() *SourcePosts {
	return &SourcePosts{}
}

func (s *SourcePosts) UID() string {
	return fmt.Sprintf("hackernews/%s", s.SortBy)
}

func (s *SourcePosts) Name() string {
	return fmt.Sprintf("HackerNews (%s)", s.SortBy)
}

func (s *SourcePosts) URL() string {
	return fmt.Sprintf("https://news.ycombinator.com/%s", s.SortBy)
}

type hackerNewsPost struct {
	raw *gohn.Item
}

func (p *hackerNewsPost) UID() string {
	return fmt.Sprintf("%d", *p.raw.ID)
}

func (p *hackerNewsPost) Title() string {
	return *p.raw.Title
}

func (p *hackerNewsPost) Body() string {
	body := *p.raw.Title
	if p.raw.URL != nil {
		article, err := readability.FromURL(*p.raw.URL, 5*time.Second)
		if err == nil {
			body += fmt.Sprintf("\n\nReferenced article: \n%s", article.TextContent)
		} else {
			slog.Error("Failed to fetch hacker news article", "error", err, "url", *p.raw.URL)
		}
	}
	return body
}

func (p *hackerNewsPost) URL() string {
	if p.raw.URL != nil {
		return *p.raw.URL
	}
	return fmt.Sprintf("https://news.ycombinator.com/item?id=%d", *p.raw.ID)
}

func (p *hackerNewsPost) ImageURL() string {
	return ""
}

func (p *hackerNewsPost) CreatedAt() time.Time {
	return time.Unix(int64(*p.raw.Time), 0)
}

func (s *SourcePosts) Initialize() error {
	if s.SortBy != "top" && s.SortBy != "new" && s.SortBy != "best" {
		s.SortBy = "top"
	}

	var err error
	s.client, err = gohn.NewClient(nil)
	if err != nil {
		return fmt.Errorf("creating hacker news client: %v", err)
	}

	return nil
}

func (s *SourcePosts) Stream(ctx context.Context, feed chan<- common.Activity, errs chan<- error) {
	posts, err := s.fetchHackerNewsPosts(ctx)

	if err != nil {
		errs <- fmt.Errorf("fetching posts: %v", err)
		return
	}

	for _, post := range posts {
		feed <- post
	}

}

func (s *SourcePosts) fetchHackerNewsPosts(ctx context.Context) ([]*hackerNewsPost, error) {
	var storyIDs []*int
	var err error

	switch s.SortBy {
	case "top":
		storyIDs, err = s.client.Stories.GetTopIDs(ctx)
	case "new":
		storyIDs, err = s.client.Stories.GetNewIDs(ctx)
	case "best":
		storyIDs, err = s.client.Stories.GetBestIDs(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("fetching story IDs: %v", err)
	}

	if len(storyIDs) == 0 {
		return nil, fmt.Errorf("no stories found")
	}

	posts := make([]*hackerNewsPost, 0, len(storyIDs))
	for _, id := range storyIDs {
		if id == nil {
			continue
		}

		story, err := s.client.Items.Get(ctx, *id)
		if err != nil {
			slog.Error("Failed to fetch hacker news story", "error", err, "id", *id)
			continue
		}

		if story == nil {
			continue
		}

		posts = append(posts, &hackerNewsPost{raw: story})
	}

	if len(posts) == 0 {
		return nil, fmt.Errorf("no valid stories found")
	}

	return posts, nil
}
