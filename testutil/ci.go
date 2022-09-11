package testutil

import "os"

// CI tries to detect if the test is being run on a CI server.
// If so, a string identifying the CI is returned. The current implementation reports as follows:
// - GitHub Actions: "github"
// - No CI: ""
func CI() string {
	if _, ok := os.LookupEnv("GITHUB_WORKFLOW"); ok {
		return "github"
	}
	return ""
}
