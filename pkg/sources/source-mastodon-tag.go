package sources

import (
	"context"
	"fmt"
	"github.com/mattn/go-mastodon"
)

type MastodonTagSource struct {
	InstanceURL string
	Hashtag     string
}

func NewMastodonTagSource() *MastodonTagSource {
	return &MastodonTagSource{
		InstanceURL: "https://mastodon.social",
	}
}

func (s *MastodonTagSource) UID() string {
	return fmt.Sprintf("mastodon/%s/%s", s.InstanceURL, s.Hashtag)
}

func (s *MastodonTagSource) Name() string {
	return fmt.Sprintf("Mastodon (%s)", s.Hashtag)
}

func (s *MastodonTagSource) URL() string {
	return fmt.Sprintf("%s/tags/%s", s.InstanceURL, s.Hashtag)
}

func (s *MastodonTagSource) Initialize() error {
	if s.InstanceURL == "" {
		return fmt.Errorf("instance URL is required")
	}
	if s.Hashtag == "" {
		return fmt.Errorf("hashtag is required")
	}

	return nil
}

func (s *MastodonTagSource) Stream(ctx context.Context, feed chan<- Activity, errs chan<- error) {
	client := mastodon.NewClient(&mastodon.Config{
		Server:       s.InstanceURL,
		ClientID:     "pulse-feed-aggregation",
		ClientSecret: "pulse-feed-aggregation",
	})

	limit := 15
	posts, err := fetchHashtagPosts(client, s.Hashtag, limit)
	if err != nil {
		errs <- fmt.Errorf("failed to fetch posts: %w", err)
		return
	}

	for _, post := range posts {
		feed <- post
	}
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
