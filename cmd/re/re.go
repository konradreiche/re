package main

import (
	"fmt"
	"os"
	"strconv"

	re "github.com/konradreiche/re/internal"
	"github.com/spf13/cobra"
)

var (
	commander *re.Command
	pr        int
	lines     int
	message   string
)

var rootCmd = &cobra.Command{
	Use:               "re",
	Short:             "📬 re (again) – review, respond, rethink",
	PersistentPreRunE: newCommander,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var approveCmd = &cobra.Command{
	Use:     "approve",
	Short:   "Approve a pull request",
	Args:    cobra.ExactArgs(1),
	PreRunE: parseIntArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		return commander.ApprovePullRequest(cmd.Context(), pr, message)
	},
}

var commentCmd = &cobra.Command{
	Use:     "comment",
	Short:   "Comment on a pull request",
	Args:    cobra.ExactArgs(1),
	PreRunE: parseIntArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		return commander.ApprovePullRequest(cmd.Context(), pr, message)
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		return commander.CreatePullRequest(cmd.Context())
	},
}

var openCmd = &cobra.Command{
	Use:     "open",
	Short:   "Open a pull request in the browser",
	Args:    cobra.ExactArgs(1),
	PreRunE: parseIntArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		return commander.OpenPullRequest(cmd.Context(), pr)
	},
}

var diffCmd = &cobra.Command{
	Use:     "diff",
	Short:   "Display the diff of a pull request",
	Args:    cobra.ExactArgs(1),
	PreRunE: parseIntArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		return commander.PrintDiff(cmd.Context(), pr)
	},
}

var checkoutCmd = &cobra.Command{
	Use:     "checkout",
	Short:   "Locally checkout a pull request",
	Args:    cobra.ExactArgs(1),
	PreRunE: parseIntArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		return commander.CheckoutPullRequest(cmd.Context(), pr)
	},
}

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "List notifications",
	RunE: func(cmd *cobra.Command, args []string) error {
		return commander.PrintNotifications(cmd.Context())
	},
}

var listCmd = &cobra.Command{
	Use:   "ls",
	Short: "List pull requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		return commander.ListPullRequests(cmd.Context(), lines, false)
	},
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Updates the branch using --force-with-lease",
	RunE: func(cmd *cobra.Command, args []string) error {
		return commander.PushBranch(cmd.Context())
	},
}

var readyCmd = &cobra.Command{
	Use:     "ready",
	Short:   "Mark a pull request as ready for review",
	Args:    cobra.ExactArgs(1),
	PreRunE: parseIntArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		return commander.MarkPullRequestReady(cmd.Context(), pr)
	},
}

var showCmd = &cobra.Command{
	Use:     "show",
	Short:   "Display a pull requst",
	Args:    cobra.ExactArgs(1),
	PreRunE: parseIntArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		return commander.PrintComments(cmd.Context(), pr)
	},
}

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Show pull requests that require your review",
	RunE: func(cmd *cobra.Command, args []string) error {
		return commander.PrintPendingReviews(cmd.Context(), lines, false)
	},
}

func newCommander(cmd *cobra.Command, args []string) error {
	// TODO: Consider using command annotations to store whether a git repository
	// is required to simplify this logic.
	requireGit := true
	switch cmd.Name() {
	case "re":
		return nil
	case "review", "inbox":
		requireGit = false
	}
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	commands, err := re.NewCommand(cmd.Context(), re.NewConfig(), re.WithRequireGit(requireGit))
	if err != nil {
		return err
	}
	commander = commands
	return nil
}

func parseIntArg(cmd *cobra.Command, args []string) error {
	n, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	pr = n
	return nil
}

func main() {
	rootCmd.PersistentFlags().IntVarP(&lines, "lines", "n", 20, "print up to many lines")
	rootCmd.PersistentFlags().StringVarP(&message, "message", "m", "", "provide comment")

	rootCmd.AddCommand(readyCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(commentCmd)
	rootCmd.AddCommand(checkoutCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(approveCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(inboxCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(showCmd)

	if err := rootCmd.Execute(); err != nil {
		exit(err)
	}
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
