package re

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type Command struct {
	client *Client
	org    string
	name   string
}

func NewCommand(ctx context.Context, config Config) (*Command, error) {
	client, err := NewClient(ctx, config)
	if err != nil {
		return nil, err
	}
	org, name, err := GetRepositoryAndOrgName()
	if err != nil {
		return nil, err
	}
	return &Command{
		client: client,
		org:    org,
		name:   name,
	}, nil
}

func (c *Command) ApprovePullRequest(ctx context.Context, pr int, message string) error {
	return c.client.ReviewPullRequest(ctx, c.org, c.name, pr, "APPROVE", message)
}

func (c *Command) CommentPullRequest(ctx context.Context, pr int, message string) error {
	return c.client.ReviewPullRequest(ctx, c.org, c.name, pr, "COMMENT", message)
}

func (c *Command) PrintDiff(ctx context.Context, pr int) error {
	return c.client.FetchDiff(ctx, c.org, c.name, pr, false)
}

func (c *Command) MarkPullRequestReady(ctx context.Context, pr int) error {
	return c.client.MarkAsReady(ctx, c.org, c.name, pr)
}

func (c *Command) PrintComments(ctx context.Context, pr int) error {
	return c.client.FetchComments(ctx, pr, c.org, c.name)
}

func (c *Command) PrintPendingReviews(ctx context.Context, limit int, includeTeamReview bool) error {
	query := "is:pr is:open user-review-requested:@me"
	if includeTeamReview {
		query = "is:pr is:open review-requested:@me"
	}
	return c.client.FetchMyPullRequestReviewQueue(ctx, query, c.name, limit)
}

func (c *Command) ListPullRequests(ctx context.Context, limit int, includeClosed bool) error {
	return c.client.FetchPullRequests(ctx, limit, c.org, c.name, includeClosed)
}

func (c *Command) PrintMyPullRequests(ctx context.Context, limit int) error {
	return c.client.FetchMyPullRequests(ctx, limit)
}

func (c *Command) CheckoutPullRequest(ctx context.Context, pr int) error {
	return CheckoutPullRequest(pr)
}

func (c *Command) OpenPullRequest(ctx context.Context, pr int) error {
	endpoint := "https://github.com"
	if ghe := os.Getenv("GITHUB_ENTERPRISE_URL"); ghe != "" {
		endpoint = ghe
	}
	cmd := exec.Command("chromium", endpoint+"/"+c.org+"/"+c.name+"/pull/"+fmt.Sprint(pr))
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func (c *Command) CreatePullRequest(ctx context.Context) error {
	if err := PushToOrigin(); err != nil {
		return err
	}
	title, body, err := GetTitleAndBody()
	if err != nil {
		return err
	}
	branch, err := CurrentBranch()
	if err != nil {
		return err
	}
	defaultBranch, err := GetDefaultBranch()
	if err != nil {
		return err
	}
	return c.client.CreatePullRequest(ctx, c.org, c.name, CreatePullRequest{
		Title: title,
		Head:  branch,
		Base:  defaultBranch,
		Body:  body,
		Draft: true,
	})
}

func (c *Command) PushBranch(ctx context.Context) error {
	return UpdateOrigin()
}
