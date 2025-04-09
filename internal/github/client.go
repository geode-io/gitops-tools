package github

import (
	"context"
	"net/http"
	"time"

	actions "github.com/sethvargo/go-githubactions"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v61/github"
)

var (
	failedConclusions = []string{"failure", "cancelled", "timed_out"}
)

type Client struct {
	*github.Client
	ctx context.Context
}

type ClientOpts struct {
	AppKey            string
	AppId, AppInstallationId int64
}

func NewClient(opts *ClientOpts) (*Client, error) {
	var err error
	ctx := context.Background()
	client := github.NewClient(nil)
	client, err = GetGHClient(opts, ctx)
	if err != nil {
		return nil, err
	}
	c := &Client{
		Client: client,
		ctx:    ctx,
	}

	err = c.CheckRateLimit()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func GetGHClient(opts *ClientOpts, ctx context.Context) (*github.Client, error) {
	itr, err := ghinstallation.New(http.DefaultTransport, opts.AppId, opts.AppInstallationId, []byte(opts.AppKey))
	if err != nil {
		return nil, err
	}
	client := github.NewClient(&http.Client{Transport: itr})
	return client, nil
}

func (c *Client) CheckRateLimit() error {
	limit, resp, err := c.Client.RateLimit.Get(c.ctx)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil
		}
		return err
	}
	lim := limit.Core.Limit
	remaining := limit.Core.Remaining
	resetsIn := time.Until(limit.Core.Reset.Time)
	actions.Infof("GitHub API rate limit: %d, remaining: %d, resets in: %s", lim, remaining, resetsIn)
	return nil
}

func GetOwnerAndRepo(e interface{}) (string, string) {
	switch e := e.(type) {
	case *github.CheckSuite:
		return e.GetRepository().GetOwner().GetLogin(), e.GetRepository().GetName()
	case *github.PullRequest:
		return e.GetBase().GetRepo().GetOwner().GetLogin(), e.GetBase().GetRepo().GetName()
	}
	return "", ""
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
