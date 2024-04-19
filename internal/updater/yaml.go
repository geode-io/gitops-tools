package updater

import (
	"os"

	"github.com/goccy/go-yaml"
)

func UpdateYaml(configPath, key, value string) error {
	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	ymlConf := make(map[string]interface{})
	err = yaml.NewDecoder(file).Decode(&ymlConf)
	if err != nil {
		return err
	}

	ymlConf[key] = value
	content, err := yaml.Marshal(ymlConf)
	if err != nil {
		return err
	}

	err = os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		return err
	}

	return nil
}
