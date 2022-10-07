package match

import "github.com/bisgardo/dupe-nukem/scan"

func innerBuildMatchIndex(srcDir *scan.Dir, targets []Index, res Index) {
	for _, file := range srcDir.Files {
		matches, ok := res[file.Hash]
		if ok {
			// File is a duplicate; has already been matched.
			continue
		}
		// Find matches in targets for file by its hash.
		for _, t := range targets {
			if ms, ok := t[file.Hash]; ok {
				matches = append(matches, ms...)
			}
		}
		// Insert any matches into index.
		// Non-matches are not inserted as that just increases memory usage without
		// adding any information.
		// It does circumvent the duplication check above when a file has no matches,
		// which adds a little redundancy for duplicate files with no matches.
		// This is not expected to be a problem.
		if len(matches) > 0 {
			res[file.Hash] = matches
		}
	}
	for _, d := range srcDir.Dirs {
		innerBuildMatchIndex(d, targets, res)
	}
}

// BuildMatchIndex merges the target indexes on the key set defined as the set of hashes of all files in srcRoot.
func BuildMatchIndex(srcRoot *scan.Dir, targets []Index) Index {
	res := make(Index)
	innerBuildMatchIndex(srcRoot, targets, res)
	return res
}
