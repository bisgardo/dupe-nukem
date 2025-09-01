package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bisgardo/dupe-nukem/scan"
	. "github.com/bisgardo/dupe-nukem/testutil/testdata"
)

const contents1 = `x\n`
const contents2 = `y\n`
const contents3 = `z\n`

func simulate(root DirNode, name string) scan.Result {
	return scan.Result{
		TypeVersion: scan.CurrentResultTypeVersion,
		Root:        root.SimulateScan(name),
	}
}

func run(root DirNode, name string) error {
	res := simulate(root, name)
	jsonBytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fmt.Sprintf("%v.json", name), jsonBytes, 0666)
}

func main() {
	name1 := "test1"
	name2 := "test2"

	root1 := DirNode{
		"a": FileNode{C: contents1},
		"b": DirNode{
			"c": FileNode{C: contents1},
		},
		"d": FileNode{C: contents2},
		"e": DirNode{
			"a": FileNode{C: contents3},
			"g": FileNode{},
		},
	}
	root2 := DirNode{
		"a": FileNode{C: contents1},
		"b": DirNode{
			"a": FileNode{C: contents3},
			"c": FileNode{C: contents2},
			"g": FileNode{},
			"x": FileNode{C: contents3},
			"y": FileNode{C: contents3},
			"z": FileNode{C: contents3},
			"k": FileNode{C: contents3},
		},
		"e": DirNode{
			"a": FileNode{C: contents3},
		},
	}

	if err := run(root1, name1); err != nil {
		panic(err)
	}
	if err := run(root2, name2); err != nil {
		panic(err)
	}
}
