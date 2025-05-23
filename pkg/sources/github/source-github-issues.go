package github

import (
	"context"
	"fmt"
	"github.com/glanceapp/glance/pkg/sources/common"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v72/github"
)

type SourceIssues struct {
	Repository string `json:"repository"`
	Token      string `json:"token"`
	client     *github.Client
}

func NewIssuesSource() *SourceIssues {
	return &SourceIssues{}
}

func (s *SourceIssues) UID() string {
	return fmt.Sprintf("issues/%s", s.Repository)
}

func (s *SourceIssues) Name() string {
	return fmt.Sprintf("Issue Activity (%s)", s.Repository)
}

func (s *SourceIssues) URL() string {
	return fmt.Sprintf("https://github.com/%s", s.Repository)
}

type issueActivity struct {
	raw *github.Issue
}

func (i issueActivity) UID() string {
	return fmt.Sprintf("issue-%d", i.raw.GetNumber())
}

func (i issueActivity) Title() string {
	return i.raw.GetTitle()
}

func (i issueActivity) Body() string {
	return i.raw.GetBody()
}

func (i issueActivity) URL() string {
	return i.raw.GetHTMLURL()
}

func (i issueActivity) ImageURL() string {
	return ""
}

func (i issueActivity) CreatedAt() time.Time {
	return i.raw.GetUpdatedAt().Time
}

type issueActivityList []issueActivity

func (i issueActivityList) sortByNewest() issueActivityList {
	sort.Slice(i, func(a, b int) bool {
		return i[a].CreatedAt().After(i[b].CreatedAt())
	})
	return i
}

func (s *SourceIssues) Initialize() error {
	token := s.Token
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	if token != "" {
		s.client = github.NewClient(nil).WithAuthToken(token)
	} else {
		s.client = github.NewClient(nil)
	}

	return nil
}

func (s *SourceIssues) Stream(ctx context.Context, feed chan<- common.Activity, errs chan<- error) {
	activities, err := fetchIssueActivities(ctx, s.client, s.Repository)

	if err != nil {
		errs <- err
		return
	}

	for _, activity := range activities {
		feed <- activity
	}
}

func fetchIssueActivities(ctx context.Context, client *github.Client, repository string) (issueActivityList, error) {
	activities, err := fetchIssueActivityTask(ctx, client, repository)
	if err != nil {
		return nil, err
	}

	activities.sortByNewest()
	return activities, nil
}

func fetchIssueActivityTask(ctx context.Context, client *github.Client, repository string) (issueActivityList, error) {
	activities := make([]issueActivity, 0)

	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s", repository)
	}
	owner, repo := parts[0], parts[1]

	issues, _, err := client.Issues.ListByRepo(ctx, owner, repo, &github.IssueListByRepoOptions{
		State:       "all",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: github.ListOptions{PerPage: 10},
	})
	if err != nil {
		return nil, err
	}

	for _, issue := range issues {
		activities = append(activities, issueActivity{raw: issue})
	}

	return activities, nil
}
