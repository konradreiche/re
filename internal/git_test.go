package re

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFormatBody(t *testing.T) {
	input := `Update schema
Had to delete Query case manually, not sure why it didn't get created
with gqlclientgen.

This is another line to test things.`

	gotTitle, gotBody, err := formatTitleAndBody([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	wantBody := `Had to delete Query case manually, not sure why it didn't get created with gqlclientgen.

This is another line to test things.`
	if diff := cmp.Diff(gotBody, wantBody); diff != "" {
		t.Errorf("diff: %s", diff)
	}

	wantTitle := "Update schema"
	if diff := cmp.Diff(gotTitle, wantTitle); diff != "" {
		t.Errorf("diff: %s", diff)
	}
}
