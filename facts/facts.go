package facts

import (
	"github.com/mitchellh/go-homedir"
)

const applicationName = "gloner"

func GetApplicationName() string {
	return applicationName
}

func GetHomeDirectory() string {
	homeDir, err := homedir.Dir()
	if err != nil {
		return ""
	}

	return homeDir
}
