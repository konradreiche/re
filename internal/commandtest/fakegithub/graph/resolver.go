package graph

import "github.com/konradreiche/re/internal/commandtest/fakegithub/graph/model"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
type Resolver struct {
	Viewer     *model.User
	Repository *model.Repository
}
