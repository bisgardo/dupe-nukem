package match

import (
	"fmt"

	"github.com/bisgardo/dupe-nukem/scan"
)

type key struct {
	size int64
	hash uint64
}

func (k key) String() string {
	return fmt.Sprintf("%d,%d", k.size, k.hash)
}

// Matches is a map from source file hash to the target files whose size match and contents hash to this value.
// Note that matching is performed on only on hash values even though we could compare file sizes also.
// This is deemed good enough until hash collisions are shown to be a problem.
type Matches map[key][]Match

type Match struct {
	TargetIndex int
	File        *File
}

// BuildMatch merges the target indexes on the key set defined as the set of hashes of all files in srcRoot.
func BuildMatch(srcRoot *scan.Dir, targets []Target) Matches {
	res := make(Matches)
	innerBuildMatch(srcRoot, targets, res)
	return res
}

func innerBuildMatch(srcDir *scan.Dir, targets []Target, res Matches) {
	for _, file := range srcDir.Files {
		k := key{size: file.Size, hash: file.Hash}
		if _, ok := res[k]; ok {
			// File is a duplicate; has already been matched.
			continue
		}

		// Insert any matches into index.
		// Non-matches are not inserted as that just increases memory usage without
		// adding any information.
		// It does circumvent the duplication check above when a file has no matches,
		// which adds a little redundancy for duplicate files with no matches.
		// This could affect performance in pathological cases, but not correctness.
		if m := findMatches(targets, file.Hash); len(m) > 0 {
			res[k] = m
		}
	}
	for _, d := range srcDir.Dirs {
		innerBuildMatch(d, targets, res)
	}
}

// findMatches finds matches in targets for a file by its hash.
func findMatches(targets []Target, hash uint64) []Match {
	var res []Match
	for targetIdx, t := range targets {
		if matchingFiles, ok := t.Index[hash]; ok {
			for _, f := range matchingFiles {
				res = append(res, Match{
					TargetIndex: targetIdx,
					File:        f,
				})
			}
		}
	}
	return res
}
