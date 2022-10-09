package match

import "github.com/bisgardo/dupe-nukem/scan"

// Matches is a map from source file hash to the target files whose contents hash to this value.
// The fact that the type is identical to Index is coincidental, so their definitions are kept separate.
// Note that matching is performed on only on hash values even though we could compare file sizes also.
// This is deemed good enough until hash collisions are shown to be a problem.
type Matches map[uint64][]*File

func innerBuildMatch(srcDir *scan.Dir, targets []Index, res Matches) {
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
		// This could affect performance in pathological cases, but not correctness.
		if len(matches) > 0 {
			res[file.Hash] = matches
		}
	}
	for _, d := range srcDir.Dirs {
		innerBuildMatch(d, targets, res)
	}
}

// BuildMatch merges the target indexes on the key set defined as the set of hashes of all files in srcRoot.
func BuildMatch(srcRoot *scan.Dir, targets []Index) Matches {
	res := make(Matches)
	innerBuildMatch(srcRoot, targets, res)
	return res
}
