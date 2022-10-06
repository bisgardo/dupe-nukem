package match

import "github.com/bisgardo/dupe-nukem/scan"

type Matches map[uint64][]*File

func innerBuildMatches(srcDir *scan.Dir, targets []Index, res Matches) {
	for _, f := range srcDir.Files {
		var matches []*File
		for _, t := range targets {
			if ms, ok := t[f.Hash]; ok {
				matches = append(matches, ms...)
			}
		}
		if len(matches) > 0 {
			res[f.Hash] = append(res[f.Hash], matches...)
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
