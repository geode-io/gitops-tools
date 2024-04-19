package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"

	actions "github.com/sethvargo/go-githubactions"
)

type GitOpsConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Spec       struct {
		ConfigRepo  ConfigRepo   `yaml:"configRepo"`
		TargetFiles []TargetFile `yaml:"targetFiles"`
		Deployments []Deployment `yaml:"deployments"`
	} `yaml:"spec"`
}

type ConfigRepo struct {
	Owner         string `yaml:"owner"`
	Repo          string `yaml:"repo"`
	AppPathPrefix string `yaml:"appPathPrefix"`
	App           string `yaml:"app"`
}

type TargetFile struct {
	Path     string `yaml:"path"`
	Replacer string `yaml:"replacer"`
	Key      string `yaml:"key"`
	Regex    struct {
		Pattern string `yaml:"pattern"`
		Tmpl    string `yaml:"tmpl"`
	} `yaml:"regex"`
}

type Deployment struct {
	SourceBranch string `yaml:"sourceBranch"`
	TargetStack  string `yaml:"targetStack"`
	AutoDeploy   bool   `yaml:"autoDeploy"`
}

func (g *GitOpsConfig) RepoUrl() string {
	return fmt.Sprintf("https://%s/%s/%s", "github.com", g.Spec.ConfigRepo.Owner, g.Spec.ConfigRepo.Repo)
}

func (g *GitOpsConfig) Validate() error {
	if g.Spec.ConfigRepo.Owner == "" {
		return fmt.Errorf("configRepo.owner is required")
	}
	if g.Spec.ConfigRepo.Repo == "" {
		return fmt.Errorf("configRepo.repo is required")
	}
	if g.Spec.ConfigRepo.AppPathPrefix == "" {
		return fmt.Errorf("configRepo.appPathPrefix is required")
	}
	if g.Spec.ConfigRepo.App == "" {
		return fmt.Errorf("configRepo.app is required")
	}
	if len(g.Spec.TargetFiles) == 0 {
		return fmt.Errorf("targetFiles is required")
	}
	if len(g.Spec.Deployments) == 0 {
		return fmt.Errorf("deployments is required")
	}
	return nil
}

func GetConfig(globalConfigPath, appConfigPath, appName string) (*GitOpsConfig, error) {
	globalConfig, err := ReadConfig(globalConfigPath)
	if err != nil && !os.IsNotExist(err) {
		globalConfig = nil
		actions.Warningf("global config file %s not found", globalConfigPath)
	} else if err != nil {
		return nil, err
	}

	appConfig, err := ReadConfig(appConfigPath)
	if err != nil && !os.IsNotExist(err) {
		appConfig = nil
		actions.Warningf("app config file %s not found", appConfigPath)
	} else if err != nil {
		return nil, err
	}
	return FinalizeConfig(globalConfig, appConfig, appName)
}

func ReadConfig(path string) (*GitOpsConfig, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config: %s", err)
	}

	c := GitOpsConfig{}
	err = yaml.Unmarshal(fileBytes, &c)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %s", err)
	}

	return &c, nil
}

func FinalizeConfig(globalConfig, appConfig *GitOpsConfig, appName string) (*GitOpsConfig, error) {
	finalConf := &GitOpsConfig{}
	if globalConfig != nil {
		finalConf = globalConfig
	}
	if appName != "" {
		finalConf.Spec.ConfigRepo.App = appName
	}
	if appConfig != nil {
		if appConfig.Spec.ConfigRepo.Owner != "" {
			finalConf.Spec.ConfigRepo.Owner = appConfig.Spec.ConfigRepo.Owner
		}
		if appConfig.Spec.ConfigRepo.Repo != "" {
			finalConf.Spec.ConfigRepo.Repo = appConfig.Spec.ConfigRepo.Repo
		}
		if appConfig.Spec.ConfigRepo.AppPathPrefix != "" {
			finalConf.Spec.ConfigRepo.AppPathPrefix = appConfig.Spec.ConfigRepo.AppPathPrefix
		}
		if appConfig.Spec.ConfigRepo.App != "" {
			finalConf.Spec.ConfigRepo.App = appConfig.Spec.ConfigRepo.App
		}
		if len(appConfig.Spec.TargetFiles) > 0 {
			finalConf.Spec.TargetFiles = appConfig.Spec.TargetFiles
		}
		if len(appConfig.Spec.Deployments) > 0 {
			finalConf.Spec.Deployments = appConfig.Spec.Deployments
		}
	}
	err := finalConf.Validate()
	if err != nil {
		return nil, err
	}
	return finalConf, nil
}
