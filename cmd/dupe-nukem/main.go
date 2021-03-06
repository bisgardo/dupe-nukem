package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

func main() {
	// ANNOYANCE The description of cobra's default help command is upper case and cannot be changed
	//           without doing the whole command ourselves (inconsistently, flags are lower case!).
	//           So for now we follow the same convention.
	//           Consider vendoring or finding a replacement for this library to fix this
	//           and also get rid of all of its irrelevant dependencies.
	// IDEA Require output file as a parameter rather than just using stdout.
	//      Use the filename or an additional flag to add compression.
	//      Use another flag to specify encryption password.
	//      Also output a file with a cryptographic hash of the data structure (or include in the file?).
	rootCmd := &cobra.Command{Use: "dupe-nukem", SilenceUsage: true, SilenceErrors: true}
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan directory and dump result as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			dir, err := flags.GetString("dir")
			if err != nil {
				return err
			}
			skipExpr, err := flags.GetString("skip")
			if err != nil {
				return err
			}
			cacheFile, err := flags.GetString("cache")
			if err != nil {
				return err
			}

			res, err := Scan(dir, skipExpr, cacheFile)
			if err != nil {
				return err
			}

			bs, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(bs))
			return err
		},
	}
	flags := scanCmd.Flags()
	flags.String("dir", "", "directory to scan")
	flags.String("skip", "", "comma-separated list of directories to skip")
	flags.String("cache", "", "file from a previous call to 'scan' to use as hash cache")

	rootCmd.AddCommand(scanCmd)
	if err := rootCmd.Execute(); err != nil {
		// Print error with stack trace.
		log.Fatalf("error: %+v\n", err)
	}
}
