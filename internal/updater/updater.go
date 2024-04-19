package updater

import (
	"fmt"
	"gitops-actions/internal/config"
)

func UpdateFiles(targetFiles []config.TargetFile, basePath, value string) error {
	for _, tf := range targetFiles {
		path := fmt.Sprintf("%s/%s", basePath, tf.Path)
		switch tf.Replacer {
		case "regex":
			err := RegexReplace(path, tf.Regex.Pattern, tf.Regex.Tmpl, value)
			if err != nil {
				return err
			}
		case "yaml":
			err := UpdateYaml(path, tf.Key, value)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid replacer: %s", tf.Replacer)
		}
	}
	return nil
}
