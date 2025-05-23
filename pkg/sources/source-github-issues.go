package sources

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v72/github"
)

type githubIssuesSource struct {
	sourceBase    `yaml:",inline"`
	Issues        issueActivityList `yaml:"-"`
	Repository    string            `yaml:"repository"`
	Token         string            `yaml:"token"`
	Limit         int               `yaml:"limit"`
	ActivityTypes []string          `yaml:"activity-types"`
	client        *github.Client
}

func (s *githubIssuesSource) Feed() []Activity {
	activities := make([]Activity, len(s.Issues))
	for i, issue := range s.Issues {
		activities[i] = issue
	}
	return activities
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

func (s *githubIssuesSource) Initialize() error {
	s.withTitle("Issue Activity").withCacheDuration(30 * time.Minute)

	if s.Limit <= 0 {
		s.Limit = 10
	}

	if len(s.ActivityTypes) == 0 {
		s.ActivityTypes = []string{"opened", "closed", "commented"}
	}

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

func (s *githubIssuesSource) Update(ctx context.Context) {
	activities, err := fetchIssueActivities(ctx, s.client, s.Repository, s.ActivityTypes)

	if !s.canContinueUpdateAfterHandlingErr(err) {
		return
	}

	if len(activities) > s.Limit {
		activities = activities[:s.Limit]
	}

	s.Issues = activities
}

func fetchIssueActivities(ctx context.Context, client *github.Client, repository string, activityTypes []string) (issueActivityList, error) {
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
