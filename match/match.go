package match

import "github.com/bisgardo/dupe-nukem/scan"

// Matches is a map from hash to a set of files from the target dirs.
// The reason for using a set rather than a slice is that duplicate files from the source would
// The matching files are represented as a set to deduplicate insertions cased by duplicate source files.
type Matches map[uint64]FileSet

func innerBuildMatches(srcDir *scan.Dir, targets []Index, res Matches) {
	for _, f := range srcDir.Files {
		// TODO If res[f.Hash] is non-nil that means that 'f' a is duplicate. So we can just 'continue'? And use slice instead of set?!

		// Find matches in targets for file by hash.
		var matches []*File
		for _, t := range targets {
			if ms, ok := t[f.Hash]; ok {
				matches = append(matches, ms...)
			}
		}
		// Insert matches into index.
		if len(matches) > 0 {
			// Get or create FileSet for the given hash.
			r := res[f.Hash]
			if r == nil {
				r = make(FileSet)
				res[f.Hash] = r
			}
			// Insert matches into FileSet.
			for _, m := range matches {
				r[m] = struct{}{}
			}
		}
	}
	for _, d := range srcDir.Dirs {
		innerBuildMatches(d, targets, res)
	}
}

func BuildMatches(srcRoot *scan.Dir, targets []Index) Matches {
	res := make(Matches)
	innerBuildMatches(srcRoot, targets, res)
	return res
}
