package match

import "github.com/bisgardo/dupe-nukem/scan"

//goland:noinspection GoSnakeCaseUsage
var (
	testdata_x_a = scan.NewFile("a", 2, 620331299357648818) // contents: "a"
	testdata_x_b = scan.NewFile("b", 2, 623218616892763229) // contents: "b"
	testdata_x_c = scan.NewFile("c", 2, 622257643729896040) // contents: "c"
	testdata_y_a = scan.NewFile("a", 2, 620331299357648818) // contents: "a"
	testdata_y_b = scan.NewFile("b", 2, 620331299357648818) // contents: "a"
	testdata_y_c = scan.NewFile("c", 2, 617474768148124315) // contents: "d"

	testdata_x = &scan.Dir{Name: "x", Files: []*scan.File{testdata_x_a, testdata_x_b, testdata_x_c}}
	testdata_y = &scan.Dir{Name: "y", Files: []*scan.File{testdata_y_a, testdata_y_b, testdata_y_c}}
)
