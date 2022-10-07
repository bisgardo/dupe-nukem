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
	hashCmd := &cobra.Command{
		Use:   "hash",
		Short: "Compute the FNV-1a hash of the contents of the file at the provided path or stdin if none was provided",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			file, err := flags.GetString("file")
			if err != nil {
				return err
			}
			res, err := Hash(file)
			if err != nil {
				return err
			}
			fmt.Println(res)
			return nil
		},
	}
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
	scanFlags := scanCmd.Flags()
	scanFlags.String("dir", "", "directory to scan")
	scanFlags.String("skip", "", "comma-separated list of directories to skip")
	scanFlags.String("cache", "", "file from a previous call to 'scan' to use as hash cache")

	hashFlags := hashCmd.Flags()
	hashFlags.String("file", "", "file to hash")

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(hashCmd)
	if err := rootCmd.Execute(); err != nil {
		// Print error with stack trace.
		log.Fatalf("error: %+v\n", err)
	}
}
