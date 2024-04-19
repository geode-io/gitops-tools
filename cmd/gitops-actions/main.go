package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	actions "github.com/sethvargo/go-githubactions"

	"gitops-actions/internal/config"
	"gitops-actions/internal/git"
	"gitops-actions/internal/github"
	"gitops-actions/internal/updater"
	"gitops-actions/internal/version"
)

func main() {
	globalConfig := kingpin.Flag("global-config", "Path to the gitops global config file").Envar("GLOBAL_CONFIG").String()
	appName := kingpin.Flag("app-name", "Name of the app. required if app-config is not provided").Envar("APP_NAME").String()
	appConfig := kingpin.Flag("app-config", "Path to the gitops app config file. required if app-name is not provided").Envar("APP_CONFIG").String()
	value := kingpin.Flag("value", "Value to update in the config files").Required().Envar("VALUE").String()
	ghToken := kingpin.Flag("gh-token", "Github Token for git and Github operations").Envar("GH_TOKEN").String()
	ghAppKey := kingpin.Flag("gh-app-key", "Github App Key for Github operations").Envar("GH_APP_KEY").String()
	ghAppId := kingpin.Flag("gh-app-id", "Github App ID for Github operations").Envar("GH_APP_ID").Int64()
	ghAppInstallationId := kingpin.Flag("gh-app-installation-id", "Github App Installation ID for Github operations").Envar("GH_APP_INSTALLATION_ID").Int64()
	gitCommitAuthorName := kingpin.Flag("git-commit-author-name", "Author name for git commit").Default("gitops-actions").Envar("GIT_COMMIT_AUTHOR_NAME").String()
	gitCommitAuthorEmail := kingpin.Flag("git-commit-author-email", "Author email for git commit").Default("gitops-actions@geode.io").Envar("GIT_COMMIT_AUTHOR_EMAIL").String()
	prTitle := kingpin.Flag("pr-title", "Title for the PR in the config repo").Envar("PR_TITLE").String()
	prBody := kingpin.Flag("pr-body", "Body for the PR in the config repo").Envar("PR_BODY").String()
	ver := kingpin.Flag("version", "Print version").Short('v').Bool()
	kingpin.Parse()

	if *ver {
		fmt.Println(version.VersionInfo())
		fmt.Println(version.BuildContext())
		os.Exit(0)
	}

	actions.Infof("Starting gitops-actions ...")
	actions.Group("🔷 Version Info")
	actions.Infof(version.VersionInfo())
	actions.Infof(version.BuildContext())
	actions.EndGroup()

	actions.Group("✅ Initializing")
	actions.Debugf("merging configs: global=%s, gitops=%s", *globalConfig, *appConfig)
	c, err := config.GetConfig(*globalConfig, *appConfig, *appName)
	if err != nil {
		actions.Fatalf("error getting config: %s", err.Error())
	}

	actions.Infof("initializing git client ...")
	git, err := git.NewClient(&git.ClientOpts{
		Token:             *ghToken,
		AppKey:            *ghAppKey,
		AppId:             *ghAppId,
		AppInstallationId: *ghAppInstallationId,
		AuthorName:        *gitCommitAuthorName,
		AuthorEmail:       *gitCommitAuthorEmail,
	})
	if err != nil {
		actions.Fatalf("error creating git client: %s", err.Error())
	}

	actions.Infof("initializing github client ...")
	gh, err := github.NewClient(&github.ClientOpts{
		Token:             *ghToken,
		AppKey:            *ghAppKey,
		AppId:             *ghAppId,
		AppInstallationId: *ghAppInstallationId,
	})
	if err != nil {
		actions.Fatalf("error creating github client: %s", err.Error())
	}
	actions.EndGroup()

	for _, d := range c.Spec.Deployments {
		actions.Group(fmt.Sprintf("🚀 Deployment: %s", d.TargetStack))
		actions.Infof("Starting the deployment process")
		defer actions.EndGroup()
		clonePath, err := os.MkdirTemp("", "gitops-actions-*")
		if err != nil {
			actions.Fatalf("error creating temp directory: %s", err.Error())
		}
		gitOpsRepo := c.RepoUrl()
		branchName := fmt.Sprintf("%s/%s", c.Spec.ConfigRepo.App, d.TargetStack)
		appPath := fmt.Sprintf("%s/%s/%s/%s", clonePath, c.Spec.ConfigRepo.AppPathPrefix, c.Spec.ConfigRepo.App, d.TargetStack)

		actions.Infof("cloning repo: %s", gitOpsRepo)
		repo, err := git.CloneAndCheckout(gitOpsRepo, clonePath, branchName)
		if err != nil {
			actions.Fatalf("error cloning and checking out repo: %s", err.Error())
		}

		actions.Infof("updating files in %s path", appPath)
		err = updater.UpdateFiles(c.Spec.TargetFiles, appPath, *value)
		if err != nil {
			actions.Fatalf("error updating files: %s", err.Error())
		}

		actions.Infof("committing and pushing changes ...")
		hadChanges, err := git.CommitAndPush(repo, fmt.Sprintf("automated commit to update tag to %s", *value))
		if err != nil && !strings.Contains(err.Error(), "already up-to-date") {
			hadChanges = false
			actions.Infof("branch is already up-to-date, skipping ...")
		}
		if !hadChanges {
			actions.Infof("no changes to commit, skipping PR creation and deployment ...")
			err = os.RemoveAll(clonePath)
			if err != nil {
				actions.Warningf("error removing directory: %s", err.Error())
			}
			continue
		}

		actions.Debugf("cleaning up temp directory ...")
		err = os.RemoveAll(clonePath)
		if err != nil {
			actions.Warningf("error removing directory: %s", err.Error())
		}

		prTitle := *prTitle
		if prTitle == "" {
			prTitle = fmt.Sprintf("[CI] Automated PR to update %s", branchName)
		}
		prBody := *prBody
		if prBody == "" {
			prBody = fmt.Sprintf("Automated PR to %s with the new value", branchName)
		}
		actions.Infof("creating PR ...")
		pr, err := gh.CreatePR(
			c.Spec.ConfigRepo.Owner, c.Spec.ConfigRepo.Repo,
			branchName, d.SourceBranch,
			prTitle, prBody,
		)
		if err != nil && strings.Contains(err.Error(), "pull request already exists") {
			actions.Infof("PR already exists, skipping ...")
			if d.AutoDeploy {
				actions.Infof("Merge and deploy PR ...")
				pr, err := gh.GetPR(c.Spec.ConfigRepo.Owner, c.Spec.ConfigRepo.Repo, branchName)
				if err != nil {
					actions.Fatalf("error getting PR: %s", err.Error())
				}
				err = gh.Deploy(pr)
				if err != nil {
					actions.Fatalf("error deploying: %s. aborting ...", err.Error())
				}
			}
		} else if err != nil {
			actions.Fatalf("error creating PR: %s", err.Error())
		} else {
			actions.Infof("PR created: %s", pr.GetHTMLURL())
			if d.AutoDeploy {
				err = gh.Deploy(pr)
				if err != nil {
					actions.Fatalf("error deploying: %s. aborting ...", err.Error())
				}
				actions.Infof("PR deployed: %s\n", pr.GetHTMLURL())
			}
		}
	}
}
