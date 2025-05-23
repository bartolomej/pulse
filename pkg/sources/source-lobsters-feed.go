package sources

import (
	"context"
	"fmt"
	"time"
)

type lobstersFeedSource struct {
	sourceBase  `yaml:",inline"`
	Posts       []*lobstersPost `yaml:"-"`
	InstanceURL string          `yaml:"instance-url"`
	CustomURL   string          `yaml:"custom-url"`
	Limit       int             `yaml:"limit"`
	SortBy      string          `yaml:"sort-by"`
	client      *LobstersClient
}

func (s *lobstersFeedSource) Feed() []Activity {
	activities := make([]Activity, len(s.Posts))
	for i, post := range s.Posts {
		activities[i] = post
	}
	return activities
}

func (s *lobstersFeedSource) Initialize() error {
	s.withTitle("Lobsters Feed").withCacheDuration(time.Hour)

	if s.InstanceURL == "" {
		s.withTitleURL("https://lobste.rs")
	} else {
		s.withTitleURL(s.InstanceURL)
	}

	if s.SortBy == "" {
		s.SortBy = "hot"
	}

	if s.SortBy != "hot" && s.SortBy != "new" {
		return fmt.Errorf("sort-by must be either 'hot' or 'new'")
	}

	if s.Limit <= 0 {
		s.Limit = 15
	}

	s.client = NewLobstersClient(s.InstanceURL)

	return nil
}

func (s *lobstersFeedSource) Update(ctx context.Context) {
	var stories []*Story
	var err error

	if s.CustomURL != "" {
		stories, err = s.client.GetStoriesFromCustomURL(ctx, s.CustomURL)
	} else {
		stories, err = s.client.GetStoriesBySort(ctx, s.SortBy)
	}

	if !s.canContinueUpdateAfterHandlingErr(err) {
		return
	}

	if len(stories) == 0 {
		return
	}

	posts := make([]*lobstersPost, 0, len(stories))
	for _, story := range stories {
		posts = append(posts, &lobstersPost{raw: story})
	}

	if s.Limit < len(posts) {
		posts = posts[:s.Limit]
	}

	s.Posts = posts
}
