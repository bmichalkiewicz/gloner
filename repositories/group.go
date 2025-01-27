package repositories

import (
	"fmt"
	"regexp"
)

var pattern = `^git@([a-zA-Z0-9.-]+):([a-zA-Z0-9_.-]+)(/[a-zA-Z0-9_./-]*)?\.git$`

type Group struct {
	Name     string    `yaml:"name"`
	Projects []Project `yaml:"projects"`
}

type Project struct {
	URL string `yaml:"url"`
}

func Validate(url string) error {
	matched, err := regexp.MatchString(pattern, url)
	if err != nil {
		return fmt.Errorf("error validating SSH URL: %v", err)
	}
	if !matched {
		return fmt.Errorf("invalid SSH URL provided (example: git@gitlab.com:goodgroup/goodrepo.git)")
	}

	return nil
}

func Decode(url string) (*Group, error) {
	// Validate the SSH URL
	err := Validate(url)
	if err != nil {
		return nil, err
	}

	// Compile the regex and find submatches
	regex := regexp.MustCompile(`^git@([a-zA-Z0-9.-]+):([a-zA-Z0-9_.-]+)(/[a-zA-Z0-9_./-]*)?\.git$`)
	submatches := regex.FindStringSubmatch(url)

	// Ensure the submatches contain the expected groups
	if len(submatches) < 3 {
		return nil, fmt.Errorf("failed to extract groups from SSH URL")
	}

	// Extract the main group (e.g., "goodgroup")
	mainGroup := submatches[2]

	// Return the Group with the extracted name
	return &Group{
		Name: mainGroup, // Main group
		Projects: []Project{
			{
				URL: url,
			},
		},
	}, nil
}
