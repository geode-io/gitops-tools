package github

import (
	"fmt"

	"github.com/google/go-github/v61/github"
	actions "github.com/sethvargo/go-githubactions"
)

// TODO: Make this more configurable to allow for different types of checks
func (c *Client) WaitForPRChecks(pr *github.PullRequest) error {
	checks, err := c.GetPRChecks(pr)
	if err != nil {
		checksErr := c.ChecksErr(checks)
		switch err.Error() {
		case "CheckNotCompleted":
			actions.Infof("One or more checks have not completed yet. retrying...")
			return checksErr
		case "CheckFailed":
			actions.Infof("One or more checks failed. retrying...")
			return checksErr
		case "NoChecksFound":
			actions.Infof("No checks found for PR. This is likely due to a delay in the checks being reported by GitHub. retrying...")
			return checksErr
		default:
			return err
		}
	}
	return nil
}

func (c *Client) ChecksErr(checks *github.ListCheckSuiteResults) (err error) {
	success := true
	if checks == nil || len(checks.CheckSuites) == 0 {
		return fmt.Errorf("NoChecksFound")
	}
	for _, check := range checks.CheckSuites {
		success = evaluateConclusion(check.GetConclusion())
		slug := check.GetApp().GetSlug()
		actions.Debugf("Check: %d, status: %s, conclusion: %s", check.GetID(), check.GetStatus(), check.GetConclusion())
		if slug != "github-actions" {
			actions.Debugf("Skipping check from %s as it is not a github action", slug)
			continue
		}
		if check.GetStatus() != "completed" {
			return fmt.Errorf("CheckNotCompleted")
		}
		if !success {
			actions.Infof("one or more checks failed.")
			return fmt.Errorf("CheckFailed")
		}
	}
	return nil
}

func (c *Client) GetPRChecks(pr *github.PullRequest) (*github.ListCheckSuiteResults, error) {
	owner, repo := GetOwnerAndRepo(pr)
	num := pr.GetNumber()
	checks, _, err := c.Checks.ListCheckSuitesForRef(c.ctx, owner, repo, fmt.Sprintf("refs/pull/%d/head", num), nil)
	if err != nil {
		return nil, err
	}
	return checks, nil
}

func evaluateConclusion(conclusion string) bool {
	for _, c := range failedConclusions {
		if c == conclusion {
			return false
		}
	}
	return true
}
