package re

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func GetRepositoryAndOrgName() (string, string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	b, err := cmd.CombinedOutput()
	if err != nil {
		if code, ok := err.(*exec.ExitError); ok {
			// TODO: consider moving to calling side
			if code.ExitCode() == 128 {
				return "", "", formatCommandError("GetRepositoryAndOrgName", cmd, b)
			}
		}
		return "", "", formatCommandError("GetRepositoryAndOrgName", cmd, b)
	}
	origin := strings.TrimSuffix(string(b), "\n")
	split := strings.SplitN(origin, ":", 2)
	if len(split) == 1 {
		// TODO: consider moving to calling side
		return "", "", fmt.Errorf("GetRepositoryAndOrgName: invalid string: %s", origin)
	}
	return filepath.Dir(split[1]), strings.TrimSuffix(filepath.Base(split[1]), ".git"), nil
}

func PushToOrigin() error {
	branch, err := CurrentBranch()
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "push", "origin", branch)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("PushToOrigin: %s: %s", cmd.String(), string(b))
	}
	fmt.Println(string(b))
	return nil
}

func CheckoutPullRequest(pr int) error {
	if _, err := exec.Command("git", "fetch", "origin", "pull/"+fmt.Sprint(pr)+"/head").CombinedOutput(); err != nil {
		return err
	}
	if _, err := exec.Command("git", "checkout", "FETCH_HEAD").CombinedOutput(); err != nil {
		return err
	}
	return nil
}

func CurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(b), "\n"), nil
}

func GetTitleAndBody() (string, string, error) {
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%s%n%b")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", err
	}
	body, title, err := formatTitleAndBody(b)
	if err != nil {
		return "", "", err
	}

	return body, title, nil
}

func GetDefaultBranch() (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	split := strings.Split(strings.TrimSuffix(string(b), "\n"), "/")
	return split[len(split)-1], nil
}

func formatTitleAndBody(logOutput []byte) (string, string, error) {
	split := strings.SplitN(string(logOutput), "\n", 2)
	title := split[0]
	body := split[1]

	body = strings.TrimSpace(body)

	return title, JoinLines(body), nil
}

func JoinLines(input string) string {
	var output bytes.Buffer
	scanner := bufio.NewScanner(strings.NewReader(input))
	var paragraphLines []string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if len(paragraphLines) > 0 {
				output.WriteString(strings.Join(paragraphLines, " ") + "\n\n")
				paragraphLines = nil
			} else {
				output.WriteString("\n")
			}
		} else {
			paragraphLines = append(paragraphLines, line)
		}
	}

	if len(paragraphLines) > 0 {
		output.WriteString(strings.Join(paragraphLines, " "))
	}
	return output.String()
}

func formatCommandError(name string, cmd *exec.Cmd, output []byte) error {
	return fmt.Errorf("%s: %s: %s", name, cmd.String(), strings.TrimSuffix(string(output), "\n"))
}
