package testutil

import (
	"log"
	"os"
	"runtime"
)

// CI attempts to detect if the process is being run on a CI server.
// If so, a string identifying the CI is returned. The current implementation reports as follows:
// - GitHub Actions: "github"
// - No CI: ""
func CI() string {
	if e, ok := os.LookupEnv("GITHUB_WORKFLOW"); ok {
		//return "github"
		return e // TODO: for debugging; revert
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
	// From ''.
	f, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	err = f.Close()
	log.Println(err) // TODO: for debugging; remove
	return true
}
