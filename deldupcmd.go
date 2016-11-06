// Run deldup form command line
package main

import (
	"github.com/valeriugold/deldup/deldup"
	"flag"

	"fmt"
	"strings"
	// "text/tabwriter"
	//"text/template"
	"html/template"
)

var cmdLineDirsIn = flag.String("dirs", ".", "search for duplicates in these dirs")
var cmdLineDirsExclude = flag.String("exclude", "", "omit seraching in these dirs")

func main() {
	// get arguments
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Parse the given dir list and and print duplicates\n")
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	roots := strings.Split(*cmdLineDirsIn, ",")
	if len(roots) == 0 {
		roots = []string{"."}
	}
	exclude := make(map[string]bool)
	for _, s := range strings.Split(*cmdLineDirsExclude, ",") {
		exclude[s] = true
	}

	dirCurrent, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		// log.Fatal(err)
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	cacheFileName := dirCurrent + "/deldup.cache"

	dups := GetDups(&roots, &exclude, cacheFileName)

	fmt.Println("\n\nUnsorted\n\n")
	printDuplicates(&dups)
	
	fmt.Println("\n\nSortBySize\n\n")
	dups.SortCustom(SortBySize)
	printDuplicates(&dups)
	fmt.Println("\n\nSortByName\n\n")
	dups.SortCustom(SortByName)
	printDuplicates(&dups)
	fmt.Println("\n\nSortBySiblingsCount\n\n")
	dups.SortCustom(SortBySiblingsCount)
	printDuplicates(&dups)

	// printDirFilesTabbed(tbl)
	// printDirFilesHtmlTable(tbl)
	out := GetHtmlTableFromGroups(&dups)
	fmt.Printf("%s\n", out)
}

func printDuplicates(dups *Groups) {
	fmt.Println("here are duplicates")
	for _, fg := range *dups {
		if len(fg) == 0 || fg[0].Stats.Size() == 0 {
			continue
		}
		fmt.Printf("l:%d: s:%v\n", fg[0].Stats.Size(), fg[0].Md5sum)
		for _, f := range fg {
			fmt.Printf("        %s\n", f.FullName)
		}
	}
}


// type DirFiles struct {
// 	Dir	string
// 	Dups	[]filesStats
// }
// func printDirFilesTabbed(tbl []DirFiles) {
// 	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, '.', 0)
// 	for _, df := range tbl {
// 		fmt.Fprintf(w, "dir=%s rows=%d \t same\n", df.Dir, len(df.Dups))
// 		for _, row := range df.Dups {
// 			for _, f := range row {
// 				fmt.Fprintf(w, "  %s\t", f.FullName)
// 			}
// 			fmt.Fprintf(w, "\n")
// 		}
// 	}
// 	w.Flush()
// }
