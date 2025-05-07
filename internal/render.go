package re

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

//go:embed markdown.json
var stylesheet embed.FS

var (
	yellow = lipgloss.NewStyle().
		Foreground(lipgloss.Color("3"))
	blue = lipgloss.NewStyle().
		Foreground(lipgloss.Color("4"))
	white = lipgloss.NewStyle().
		Foreground(lipgloss.Color("15"))
)

func (c *Client) printNotificationHeader(notification Notification, pr int) error {
	header := fmt.Sprintf(blue.Render("%d: %s"), pr, notification.Subject.Title)
	fmt.Println(header)
	return nil
}

func (c *Client) printPullRequests(pullRequestEdges []*PullRequestEdge) error {
	var differentRepositories bool
	var repositoryName string
	for _, edge := range pullRequestEdges {
		if repositoryName == "" {
			repositoryName = edge.Node.Repository.Name
		}
		if repositoryName != edge.Node.Repository.Name {
			differentRepositories = true
			break
		}
	}

	fmt.Print("\r") // TODO: cross-platform

	red := lipgloss.NewStyle().
		Foreground(lipgloss.Color("1"))
	green := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2"))
	yellow := lipgloss.NewStyle().
		Foreground(lipgloss.Color("3"))

	writer := tabwriter.NewWriter(os.Stdout, 3, 3, 3, ' ', 0)
	for _, edge := range pullRequestEdges {
		var (
			participating   bool
			lastCommentByMe bool
		)

		pr := edge.Node
		createdAt, err := time.Parse(time.RFC3339, string(pr.CreatedAt))
		if err != nil {
			return err
		}
		if pr.HeadRef != nil && len(pr.HeadRef.Name) > 15 {
			pr.HeadRef.Name = pr.HeadRef.Name[:15] + "…"
		}
		if len(pr.Author.Login) > 30 {
			pr.Author.Login = pr.Author.Login[:30] + "…"
		}
		author := white.Render(pr.Author.Login)
		if pr.Author.Login == c.login {
			author = green.Render(c.login)
		}

		numComments := pr.Comments.TotalCount
		for i, review := range pr.Reviews.Edges {
			if review.Node.Body != "" {
				numComments += 1
			}
			numComments += review.Node.Comments.TotalCount

			if review.Node.Author.Login == c.login {
				participating = true
				if i == len(pr.Reviews.Edges)-1 {
					lastCommentByMe = true
				}
			}
		}

		if len(pr.Title) > 80 {
			pr.Title = pr.Title[:80] + "…"
		}

		statusCheckIcon := green.Render("✓")
		switch statusCheck(pr) {
		case StatusStatePending, StatusStateExpected:
			statusCheckIcon = yellow.Render("◯")
		case StatusStateFailure, StatusStateError:
			statusCheckIcon = red.Render("✗")
		}

		mailIcon := white.Render("🗨")

		comments := white.Render(fmt.Sprintf("%3d", numComments))
		if participating {
			mailIcon = green.Render("🗨")
			comments = green.Render(fmt.Sprintf("%3d", numComments))
			if !lastCommentByMe {
				mailIcon = yellow.Render("🗨")
				comments = yellow.Render(fmt.Sprintf("%3d", numComments))
			}
		}

		fmt.Fprintf(writer, "%s\t%s\t%s\t%v %s\t%v %s",
			white.Render(fmt.Sprint(pr.Number)),
			author,
			white.Render(pr.Title),
			comments,
			mailIcon,
			white.Render(getAge(createdAt, true)),
			statusCheckIcon,
		)

		if differentRepositories {
			fmt.Fprintf(writer, "\t%s", white.Render(pr.Repository.Name))
		}

		fmt.Fprint(writer, "\n")
	}
	return writer.Flush()
}

func statusCheck(pr *PullRequest) StatusState {
	if pr.Commits == nil {
		return StatusStateSuccess
	}
	for _, node := range pr.Commits.Nodes {
		if node.Commit.Status == nil {
			// Not every commit has a status associated with it.
			continue
		}
		for _, context := range node.Commit.Status.Contexts {
			if context.State != StatusStateSuccess {
				return context.State
			}
		}
	}
	return StatusStateSuccess
}

func printComments(pr *PullRequest, comments []*comment) error {
	yellow := lipgloss.NewStyle().
		Foreground(lipgloss.Color("3"))

	b, err := stylesheet.ReadFile("markdown.json")
	if err != nil {
		return err
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(b),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		return err
	}

	body, err := r.Render(pr.Body)
	if err != nil {
		return err
	}

	createdAt, err := time.Parse(time.RFC3339, string(pr.CreatedAt))
	if err != nil {
		return err
	}
	fmt.Printf("%s (%s)\n\n", yellow.Render(pr.Author.Login), yellow.Render(getAge(createdAt, false)))
	fmt.Printf("%s\n", body)

	diffHunks := make(map[string]string)
	for _, comment := range comments {
		body := strings.ReplaceAll(comment.body, "\r\n", "\n")
		body, err := r.Render(body)
		if err != nil {
			return err
		}
		fmt.Printf("%s (%s)\n\n", yellow.Render(comment.author), yellow.Render(getAge(comment.createdAt, false)))

		if comment.diffHunk != "" && diffHunks[comment.diffHunk] == "" {
			diff, err := r.Render("```diff\n" + comment.diffHunk + "\n```")
			if err != nil {
				return err
			}
			fmt.Printf("%s\n\n", diff)
			diffHunks[comment.diffHunk] = comment.diffHunk
		}

		fmt.Printf("%s\n", body)
	}
	return nil
}

func printDiff(patches []fileResp) error {
	var b bytes.Buffer

	for _, file := range patches {
		switch file.Status {
		case "added":
			fmt.Fprintf(&b, "diff --git a/%s\n", file.Filename)
			fmt.Fprintf(&b, "--- /dev/null\n")
			fmt.Fprintf(&b, "+++ b/%s\n", file.Filename)
		case "modified":
			fmt.Fprintf(&b, "diff --git a/%s b/%s\n", file.Filename, file.Filename)
			fmt.Fprintf(&b, "--- a/%s\n", file.Filename)
			fmt.Fprintf(&b, "+++ b/%s\n", file.Filename)
		case "removed":
			fmt.Fprintf(&b, "diff --git a/%s b/%s\n", file.Filename, file.Filename)
			fmt.Fprintf(&b, "--- a/%s\n", file.Filename)
			fmt.Fprint(&b, "+++ /dev/null\n")
		case "renamed":
			fmt.Fprintf(&b, "diff --git a/%s b/%s\n", file.PreviousFilename, file.Filename)
			fmt.Fprintf(&b, "rename from %s\n", file.PreviousFilename)
			fmt.Fprintf(&b, "rename to %s\n", file.Filename)
			if file.Changes > 0 {
				fmt.Fprintf(&b, "--- a/%s\n", file.PreviousFilename)
				fmt.Fprintf(&b, "+++ b/%s\n", file.Filename)
			}
		default:
			return fmt.Errorf("printDiff: unhandled file status: %s", file.Status)
		}
		if file.Patch != "" {
			fmt.Fprintln(&b, file.Patch)
		}
	}

	cmd := exec.Command("delta")
	cmd.Stdin = strings.NewReader(b.String())
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func printDescription(pr *PullRequest) error {
	user, ok := pr.Author.Value.(*User)
	if !ok {
		return fmt.Errorf("printDescription: unexpected type: %T", pr.Author.Value)
	}

	fmt.Println(yellow.Render(pr.Title))
	fmt.Println(white.Render("Author:", *user.Name))
	fmt.Println(white.Render("Date:   " + string(pr.CreatedAt)))
	fmt.Println()

	b, err := stylesheet.ReadFile("markdown.json")
	if err != nil {
		return err
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(80),
		glamour.WithStylesFromJSONBytes(b),
	)
	if err != nil {
		return err
	}
	description, err := r.Render(pr.Body)
	if err != nil {
		return err
	}
	fmt.Println(description)
	return nil
}
