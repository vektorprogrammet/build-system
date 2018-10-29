package messenger

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type GithubCommenter struct {
	ProgressCommentId int64
	PrNumber          int
}

func (g *GithubCommenter) createClient() (*github.Client, context.Context) {
	ctx := context.Background()
	accessToken := os.Getenv("GITHUB_ACCESS_TOKEN")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc), ctx
}

func (g *GithubCommenter) Comment(comment string) (*github.IssueComment, error) {
	client, ctx := g.createClient()

	prComment := github.IssueComment{
		Body: &comment,
	}

	issue, _, err := client.Issues.CreateComment(ctx, "vektorprogrammet", "vektorprogrammet", g.PrNumber, &prComment)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

func (g *GithubCommenter) EditComment(id int64, comment string) (*github.IssueComment, error) {
	client, ctx := g.createClient()

	prComment := github.IssueComment{
		Body: &comment,
	}

	issue, _, err := client.Issues.EditComment(ctx, "vektorprogrammet", "vektorprogrammet", id, &prComment)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

func (g *GithubCommenter) DeleteComment(id int64) error {
	client, ctx := g.createClient()

	_, err := client.Issues.DeleteComment(ctx, "vektorprogrammet", "vektorprogrammet", id)
	if err != nil {
		return err
	}

	return nil
}

func (g *GithubCommenter) StartingDeploy() {
	issueComment, _ := g.Comment("Starting deploy to staging server...")
	g.ProgressCommentId = *issueComment.ID
}

func (g *GithubCommenter) UpdateProgress(message string, progress int) {
	comment := fmt.Sprintf("Deploying this pull request to the staging server... %d %%\n%s", progress, message)
	g.EditComment(g.ProgressCommentId, comment)
}

func (g *GithubCommenter) Delete() {
	g.DeleteComment(g.ProgressCommentId)
}
