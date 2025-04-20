package commandtest

import (
	"testing"

	re "github.com/konradreiche/re/internal"
	"github.com/konradreiche/re/internal/commandtest/fakegithub"
)

func New(tb testing.TB) *re.Command {
	fake := fakegithub.New(tb)
	config := re.Config{
		Endpoint: fake.URL,
	}
	command, err := re.NewCommand(tb.Context(), config)
	if err != nil {
		tb.Fatal(err)
	}
	return command
}
