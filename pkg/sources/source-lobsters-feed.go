package sources

import (
	"context"
	"fmt"
)

type LobstersFeedSource struct {
	InstanceURL string `yaml:"instance-url"`
	CustomURL   string `yaml:"custom-url"`
	FeedName    string `yaml:"feed"`
	client      *LobstersClient
}

func NewLobstersFeedSource() *LobstersFeedSource {
	return &LobstersFeedSource{
		InstanceURL: "https://lobste.rs",
	}
}

func (s *LobstersFeedSource) UID() string {
	return fmt.Sprintf("lobsters-feed/%s/%s", s.InstanceURL, s.FeedName)
}

func (s *LobstersFeedSource) Name() string {
	return fmt.Sprintf("Lobsters (%s)", s.FeedName)
}

func (s *LobstersFeedSource) URL() string {
	return fmt.Sprintf("https://lobste.rs/%s", s.FeedName)
}

func (s *LobstersFeedSource) Initialize() error {

	if s.FeedName == "" {
		s.FeedName = "hottest"
	}

	if s.FeedName != "hottest" && s.FeedName != "newest" {
		return fmt.Errorf("sort-by must be either 'hottest' or 'newest'")
	}

	s.client = NewLobstersClient(s.InstanceURL)

	return nil
}

func (s *LobstersFeedSource) Stream(ctx context.Context, feed chan<- Activity, errs chan<- error) {
	var stories []*Story
	var err error

	if s.CustomURL != "" {
		stories, err = s.client.GetStoriesFromCustomURL(ctx, s.CustomURL)
	} else {
		stories, err = s.client.GetStoriesByFeed(ctx, s.FeedName)
	}

	if err != nil {
		errs <- err
		return
	}

	for _, story := range stories {
		feed <- &lobstersPost{raw: story}
	}

}
