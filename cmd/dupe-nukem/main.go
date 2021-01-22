package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bisgardo/dupe-nukem/scan"
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
			// TODO Replace '.' with working dir.
			res, err := scan.Run(filepath.Clean(dir))
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

	rootCmd.AddCommand(scanCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
