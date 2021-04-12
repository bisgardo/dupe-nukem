module github.com/bisgardo/dupe-nukem

go 1.12

require (
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.3.0
)

// Hack to avoid the insane dependency explosion caused by this dependency of cobra.
// Should be removed once cobra has gotten their dependencies under control.
replace github.com/spf13/viper => ./dummy/viper
