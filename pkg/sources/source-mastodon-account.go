package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/mattn/go-mastodon"
)

type mastodonAccountSource struct {
	sourceBase
	Posts       []*mastodonPost
	InstanceURL string
	Account     string
	Limit       int
}

func (s *mastodonAccountSource) Feed() []Activity {
	activities := make([]Activity, len(s.Posts))
	for i, post := range s.Posts {
		activities[i] = post
	}
	return activities
}

func (s *mastodonAccountSource) Initialize() error {
	if s.InstanceURL == "" {
		return fmt.Errorf("instance URL is required")
	}
	if s.Account == "" {
		return fmt.Errorf("account is required")
	}
	if s.Limit <= 0 {
		s.Limit = 20
	}

	s.withTitle("Mastodon Account").
		withTitleURL(s.InstanceURL).
		withCacheDuration(30 * time.Minute)

	return nil
}

func (s *mastodonAccountSource) Update(ctx context.Context) {
	client := mastodon.NewClient(&mastodon.Config{
		Server:       s.InstanceURL,
		ClientID:     "pulse-feed-aggregation",
		ClientSecret: "pulse-feed-aggregation",
	})

	accountID, err := getAccountID(client, s.Account)
	if err != nil {
		s.withError(fmt.Errorf("failed to get account ID: %w", err))
		return
	}

	posts, err := fetchAccountPosts(client, accountID, s.Limit)
	if err != nil {
		s.withError(fmt.Errorf("failed to fetch posts: %w", err))
		return
	}

	s.Posts = posts
}

func getAccountID(client *mastodon.Client, account string) (mastodon.ID, error) {
	accounts, err := client.Search(context.Background(), account, false)
	if err != nil {
		return "", fmt.Errorf("failed to search for account: %w", err)
	}

	if len(accounts.Accounts) == 0 {
		return "", fmt.Errorf("account not found: %s", account)
	}

	return accounts.Accounts[0].ID, nil
}

func fetchAccountPosts(client *mastodon.Client, accountID mastodon.ID, limit int) ([]*mastodonPost, error) {
	statuses, err := client.GetAccountStatuses(context.Background(), accountID, &mastodon.Pagination{
		Limit: int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get account statuses: %w", err)
	}

	posts := make([]*mastodonPost, len(statuses))
	for i, status := range statuses {
		posts[i] = &mastodonPost{raw: status}
	}

	return posts, nil
}
