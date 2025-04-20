package fakegithub

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/konradreiche/re/internal/commandtest/fakegithub/graph"
	"github.com/konradreiche/re/internal/commandtest/fakegithub/graph/model"
	"github.com/vektah/gqlparser/v2/ast"
)

type FakeGitHub struct {
	URL string
}

func New(tb testing.TB) *FakeGitHub {
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{
		Viewer: &model.User{
			Login: "foo",
		},
		Repository: &model.Repository{
			ID:   "1",
			Name: "test-repo",
			PullRequests: &model.PullRequestConnection{
				Edges: []*model.PullRequestEdge{
					{
						Node: &model.PullRequest{
							ID:     "1",
							Author: &model.User{Login: "foo"},
							HeadRef: &model.Ref{
								Name: "main",
							},
							CreatedAt: "2006-01-02T15:04:05Z",
							Comments: &model.IssueCommentConnection{
								TotalCount: 1,
							},
							Repository: &model.Repository{Name: "test-repo"},
							Reviews: &model.PullRequestReviewConnection{
								Edges: []*model.PullRequestReviewEdge{},
							},
						},
					},
				},
			},
		},
	}}))
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	ts := httptest.NewServer(srv)
	tb.Cleanup(ts.Close)

	http.Handle("/graphql/query", srv)

	return &FakeGitHub{
		URL: ts.URL,
	}
}
