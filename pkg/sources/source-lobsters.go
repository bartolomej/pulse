package sources

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-shiori/go-readability"
)

type lobstersSource struct {
	sourceBase     `yaml:",inline"`
	Posts          []*lobstersPost `yaml:"-"`
	InstanceURL    string          `yaml:"instance-url"`
	CustomURL      string          `yaml:"custom-url"`
	Limit          int             `yaml:"limit"`
	SortBy         string          `yaml:"sort-by"`
	Tags           []string        `yaml:"tags"`
	ShowThumbnails bool            `yaml:"-"`
	client         *LobstersClient
}

type lobstersPost struct {
	raw *Story
}

func (p *lobstersPost) UID() string {
	return p.raw.ID
}

func (p *lobstersPost) Title() string {
	return p.raw.Title
}

func (p *lobstersPost) Body() string {
	body := p.raw.Title
	if p.raw.URL != "" {
		article, err := readability.FromURL(p.raw.URL, 5*time.Second)
		if err == nil {
			body += "\n\nReferenced article: \n" + article.TextContent
		} else {
			slog.Error("Failed to fetch lobster article", "error", err, "url", p.raw.URL)
		}
	}
	return body
}

func (p *lobstersPost) URL() string {
	return p.raw.URL
}

func (p *lobstersPost) ImageURL() string {
	return ""
}

func (p *lobstersPost) CreatedAt() time.Time {
	return p.raw.ParsedTime
}

func (s *lobstersSource) Feed() []Activity {
	activities := make([]Activity, len(s.Posts))
	for i, post := range s.Posts {
		activities[i] = post
	}
	return activities
}

func (s *lobstersSource) Initialize() error {
	s.withTitle("Lobsters").withCacheDuration(time.Hour)

	if s.InstanceURL == "" {
		s.withTitleURL("https://lobste.rs")
	} else {
		s.withTitleURL(s.InstanceURL)
	}

	if s.SortBy == "" || (s.SortBy != "hot" && s.SortBy != "new") {
		s.SortBy = "hot"
	}

	if s.Limit <= 0 {
		s.Limit = 15
	}

	s.client = NewLobstersClient(s.InstanceURL)

	return nil
}

func (s *lobstersSource) Update(ctx context.Context) {
	var stories []*Story
	var err error

	if s.CustomURL != "" {
		stories, err = s.client.GetStoriesFromCustomURL(ctx, s.CustomURL)
	} else {
		stories, err = s.client.GetStories(ctx, s.SortBy, s.Tags)
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
