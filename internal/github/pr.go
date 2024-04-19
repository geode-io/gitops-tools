package github

import (
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/google/go-github/v61/github"
	actions "github.com/sethvargo/go-githubactions"
)

func (c *Client) GetPR(owner, repo, branch string) (*github.PullRequest, error) {
	prs, _, err := c.PullRequests.List(c.ctx, owner, repo, &github.PullRequestListOptions{
		Head: fmt.Sprintf("%s:%s", owner, branch),
	})
	if err != nil {
		return nil, err
	}
	if len(prs) == 0 {
		return nil, fmt.Errorf("no PR found for branch %s", branch)
	}
	return prs[0], nil
}

func (c *Client) CreatePR(owner, repo, head, base, title, body string) (*github.PullRequest, error) {
	pr := &github.NewPullRequest{
		Title: &title,
		Head:  &head,
		Base:  &base,
		Body:  &body,
	}
	pull, _, err := c.Client.PullRequests.Create(c.ctx, owner, repo, pr)
	if err != nil {
		return nil, err
	}
	return pull, nil
}

func (c *Client) MergePR(pr *github.PullRequest) error {
	owner, repo := GetOwnerAndRepo(pr)
	num := pr.GetNumber()
	_, _, err := c.PullRequests.Merge(c.ctx, owner, repo, num, "", &github.PullRequestOptions{
		MergeMethod: "squash",
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Deploy(pr *github.PullRequest) error {
	// sleep for 5 seconds before checking the status of the PR checks
	time.Sleep(5 * time.Second)
	err := retry.Do(
		func() error { return c.WaitForPRChecks(pr) },
		retry.Attempts(60),
		retry.Delay(5*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(n uint, err error) {
			actions.Infof("waiting for checks to pass: %v", err)
		}),
	)
	if err != nil {
		return err
	}

	err = retry.Do(
		func() error { return c.MergePR(pr) },
		retry.Attempts(24),
		retry.Delay(5*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(n uint, err error) {
			actions.Infof("attempt: %d to merge PR: %v", n, err)
		}),
	)

	if err != nil {
		return err
	}
	return nil
}
