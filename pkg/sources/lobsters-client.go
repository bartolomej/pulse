package sources

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type LobstersClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewLobstersClient(baseURL string) *LobstersClient {
	if baseURL == "" {
		baseURL = "https://lobste.rs"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &LobstersClient{
		httpClient: defaultHTTPClient,
		baseURL:    baseURL,
	}
}

type Story struct {
	ID           string    `json:"short_id"`
	CreatedAt    string    `json:"created_at"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	Score        int       `json:"score"`
	CommentCount int       `json:"comment_count"`
	CommentsURL  string    `json:"comments_url"`
	Tags         []string  `json:"tags"`
	ParsedTime   time.Time `json:"-"`
}

func (c *LobstersClient) GetStories(ctx context.Context, sortBy string, tags []string) ([]*Story, error) {
	var url string

	if sortBy == "hot" {
		sortBy = "hottest"
	} else if sortBy == "new" {
		sortBy = "newest"
	}

	if len(tags) == 0 {
		url = fmt.Sprintf("%s/%s.json", c.baseURL, sortBy)
	} else {
		tagsStr := strings.Join(tags, ",")
		url = fmt.Sprintf("%s/t/%s.json", c.baseURL, tagsStr)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %v", err)
	}

	var stories []*Story
	stories, err = decodeJsonFromRequest[[]*Story](c.httpClient, req)
	if err != nil {
		return nil, fmt.Errorf("fetching stories: %v", err)
	}

	// Parse timestamps
	for _, story := range stories {
		parsedTime, err := time.Parse(time.RFC3339, story.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing time for story %s: %v", story.ID, err)
		}
		story.ParsedTime = parsedTime
	}

	return stories, nil
}

func (c *LobstersClient) GetStoriesFromCustomURL(ctx context.Context, url string) ([]*Story, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %v", err)
	}

	var stories []*Story
	stories, err = decodeJsonFromRequest[[]*Story](c.httpClient, req)
	if err != nil {
		return nil, fmt.Errorf("fetching stories: %v", err)
	}

	// Parse timestamps
	for _, story := range stories {
		parsedTime, err := time.Parse(time.RFC3339, story.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing time for story %s: %v", story.ID, err)
		}
		story.ParsedTime = parsedTime
	}

	return stories, nil
}
