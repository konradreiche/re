package re

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"time"

	gqlclient "git.sr.ht/~emersion/gqlclient"
)

type Client struct {
	login    string
	endpoint string
	client   *http.Client
	gql      *gqlclient.Client
}

func NewClient(ctx context.Context, config Config) (*Client, error) {
	client := &http.Client{
		Transport: &authenticatedTransport{
			transport:   http.DefaultTransport,
			accessToken: config.AccessToken,
		},
	}
	result := &Client{
		endpoint: config.RESTEndpoint,
		client:   client,
		gql:      gqlclient.New(config.Endpoint+"/graphql", client),
	}
	user, err := FetchLogin(result.gql, ctx)
	if err != nil {
		return nil, fmt.Errorf("FetchLogin failed: %w", err)
	}
	result.login = user.Login
	return result, nil
}

func (c *Client) FetchPullRequests(ctx context.Context, limit int, owner, name string, closed bool) error {
	states := []PullRequestState{"OPEN"}
	if closed {
		states = []PullRequestState{"CLOSED", "MERGED"}
	}
	repository, err := FetchPullRequests(c.gql, ctx, owner, name, int32(limit), states)
	if err != nil {
		return fmt.Errorf("FetchPullRequests: %w", err)
	}
	if repository == nil {
		return errors.New("FetchPullRequests: repository is nil")
	}
	return c.printPullRequests(repository.PullRequests.Edges)
}

type fileResp struct {
	Patch            string `json:"patch"`
	Filename         string `json:"filename"`
	Status           string `json:"status"`
	PreviousFilename string `json:"previous_filename"`
	Changes          int    `json:"changes"`
}

func (c *Client) FetchDiff(ctx context.Context, owner, repository string, pullRequest int, printRaw bool) error {
	url := c.endpoint + "/repos/" + owner + "/" + repository + "/pulls/" + fmt.Sprint(pullRequest) + "/files"
	resp, err := c.client.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	defer resp.Body.Close()
	var result []fileResp
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return err
	}
	return printDiff(result)
}

type CreatePullRequestReview struct {
	Event string `json:"event"`
	Body  string `json:"body,omitempty"`
}

func (c *Client) ReviewPullRequest(ctx context.Context, owner, repository string, pullRequest int, event, comment string) error {
	review := CreatePullRequestReview{
		Event: event,
		Body:  comment,
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(review); err != nil {
		return err
	}

	url := c.endpoint + "/repos/" + owner + "/" + repository + "/pulls/" + fmt.Sprint(pullRequest) + "/reviews"
	resp, err := c.client.Post(url, "application/vnd.github+json", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	return nil
}

func (c *Client) FetchMyPullRequests(ctx context.Context, limit int) error {
	user, err := FetchMyPullRequests(c.gql, ctx, int32(limit))
	if err != nil {
		return err
	}
	return c.printPullRequests(user.PullRequests.Edges)
}

func (c *Client) FetchMyPullRequestReviewQueue(ctx context.Context, query, repository string, limit int) error {
	result, err := FetchMyPullRequestReviewQueue(c.gql, ctx, query, int32(limit))
	if err != nil {
		return err
	}

	edges := make([]*PullRequestEdge, len(result.Edges))
	for i, edge := range result.Edges {
		pr := edge.Node.Value.(*PullRequest)
		if lastReviewAt := c.getLastReviewRequested(pr.TimelineItems.Nodes); lastReviewAt != "" {
			lastReview, err := time.Parse(time.RFC3339, string(lastReviewAt))
			if err != nil {
				return err
			}
			pr.CreatedAt = DateTime(ReviewDue(lastReview).Format(time.RFC3339))
		}

		edges[i] = &PullRequestEdge{
			Node: pr,
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		a, err := time.Parse(time.RFC3339, string(edges[i].Node.CreatedAt))
		if err != nil {
			panic(err)
		}
		b, err := time.Parse(time.RFC3339, string(edges[j].Node.CreatedAt))
		if err != nil {
			panic(err)
		}
		return a.Before(b)
	})

	return c.printPullRequests(edges)
}

func (c *Client) getLastReviewRequested(items []*PullRequestTimelineItems) DateTime {
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		event, ok := item.Value.(*ReviewRequestedEvent)
		if !ok {
			continue
		}
		if event.RequestedReviewer == nil {
			continue
		}
		user, ok := event.RequestedReviewer.Value.(*User)
		if !ok {
			continue
		}
		if user.Login == c.login {
			return event.CreatedAt
		}
	}
	return ""
}

type CreatePullRequest struct {
	Title          string `json:"title"`
	Head           string `json:"head"`
	HeadRepository string `json:"head_repo"`
	Base           string `json:"base"`
	Body           string `json:"body"`
	Draft          bool   `json:"draft"`
}

type CreatePullResponse struct {
	Number int `json:"number"`
}

func (c *Client) CreatePullRequest(ctx context.Context, owner, repository string, args CreatePullRequest) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(args); err != nil {
		return err
	}
	url := c.endpoint + "/repos/" + owner + "/" + repository + "/pulls"
	resp, err := c.client.Post(url, "application/vnd.github.v3+json", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(resp.Status + ": " + string(b))
	}
	var result CreatePullResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return err
	}
	fmt.Println("Created pull request", result.Number)
	return nil
}

func getAge(createdAt time.Time, align bool) string {
	alignFormat := "%d"
	if align {
		alignFormat = "%2d"
	}

	d := time.Since(createdAt)
	if d.Hours() > 24 {
		days := int(d.Hours() / 24)
		if days > 365 {
			return fmt.Sprintf(alignFormat+"y ago", int(days/365))
		}
		return fmt.Sprintf(alignFormat+"d ago", days)
	} else if d.Hours() > 1 {
		return fmt.Sprintf(alignFormat+"h ago", int(math.Ceil(d.Hours())))
	}
	return fmt.Sprintf(alignFormat+"m ago", int(d.Minutes()))
}

type comment struct {
	author    string
	body      string
	createdAt time.Time
	diffHunk  string
}

func newComment(author, body, diffHunk string, createdAt string) (*comment, error) {
	parsed, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, err
	}
	return &comment{
		author:    author,
		body:      body,
		createdAt: parsed,
		diffHunk:  diffHunk,
	}, nil
}

func (c *Client) FetchDescription(ctx context.Context, number int32, name, owner string) error {
	respository, err := FetchConversation(c.gql, ctx, number, name, owner)
	if err != nil {
		return err
	}
	return printDescription(respository.PullRequest)
}

func (c *Client) FetchComments(ctx context.Context, number int, name, owner string) error {
	repository, err := FetchConversation(c.gql, ctx, int32(number), name, owner)
	if err != nil {
		return err
	}
	var comments []*comment
	for _, edge := range repository.PullRequest.Reviews.Edges {
		review := edge.Node
		if review.Body != "" {
			comment, err := newComment(review.Author.Login, review.Body, "", string(review.CreatedAt))
			if err != nil {
				return err
			}
			comments = append(comments, comment)
		}

		for _, edge := range review.Comments.Edges {
			c := edge.Node
			comment, err := newComment(c.Author.Login, c.Body, c.DiffHunk, string(c.CreatedAt))
			if err != nil {
				return err
			}
			comments = append(comments, comment)
		}
	}
	for _, edge := range repository.PullRequest.Comments.Edges {
		c := edge.Node
		comment, err := newComment(c.Author.Login, c.Body, "", string(c.CreatedAt))
		if err != nil {
			return err
		}
		comments = append(comments, comment)
	}

	sort.Slice(comments, func(i, j int) bool {
		return comments[i].createdAt.Before(comments[j].createdAt)
	})
	return printComments(repository.PullRequest, comments)
}

var clientID = "td"

func (c *Client) MarkAsReady(ctx context.Context, owner, name string, number int) error {
	repository, err := FetchPullRequestID(c.gql, ctx, owner, name, int32(number))
	if err != nil {
		return err
	}
	_, err = MarkAsReady(c.gql, ctx, MarkPullRequestReadyForReviewInput{
		ClientMutationId: &clientID,
		PullRequestId:    repository.PullRequest.Id,
	})
	if err != nil {
		return err
	}
	return nil
}

type authenticatedTransport struct {
	transport   http.RoundTripper
	accessToken string
}

func (t *authenticatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	req.Header.Add("Authorization", "Bearer "+t.accessToken)
	return t.transport.RoundTrip(req)
}
