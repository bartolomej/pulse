package sources

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v72/github"
	"gopkg.in/yaml.v3"
)

type githubReleasesSource struct {
	sourceBase       `yaml:",inline"`
	Releases         appReleaseList `yaml:"-"`
	Repository       string         `yaml:"repository"`
	Token            string         `yaml:"token"`
	GitLabToken      string         `yaml:"gitlab-token"`
	Limit            int            `yaml:"limit"`
	ShowSourceIcon   bool           `yaml:"show-source-icon"`
	IncludePreleases bool           `yaml:"include-prereleases"`
	client           *github.Client
}

func (s *githubReleasesSource) Feed() []Activity {
	activities := make([]Activity, len(s.Releases))
	for i, r := range s.Releases {
		activities[i] = r
	}
	return activities
}

func (s *githubReleasesSource) Initialize() error {
	s.withTitle("Releases").withCacheDuration(2 * time.Hour)

	if s.Limit <= 0 {
		s.Limit = 10
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

func (s *githubReleasesSource) Update(ctx context.Context) {
	release, err := fetchLatestRelease(ctx, s.client, s.Repository, s.IncludePreleases)

	if !s.canContinueUpdateAfterHandlingErr(err) {
		return
	}

	s.Releases = appReleaseList{*release}
}

type releaseSource string

const (
	releaseSourceCodeberg  releaseSource = "codeberg"
	releaseSourceGithub    releaseSource = "github"
	releaseSourceGitlab    releaseSource = "gitlab"
	releaseSourceDockerHub releaseSource = "dockerhub"
)

type appRelease struct {
	raw *github.RepositoryRelease
}

func (a appRelease) UID() string {
	return fmt.Sprintf("%d", a.raw.GetID())
}

func (a appRelease) Title() string {
	return a.raw.GetName()
}

func (a appRelease) Body() string {
	return a.raw.GetBody()
}

func (a appRelease) URL() string {
	return a.raw.GetHTMLURL()
}

func (a appRelease) ImageURL() string {
	return ""
}

func (a appRelease) CreatedAt() time.Time {
	return a.raw.GetPublishedAt().Time
}

type appReleaseList []appRelease

func (r appReleaseList) sortByNewest() appReleaseList {
	sort.Slice(r, func(i, j int) bool {
		return r[i].CreatedAt().After(r[j].CreatedAt())
	})

	return r
}

type releaseRequest struct {
	IncludePreleases bool   `yaml:"include-prereleases"`
	Repository       string `yaml:"repository"`

	source releaseSource
	token  *string
}

func (r *releaseRequest) UnmarshalYAML(node *yaml.Node) error {
	type releaseRequestAlias releaseRequest
	alias := (*releaseRequestAlias)(r)
	var repository string

	if err := node.Decode(&repository); err != nil {
		if err := node.Decode(alias); err != nil {
			return fmt.Errorf("could not umarshal repository into string or struct: %v", err)
		}
	}

	if r.Repository == "" {
		if repository == "" {
			return errors.New("repository is required")
		} else {
			r.Repository = repository
		}
	}

	parts := strings.SplitN(repository, ":", 2)
	if len(parts) == 1 {
		r.source = releaseSourceGithub
	} else if len(parts) == 2 {
		r.Repository = parts[1]

		switch parts[0] {
		case string(releaseSourceGithub):
			r.source = releaseSourceGithub
		case string(releaseSourceGitlab):
			r.source = releaseSourceGitlab
		case string(releaseSourceDockerHub):
			r.source = releaseSourceDockerHub
		case string(releaseSourceCodeberg):
			r.source = releaseSourceCodeberg
		default:
			return errors.New("invalid source")
		}
	}

	return nil
}

func fetchLatestReleases(ctx context.Context, client *github.Client, requests []*releaseRequest) (appReleaseList, error) {
	job := newJob(func(request *releaseRequest) (*appRelease, error) {
		return fetchLatestReleaseTask(ctx, client, request)
	}, requests).withWorkers(20)
	results, errs, err := workerPoolDo(job)
	if err != nil {
		return nil, err
	}

	var failed int

	releases := make(appReleaseList, 0, len(requests))

	for i := range results {
		if errs[i] != nil {
			failed++
			slog.Error("Failed to fetch release", "source", requests[i].source, "repository", requests[i].Repository, "error", errs[i])
			continue
		}

		releases = append(releases, *results[i])
	}

	if failed == len(requests) {
		return nil, errNoContent
	}

	releases.sortByNewest()

	if failed > 0 {
		return releases, fmt.Errorf("%w: could not get %d releases", errPartialContent, failed)
	}

	return releases, nil
}

func fetchLatestReleaseTask(ctx context.Context, client *github.Client, request *releaseRequest) (*appRelease, error) {
	switch request.source {
	case releaseSourceCodeberg:
		return fetchLatestCodebergRelease(request)
	case releaseSourceGithub:
		return fetchLatestGithubRelease(ctx, client, request)
	case releaseSourceGitlab:
		return fetchLatestGitLabRelease(request)
	case releaseSourceDockerHub:
		return fetchLatestDockerHubRelease(request)
	}

	return nil, errors.New("unsupported source")
}

func fetchLatestGithubRelease(ctx context.Context, client *github.Client, request *releaseRequest) (*appRelease, error) {
	parts := strings.Split(request.Repository, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s", request.Repository)
	}
	owner, repo := parts[0], parts[1]

	var release *github.RepositoryRelease
	var err error

	if !request.IncludePreleases {
		release, _, err = client.Repositories.GetLatestRelease(ctx, owner, repo)
	} else {
		releases, _, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{PerPage: 1})
		if err != nil {
			return nil, err
		}
		if len(releases) == 0 {
			return nil, fmt.Errorf("no releases found for repository %s", request.Repository)
		}
		release = releases[0]
	}

	if err != nil {
		return nil, err
	}

	return &appRelease{raw: release}, nil
}

type dockerHubRepositoryTagsResponse struct {
	Results []dockerHubRepositoryTagResponse `json:"results"`
}

type dockerHubRepositoryTagResponse struct {
	Name       string `json:"name"`
	LastPushed string `json:"tag_last_pushed"`
}

const dockerHubOfficialRepoTagURLFormat = "https://hub.docker.com/_/%s/tags?name=%s"
const dockerHubRepoTagURLFormat = "https://hub.docker.com/r/%s/tags?name=%s"
const dockerHubTagsURLFormat = "https://hub.docker.com/v2/namespaces/%s/repositories/%s/tags"
const dockerHubSpecificTagURLFormat = "https://hub.docker.com/v2/namespaces/%s/repositories/%s/tags/%s"

func fetchLatestDockerHubRelease(request *releaseRequest) (*appRelease, error) {
	nameParts := strings.Split(request.Repository, "/")

	if len(nameParts) > 2 {
		return nil, fmt.Errorf("invalid repository name: %s", request.Repository)
	} else if len(nameParts) == 1 {
		nameParts = []string{"library", nameParts[0]}
	}

	tagParts := strings.SplitN(nameParts[1], ":", 2)
	var requestURL string

	if len(tagParts) == 2 {
		requestURL = fmt.Sprintf(dockerHubSpecificTagURLFormat, nameParts[0], tagParts[0], tagParts[1])
	} else {
		requestURL = fmt.Sprintf(dockerHubTagsURLFormat, nameParts[0], nameParts[1])
	}

	httpRequest, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	if request.token != nil {
		httpRequest.Header.Add("Authorization", "Bearer "+(*request.token))
	}

	var tag *dockerHubRepositoryTagResponse

	if len(tagParts) == 1 {
		response, err := decodeJsonFromRequest[dockerHubRepositoryTagsResponse](defaultHTTPClient, httpRequest)
		if err != nil {
			return nil, err
		}

		if len(response.Results) == 0 {
			return nil, fmt.Errorf("no tags found for repository: %s", request.Repository)
		}

		tag = &response.Results[0]
	} else {
		response, err := decodeJsonFromRequest[dockerHubRepositoryTagResponse](defaultHTTPClient, httpRequest)
		if err != nil {
			return nil, err
		}

		tag = &response
	}

	var repo string
	var displayName string
	var notesURL string

	if len(tagParts) == 1 {
		repo = nameParts[1]
	} else {
		repo = tagParts[0]
	}

	if nameParts[0] == "library" {
		displayName = repo
		notesURL = fmt.Sprintf(dockerHubOfficialRepoTagURLFormat, repo, tag.Name)
	} else {
		displayName = nameParts[0] + "/" + repo
		notesURL = fmt.Sprintf(dockerHubRepoTagURLFormat, displayName, tag.Name)
	}

	release := &github.RepositoryRelease{
		Name:        &displayName,
		HTMLURL:     &notesURL,
		PublishedAt: &github.Timestamp{Time: parseRFC3339Time(tag.LastPushed)},
	}

	return &appRelease{raw: release}, nil
}

type gitlabReleaseResponseJson struct {
	TagName    string `json:"tag_name"`
	ReleasedAt string `json:"released_at"`
	Links      struct {
		Self string `json:"self"`
	} `json:"_links"`
}

func fetchLatestGitLabRelease(request *releaseRequest) (*appRelease, error) {
	httpRequest, err := http.NewRequest(
		"GET",
		fmt.Sprintf(
			"https://gitlab.com/api/v4/projects/%s/releases/permalink/latest",
			url.QueryEscape(request.Repository),
		),
		nil,
	)
	if err != nil {
		return nil, err
	}

	if request.token != nil {
		httpRequest.Header.Add("PRIVATE-TOKEN", *request.token)
	}

	response, err := decodeJsonFromRequest[gitlabReleaseResponseJson](defaultHTTPClient, httpRequest)
	if err != nil {
		return nil, err
	}

	release := &github.RepositoryRelease{
		Name:        &request.Repository,
		HTMLURL:     &response.Links.Self,
		PublishedAt: &github.Timestamp{Time: parseRFC3339Time(response.ReleasedAt)},
	}

	return &appRelease{raw: release}, nil
}

type codebergReleaseResponseJson struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	HtmlUrl     string `json:"html_url"`
}

func fetchLatestCodebergRelease(request *releaseRequest) (*appRelease, error) {
	httpRequest, err := http.NewRequest(
		"GET",
		fmt.Sprintf(
			"https://codeberg.org/api/v1/repos/%s/releases/latest",
			request.Repository,
		),
		nil,
	)
	if err != nil {
		return nil, err
	}

	response, err := decodeJsonFromRequest[codebergReleaseResponseJson](defaultHTTPClient, httpRequest)
	if err != nil {
		return nil, err
	}

	release := &github.RepositoryRelease{
		Name:        &request.Repository,
		HTMLURL:     &response.HtmlUrl,
		PublishedAt: &github.Timestamp{Time: parseRFC3339Time(response.PublishedAt)},
	}

	return &appRelease{raw: release}, nil
}

func fetchLatestRelease(ctx context.Context, client *github.Client, repository string, includePrereleases bool) (*appRelease, error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s", repository)
	}
	owner, repo := parts[0], parts[1]

	var release *github.RepositoryRelease
	var err error

	if !includePrereleases {
		release, _, err = client.Repositories.GetLatestRelease(ctx, owner, repo)
	} else {
		releases, _, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{PerPage: 1})
		if err != nil {
			return nil, err
		}
		if len(releases) == 0 {
			return nil, fmt.Errorf("no releases found for repository %s", repository)
		}
		release = releases[0]
	}

	if err != nil {
		return nil, err
	}

	return &appRelease{raw: release}, nil
}
