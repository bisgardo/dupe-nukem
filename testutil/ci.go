package testutil

import "os"

func CI() string {
    if _, ok := os.LookupEnv("GITHUB_WORKFLOW"); ok {
        return "github"
    }
    return ""
}
