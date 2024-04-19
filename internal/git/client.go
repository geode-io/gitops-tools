package git

import (
	"context"
	net "net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type ClientOpts struct {
	Token, AppKey            string
	AppId, AppInstallationId int64
	AuthorName, AuthorEmail  string
}

// Client is a wrapper around go-git to simplify git operations.
type Client struct {
	auth        *http.BasicAuth
	authMethod  string
	authorName  string
	authorEmail string
	itr         *ghinstallation.Transport
	ctx         context.Context
}

func (c *Client) RefreshToken() error {
	if c.authMethod == "app" {
		token, err := c.itr.Token(c.ctx)
		if err != nil {
			return err
		}
		c.auth.Password = token
	}
	return nil
}

// NewClient creates a new git client.
func NewClient(opts *ClientOpts) (*Client, error) {
	client := Client{
		authorName:  opts.AuthorName,
		authorEmail: opts.AuthorEmail,
	}
	token := opts.Token
	if token == "" {
		client.authMethod = "app"
		client.ctx = context.Background()
		itr, err := ghinstallation.NewKeyFromFile(net.DefaultTransport, opts.AppId, opts.AppInstallationId, opts.AppKey)
		if err != nil {
			return nil, err
		}
		client.itr = itr
		token, err = itr.Token(client.ctx)
		if err != nil {
			return nil, err
		}
	} else {
		client.authMethod = "token"
	}
	client.auth = &http.BasicAuth{
		Username: "gitops-actions",
		Password: token,
	}

	return &client, nil
}
