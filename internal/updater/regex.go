package updater

import (
	"fmt"
	"log"
	"os"
	"regexp"
)

func RegexReplace(path, pattern, tmpl, value string) error {
	contentByte, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("error opening file: %s", err)
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	result := regex.ReplaceAllString(string(contentByte), fmt.Sprintf("%s%s", tmpl, value))

	err = os.WriteFile(path, []byte(result), 0644)
	if err != nil {
		log.Fatalf("error writing to file: %s", err)
	}

	return nil
}
