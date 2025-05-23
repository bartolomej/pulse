package sources

import (
	"context"
	"fmt"
)

type LobstersTagSource struct {
	InstanceURL string `yaml:"instance-url"`
	CustomURL   string `yaml:"custom-url"`
	Tag         string `yaml:"tag"`
	client      *LobstersClient
}

func NewLobstersTagSource() *LobstersTagSource {
	return &LobstersTagSource{
		InstanceURL: "https://lobste.rs",
	}
}

func (s *LobstersTagSource) UID() string {
	return fmt.Sprintf("lobsters-tag/%s/%s", s.InstanceURL, s.Tag)
}

func (s *LobstersTagSource) Name() string {
	return fmt.Sprintf("Lobsters (#%s)", s.Tag)
}

func (s *LobstersTagSource) URL() string {
	return fmt.Sprintf("https://lobste.rs/t/%s", s.Tag)
}

func (s *LobstersTagSource) Stream(ctx context.Context, feed chan<- Activity, errs chan<- error) {
	var stories []*Story
	var err error

	if s.CustomURL != "" {
		stories, err = s.client.GetStoriesFromCustomURL(ctx, s.CustomURL)
	} else {
		stories, err = s.client.GetStoriesByTag(ctx, s.Tag)
	}

	if err != nil {
		errs <- err
		return
	}

	for _, story := range stories {
		feed <- &lobstersPost{raw: story}
	}
}

func (s *LobstersTagSource) Initialize() error {
	if s.Tag == "" {
		return fmt.Errorf("tag is required")
	}

	s.client = NewLobstersClient(s.InstanceURL)

	return nil
}
