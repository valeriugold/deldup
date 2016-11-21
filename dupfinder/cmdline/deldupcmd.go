// Run deldup form command line
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/valeriugold/deldup/dupfinder"
	//"text/template"
	// "html/template"
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

	dups := dupfinder.GetDups(&roots, &exclude, cacheFileName)

	fmt.Println("\n\nUnsorted\n\n")
	printDuplicates(&dups)

	fmt.Println("\n\nSortBySize\n\n")
	dups.SortCustom(dupfinder.SortBySize)
	printDuplicates(&dups)
	fmt.Println("\n\nSortByName\n\n")
	dups.SortCustom(dupfinder.SortByName)
	printDuplicates(&dups)
	fmt.Println("\n\nSortBySiblingsCount\n\n")
	dups.SortCustom(dupfinder.SortBySiblingsCount)
	printDuplicates(&dups)

	// printDirFilesTabbed(tbl)
	// printDirFilesHtmlTable(tbl)
	// out := GetHtmlTableFromGroups(&dups)
}

func printDuplicates(dups *dupfinder.Groups) {
	fmt.Println("here are duplicates")
	// print tabbed
	printDirFilesTabbed(dups)
	return
	for _, fg := range *dups {
		if len(fg) == 0 || fg[0].Stats.Size() == 0 {
			continue
		}
		fmt.Printf("l:%d: s:%v\n", fg[0].Stats.Size(), fg[0].Md5sum)
		for _, f := range fg {
			fmt.Printf("        %s, %d\n", f.FullName, f.SiblingsCount)
		}
	}
}

func printDirFilesTabbed(dups *dupfinder.Groups) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	for _, fg := range *dups {
		if len(fg) == 0 || fg[0].Stats.Size() == 0 {
			continue
		}
		// fmt.Fprintf(w, "dir=%s size=%d rows=%d \t \n", filepath.Dir(fg[0].FullName), fg[0].Stats.Size(), len(fg))
		fmt.Fprintf(w, "%9d \t", fg[0].Stats.Size())
		for _, f := range fg {
			fmt.Fprintf(w, "%s\t", f.FullName)
		}
		fmt.Fprintf(w, "\n")
	}
	w.Flush()
}
