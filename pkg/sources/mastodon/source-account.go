package mastodon

import (
	"context"
	"fmt"
	"github.com/glanceapp/glance/pkg/sources/common"
	"github.com/mattn/go-mastodon"
)

type SourceAccount struct {
	InstanceURL string
	Account     string
}

func NewSourceAccount() *SourceAccount {
	return &SourceAccount{
		InstanceURL: "https://mastodon.social",
	}
}

func (s *SourceAccount) UID() string {
	return fmt.Sprintf("mastodon/%s/%s", s.InstanceURL, s.Account)
}

func (s *SourceAccount) Name() string {
	return fmt.Sprintf("Mastodon (%s)", s.Account)
}

func (s *SourceAccount) URL() string {
	return fmt.Sprintf("%s/tags/%s", s.InstanceURL, s.Account)
}

func (s *SourceAccount) Initialize() error {
	if s.InstanceURL == "" {
		return fmt.Errorf("instance URL is required")
	}
	if s.Account == "" {
		return fmt.Errorf("account is required")
	}

	return nil
}

func (s *SourceAccount) Stream(ctx context.Context, feed chan<- common.Activity, errs chan<- error) {
	client := mastodon.NewClient(&mastodon.Config{
		Server:       s.InstanceURL,
		ClientID:     "pulse-feed-aggregation",
		ClientSecret: "pulse-feed-aggregation",
	})

	accountID, err := getAccountID(client, s.Account)
	if err != nil {
		errs <- fmt.Errorf("failed to get account ID: %w", err)
		return
	}

	limit := 15
	posts, err := fetchAccountPosts(client, accountID, limit)
	if err != nil {
		errs <- fmt.Errorf("failed to fetch posts: %w", err)
		return
	}

	for _, post := range posts {
		feed <- post
	}
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
