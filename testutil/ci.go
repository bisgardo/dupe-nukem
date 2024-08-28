package testutil

import (
	"os"
	"runtime"
)

// CI attempts to detect if the process is being run on a CI server.
// If so, a string identifying the CI is returned. The current implementation reports as follows:
// - GitHub Actions: "github"
// - No (or other) CI: ""
func CI() string {
	if _, ok := os.LookupEnv("GITHUB_WORKFLOW"); ok {
		// For reference, GITHUB_WORKFLOW contains the name of the workflow.
		return "github"
	}
	return ""
}

// IsWindowsAdministrator attempts to detect if the process is being run on a Windows box with administrator privileges.
// This may be used to control whether a test should be skipped if it for instance needs to create symbolic links.
func IsWindowsAdministrator() bool {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS != "windows" {
		return false
	}
	// From https://gist.github.com/jerblack/d0eb182cc5a1c1d92d92a4c4fcc416c6
	// (see https://gist.github.com/jerblack/d0eb182cc5a1c1d92d92a4c4fcc416c6?permalink_comment_id=4537925#gistcomment-4537925
	// or https://github.com/golang/go/issues/28804#issuecomment-438838144 for alternative).
	f, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
