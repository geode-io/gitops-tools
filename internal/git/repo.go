package git

import (
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Clone plane clones a git repository.
func (c *Client) Clone(url, path string) (*git.Repository, error) {
	repo, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:      url,
		Auth:     c.auth,
		Progress: nil,
	})
	return repo, err
}

// Checkout checks out a branch in a git repository.
// TODO: Add support checkout from a different source branch.
func (c *Client) Checkout(repo *git.Repository, branch string) error {
	headRef, err := repo.Head()
	if err != nil {
		return err
	}
	branchRefName := plumbing.NewBranchReferenceName(branch)
	branchCoOpts := git.CheckoutOptions{
		Branch: plumbing.ReferenceName(branchRefName),
		Force:  true,
	}
	ref := plumbing.NewHashReference(branchRefName, headRef.Hash())
	if err = repo.Storer.SetReference(ref); err != nil {
		return err
	}
	w, err := repo.Worktree()
	if err != nil {
		return err
	}
	return w.Checkout(&branchCoOpts)
}

func (c *Client) CloneAndCheckout(url, path, branch string) (*git.Repository, error) {
	if err := c.RefreshToken(); err != nil {
		return nil, err
	}
	repo, err := c.Clone(url, path)
	if err != nil {
		return nil, err
	}
	err = c.Checkout(repo, branch)
	return repo, err
}

// CommitAndPush commits and pushes changes to a git repository.
// It returns a boolean indicating whether there was a change to commit and an error if any.
func (c *Client) CommitAndPush(repo *git.Repository, commitMessage string) (bool, error) {
	if err := c.RefreshToken(); err != nil {
		return false, err
	}
	w, err := repo.Worktree()
	if err != nil {
		return false, err
	}

	err = w.AddWithOptions(&git.AddOptions{All: true})
	if err != nil {
		return false, err
	}

	s, err := w.Status()
	if err != nil {
		return false, err
	}
	if s.IsClean() {
		return false, nil
	}

	_, err = w.Commit(commitMessage, &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name:  c.authorName,
			Email: c.authorEmail,
		},
	})
	if err != nil {
		return false, err
	}

	err = repo.Push(&git.PushOptions{
		Auth:       c.auth,
		RemoteName: "origin",
		Force:      true,
	})
	return true, err
}
