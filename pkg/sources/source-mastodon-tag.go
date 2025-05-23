package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/mattn/go-mastodon"
)

type mastodonTagSource struct {
	sourceBase
	Posts       []*mastodonPost
	InstanceURL string
	Hashtag     string
	Limit       int
}

func (s *mastodonTagSource) Feed() []Activity {
	activities := make([]Activity, len(s.Posts))
	for i, post := range s.Posts {
		activities[i] = post
	}
	return activities
}

func (s *mastodonTagSource) Initialize() error {
	if s.InstanceURL == "" {
		return fmt.Errorf("instance URL is required")
	}
	if s.Hashtag == "" {
		return fmt.Errorf("hashtag is required")
	}
	if s.Limit <= 0 {
		s.Limit = 20
	}

	s.withTitle("Mastodon Hashtag").
		withTitleURL(s.InstanceURL).
		withCacheDuration(30 * time.Minute)

	return nil
}

func (s *mastodonTagSource) Update(ctx context.Context) {
	client := mastodon.NewClient(&mastodon.Config{
		Server:       s.InstanceURL,
		ClientID:     "pulse-feed-aggregation",
		ClientSecret: "pulse-feed-aggregation",
	})

	posts, err := fetchHashtagPosts(client, s.Hashtag, s.Limit)
	if err != nil {
		s.withError(fmt.Errorf("failed to fetch posts: %w", err))
		return
	}

	s.Posts = posts
}

func fetchHashtagPosts(client *mastodon.Client, hashtag string, limit int) ([]*mastodonPost, error) {
	statuses, err := client.GetTimelineHashtag(context.Background(), hashtag, false, &mastodon.Pagination{
		Limit: int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get hashtag timeline: %w", err)
	}

	posts := make([]*mastodonPost, len(statuses))
	for i, status := range statuses {
		posts[i] = &mastodonPost{raw: status}
	}

	return posts, nil
}
