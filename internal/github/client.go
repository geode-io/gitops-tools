package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	actions "github.com/sethvargo/go-githubactions"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v61/github"
	"golang.org/x/oauth2"
)

var (
	failedConclusions = []string{"failure", "cancelled", "timed_out"}
)

type Client struct {
	*github.Client
	ctx context.Context
}

type ClientOpts struct {
	Token, AppKey            string
	AppId, AppInstallationId int64
}

func NewClient(opts *ClientOpts) (*Client, error) {
	var err error
	ctx := context.Background()
	client := github.NewClient(nil)

	if opts.Token != "" {
		client, err = GetGHClient(opts, ctx, "pat")
		if err != nil {
			return nil, err
		}
	} else {
		client, err = GetGHClient(opts, ctx, "app")
		if err != nil {
			return nil, err
		}
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

func GetGHClient(opts *ClientOpts, ctx context.Context, clientType string) (*github.Client, error) {
	switch clientType {
	case "pat":
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: opts.Token},
		)
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)
		return client, nil
	case "app":
		itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, opts.AppId, opts.AppInstallationId, opts.AppKey)
		if err != nil {
			return nil, err
		}
		client := github.NewClient(&http.Client{Transport: itr})
		return client, nil

	}
	return nil, fmt.Errorf("invalid client type")
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
