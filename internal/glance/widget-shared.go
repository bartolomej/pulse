package glance

import (
	"math"
	"sort"
	"time"
)

var forumPostsTemplate = mustParseTemplate("forum-posts.html", "widget-base.html")

type forumPost struct {
	ID          string
	Title       string
	Description string
	// MatchSummary is the LLM generated rationale for why this is a good match for the filter query
	MatchSummary string
	// MatchScore is the LLM generated score indicating how well this post matches the query
	MatchScore      int
	DiscussionUrl   string
	TargetUrl       string
	TargetUrlDomain string
	ThumbnailUrl    string
	CommentCount    int
	Score           int
	Engagement      float64
	TimePosted      time.Time
	Tags            []string
	IsCrosspost     bool
}

type forumPostList []forumPost

const depreciatePostsOlderThanHours = 7
const maxDepreciation = 0.9
const maxDepreciationAfterHours = 24

func (p forumPostList) calculateEngagement() {
	var totalComments int
	var totalScore int

	for i := range p {
		totalComments += p[i].CommentCount
		totalScore += p[i].Score
	}

	numberOfPosts := float64(len(p))
	averageComments := float64(totalComments) / numberOfPosts
	averageScore := float64(totalScore) / numberOfPosts

	for i := range p {
		p[i].Engagement = (float64(p[i].CommentCount)/averageComments + float64(p[i].Score)/averageScore) / 2

		elapsed := time.Since(p[i].TimePosted)

		if elapsed < time.Hour*depreciatePostsOlderThanHours {
			continue
		}

		p[i].Engagement *= 1.0 - (math.Max(elapsed.Hours()-depreciatePostsOlderThanHours, maxDepreciationAfterHours)/maxDepreciationAfterHours)*maxDepreciation
	}
}

func (p forumPostList) sortByEngagement() {
	sort.Slice(p, func(i, j int) bool {
		return p[i].Engagement > p[j].Engagement
	})
}
