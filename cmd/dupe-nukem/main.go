package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/cobra"
)

const outFileCreatePermissions = 0755

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
			// Extract flags.
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
			outFile, err := flags.GetString("out")
			if err != nil {
				return err
			}
			if outFile != "" && outFile == cacheFile {
				log.Printf("info: scan result will be written back to cache file %q\n", outFile)
			}

			// Run command.
			res, err := Scan(dir, skipExpr, cacheFile)
			if err != nil {
				return err
			}

			// Output result.
			bs, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			f := os.Stdout
			if outFile != "" {
				f, err = openOutputFile(outFile, err)
				if err != nil {
					return err
				}
				defer func() {
					if err := f.Close(); err != nil {
						log.Printf("error closing output file %q: %v\n", f.Name(), err)
					}
				}()
				log.Printf("info: writing scan result to file %q\n", f.Name())
			}
			_, err = fmt.Fprintln(f, string(bs))
			return err
		},
	}
	flags := scanCmd.Flags()
	flags.String("dir", "", "directory to scan")
	flags.String("skip", "", "comma-separated list of directories to skip")
	flags.String("cache", "", "file from a previous call to 'scan' to use as hash cache")
	flags.String("out", "", "file to which the result of 'scan' should be written")

	rootCmd.AddCommand(scanCmd)
	if err := rootCmd.Execute(); err != nil {
		// Print error with stack trace.
		log.Fatalf("error: %+v\n", err)
	}
}

func openOutputFile(outFile string, err error) (*os.File, error) {
	f, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, outFileCreatePermissions)
	if err != nil {
		log.Printf("cannot open output file %q (error: %v) - creating temporary file instead\n", outFile, err)
		return ioutil.TempFile("", "dupe-nukem-scan")
	}
	return f, nil
}
