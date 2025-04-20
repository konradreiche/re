package re_test

import (
	"testing"

	"github.com/konradreiche/re/internal/commandtest"
)

func TestListPullRequests(t *testing.T) {
	command := commandtest.New(t)
	if err := command.ListPullRequests(t.Context(), 20, false); err != nil {
		t.Fatal(err)
	}
}
