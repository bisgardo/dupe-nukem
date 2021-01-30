package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{Use: "dupe-nukem", SilenceUsage: true}
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan directory and dump result as JSON.",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			dir, err := flags.GetString("dir")
			if err != nil {
				return err
			}
			skipDirs, err := flags.GetString("skip")
			if err != nil {
				return err
			}
			cacheFile, err := flags.GetString("cache")
			if err != nil {
				return err
			}

			res, err := Scan(dir, skipDirs, cacheFile)
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
	flags.String("dir", "", "Directory to scan.")
	flags.String("skip", "", "Comma-separated list of dirs to skip.")
	flags.String("cache", "", "File from a previous call to 'scan' to use as hash cache.")

	rootCmd.AddCommand(scanCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
