// Delete duplicates from given directory list
package dupfinder

import (
	"bufio"
	"crypto/md5"
	// "encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	// "flag"
	"os"
	"path/filepath"
	"sort"
	"sync"
)


type FileStats struct {
	FullName	string
	Stats		os.FileInfo
	Md5sum		[md5.Size]byte
	SiblingsCount	int	// no of duplicate files in the same directory
}

func (st *FileStats) String() string {
	return fmt.Sprintf("fn:%s name:%s size:%d modtime:%s",
		st.FullName, st.Stats.Name(), st.Stats.Size(),
		st.Stats.ModTime().Format("2006-01-02 15:04:05"))
}


// // list of FileStats pointers
// type filesStats	[]*FileStats
// // allow sort by FullName
// type byFullName filesStats
// func (x byFullName) Len() int           { return len(x) }
// func (x byFullName) Less(i, j int) bool { return x[i].FullName < x[j].FullName }
// func (x byFullName) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }


type FilesGroup	[]*FileStats
type Groups	[]FilesGroup

const (
	SortBySize	    = iota
	SortByName	    = iota
	SortBySiblingsCount  = iota
)


// use tokens to limit gofuncs
var tokens = make(chan struct{}, 20)

func GetDups(roots *([]string), exclude *(map[string]bool), cacheFileName string) (Groups) {
	
	// divide the files per their size
	allFiles := parseDirStructure(roots, exclude)
	
	// // just for debug print lenToNames
	// for _, v := range lenToNames {
	// 	fmt.Printf("elements: ")
	// 	for _, x := range v {
	// 		fmt.Printf(" %v, ex: %v", x, x.FullName)
	// 	}
	// 	fmt.Printf("\n")
	// }
	// fmt.Println("add md5sums")
	fmt.Println("allFiles=%d", len(allFiles))
	
	// group files with same length
	sizeGroup := groupFilesWithSameSize(&allFiles)
	fmt.Println(len(sizeGroup))
	
	// get md5sum for all potential duplicate files
	fillMd5SumField(&sizeGroup, cacheFileName)
	fmt.Println(len(sizeGroup))
	
	// get Groups of slices with same duplicate files
	var dups Groups
	for _, fg := range sizeGroup {
		addDuplicateGroups(fg, &dups)
	}
	fmt.Println(len(dups))
	
	fillSiblingsCount(&dups)

	return dups
	// fmt.Println("\n\nUnsorted\n\n")
	// printDuplicates(&dups)
	
	// fmt.Println("\n\nSortBySize\n\n")
	// dups.SortCustom(SortBySize)
	// printDuplicates(&dups)
	// fmt.Println("\n\nSortByName\n\n")
	// dups.SortCustom(SortByName)
	// printDuplicates(&dups)
	// fmt.Println("\n\nSortBySiblingsCount\n\n")
	// dups.SortCustom(SortBySiblingsCount)
	// printDuplicates(&dups)

	// // printDirFilesTabbed(tbl)
	// // printDirFilesHtmlTable(tbl)
	// out := GetHtmlTableFromGroups(&dups)
	// fmt.Printf("%s\n", out)
}



// parse dir structure
func parseDirStructure(roots *([]string), exclude *(map[string]bool)) (FilesGroup)  {
	reportFiles := make(chan *FileStats)
	var n sync.WaitGroup
	for _, dir := range *roots {
		n.Add(1)
		go addDir(dir, &n, reportFiles, exclude)
	}
	go func() {
		n.Wait()
		close(reportFiles)
	}()
	var fg FilesGroup
	for fs := range reportFiles {
		if fs.Stats.IsDir() || fs.Stats.Name() == ".DS_Store" {
			continue
		}
		fg = append(fg, fs)
	}
	return fg
}

func addDir(dir string, n *sync.WaitGroup, reportFiles chan<- *FileStats, exclude *map[string]bool) {
	defer n.Done()
	tokens <- struct{}{}
	defer func() { <- tokens }()
	if _, ok := (*exclude)[dir]; ok {
		return
	}
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "addDir: %v\n", err)
		return
	}
	for _, entry := range entries {
		FullName := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			n.Add(1)
			go addDir(FullName, n, reportFiles, exclude)
		}
		reportFiles <- &FileStats{FullName, entry, [md5.Size]byte{}, 0}
	}
}

type bySize FilesGroup
func (x bySize) Len() int           { return len(x) }
func (x bySize) Less(i, j int) bool { return x[i].Stats.Size() < x[j].Stats.Size() }
func (x bySize) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

func groupFilesWithSameSize(all *FilesGroup) (Groups) {
	var g Groups
	if len(*all) == 0 {
		return g
	}
	sort.Sort(bySize(*all))
	var fg FilesGroup
	for i := 1; i < (len(*all)); i++ {
		if (*all)[i - 1].Stats.Size() == (*all)[i].Stats.Size()  {
			if len(fg) == 0 {
				fg = append(fg, (*all)[i - 1])
			}
			fg = append(fg, (*all)[i])
		} else {
			if len(fg) > 1 {
				tmp := make(FilesGroup, len(fg))
				copy(tmp, fg)
				g = append(g, tmp)
				fg = fg[:0]
			}
		}
	}
	if len(fg) > 1 {
		g = append(g, fg)
	}
	return g
}

// get md5sum for all potential duplicate files
// modify lenToNames
func fillMd5SumField(g *Groups, cacheFileName string) {
	type cachedMd5Sum struct {
		md5sum	    [md5.Size]byte
		fullName    string
	}

	// load known md5sum values from cache
	map5 := func(path string) (map[string][md5.Size]byte) {
		map5 := make(map[string][md5.Size]byte)
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Open %s: %v\n", path, err)
			return map5
		}
		defer f.Close()
		input := bufio.NewScanner(f)
		for input.Scan() {
			var name string
			s := make([]byte, md5.Size, md5.Size)
			fmt.Sscanf(input.Text(), "%x, %s", &s, &name)
			var sum [md5.Size]byte
			copy(sum[:],s)
			// fmt.Printf("inpt=%s, s=%x, sum=%x, name=%s\n", input.Text(), s, sum, name)
			map5[name] = sum
		}
		return map5
	}(cacheFileName)
	// fmt.Println(map5)

	chSaveMd5Sums := make(chan cachedMd5Sum)
	// save md5sums and names from chSaveMd5Sums channel to file path
	go func(path string, chSave <-chan cachedMd5Sum) {
		file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			fmt.Printf("err on open path=%s: %v\n", path, err)
			os.Exit(1)
		}
		defer file.Close()
		for x := range chSave {
			s := fmt.Sprintf("%x, %s\n", x.md5sum, x.fullName)
			file.Write([]byte(s))
			file.Sync()
		}
	}(cacheFileName, chSaveMd5Sums)
	
	var n2 sync.WaitGroup
	// loop only for lenghts that fit multiple files, possible duplicates
	for _, fg := range *g {
		if len(fg) == 0 || fg[0].Stats.Size() == 0 {
			continue
		}
		for _, f := range fg {
			if n, ok := map5[f.FullName]; ok {
				// fmt.Printf("_load md5 form cache %s\n", f.FullName)
				f.Md5sum = n
			// } else if !f.Stats.IsDir() && f.Stats.Name() != ".DS_Store" {
			} else {
				n2.Add(1)
				go func(fs *FileStats) {
					defer n2.Done()
					tokens <- struct{}{}
					defer func() { <- tokens }()
					fs.Md5sum, _ = getMd5FromFile(fs.FullName)
					chSaveMd5Sums <- cachedMd5Sum{fs.Md5sum, fs.FullName}
				}(f)
			}
		}
	}
	n2.Wait()
	close(chSaveMd5Sums)
	return
}

// return map of [md5sum] -> filesStats
func addDuplicateGroups(files FilesGroup, dups *Groups) {
	m5 := make(map[[md5.Size]byte]FilesGroup)
	for _, f := range files {
		m5[f.Md5sum] = append(m5[f.Md5sum], f)
	}
	// remove md5sums and files that don't have duplicates in m5
	for _, v := range m5 {
		if len(v) > 1 {
			*dups = append(*dups, v)
		}
	}
}

func getCommonDir(sf []string) (d string, sb []string) {
	var i int = 0
Out:
	for {
		if i >= len(sf[0]) {
			break Out
		}
		c := sf[0][i]
		for _, s := range sf {
			if i >= len(s) || c != s[i] {
				break Out
			}
		}
		i++
	}
	var j int

	for j = i - 1; j >= 0; j-- {
		if sf[0][j] == '/' {
			break
		}
	}
	j++
	d = sf[0][:j]
	for _, s := range sf {
		sb = append(sb, s[j:])
		// fmt.Printf("%s ,", s[j:])
	}
	return
}

func getMd5FromFile(s string) ([md5.Size]byte, error) {
	fmt.Printf("md5 for %s\n", s)
	// write md5sum to a cache, so you don't have to compute it every time
	// cn := strings.Replace(s, "/", "_slash_", 100)
	hash := md5.New()

	file, err := os.Open(s)
	if err != nil {
		fmt.Printf("err %v", err)
		os.Exit(2)
		// return returnMD5String, err
	}
	defer file.Close()
	//Copy the file in the hash interface and check for any error
	if _, err := io.Copy(hash, file); err != nil {
		return [md5.Size]byte{}, err
	}
	// hashInBytes := hash.Sum(nil)[:16]
	h := hash.Sum(nil)
	// var a [md5.Size]byte = [md5.Size]byte{h[:md5.Size]}
	var a [md5.Size]byte
	copy(a[:], h)
	return a, nil
	// return hash.Sum(nil)[:md5.Size], nil
}

func fillSiblingsCount(dups *Groups) {
	dirs := make(map[string]int)
	// calculate SiblingsCount per dir
	for _, fg := range *dups {
		if len(fg) == 0 || fg[0].Stats.Size() == 0 {
			continue
		}
		for _, f := range fg {
			d := filepath.Dir(f.FullName)
			dirs[d]++
		}
	}
	// update files with their directories SiblingsCount
	for _, fg := range *dups {
		if len(fg) == 0 || fg[0].Stats.Size() == 0 {
			continue
		}
		for _, f := range fg {
			d := filepath.Dir(f.FullName)
			f.SiblingsCount = dirs[d]
		}
	}
}

// func GetHtmlTableFromGroups(g *Groups) {
// 	var out []byte
// 	funcMap := template.FuncMap{
// 		"length": func(x FilesGroup) int { return len(x) },
// 		"dir": func(x FilesGroup) string { return filepath.Dir(x[0].FullName) },
// 	}
// // 	const ctfn = `{{range .}}dir={{dir .}}     x     rows={{length .}}
// // {{range .}} ### FulName={{.FullName}}, next is !!!
// // {{end}}{{end}}`
// // 	t := template.Must(template.New("dupfiles").Funcs(funcMap).Parse(ctfn))
// // 	fmt.Println("templatesssssssssssssssssssssssssssss:")
// // 	err := t.Execute(os.Stdout, g)
// // 	if err != nil {
// // 		fmt.Println("executing template:", err)
// // 		os.Exit(1)
// // 	}

// 	const cht = `<table border=3>{{range .}}<tr><td>dir={{dir .}}     </td><td>     rows={{length .}}</td></tr>
// <tr>{{range .}}<td>{{.FullName}} </td>{{end}}</tr>
// {{end}}</table>
// `
// 	th := template.Must(template.New("dupfiles2").Funcs(funcMap).Parse(cht))
// 	fmt.Println("templatesssssssssssssssssssssssssssss htttttttmmmmmmmmmllllllllllllllllllll:")
// 	// err2 := th.Execute(os.Stdout, g)
// 	err2 := th.Execute(os.Stdout, g)
// 	if err2 != nil {
// 		fmt.Println("executing template:", err2)
// 		os.Exit(1)
// 	}
// }


// func printDirFilesHtmlTable(tbl []DirFiles) {
// 	funcMap := template.FuncMap{
// 		"length": func(x []filesStats) int { return len(x) },
// 	}
// 	const ctfn = `{{range .}}dir={{.Dir}}     x     rows={{length .Dups}}
// {{range .Dups}} ### {{range .}}FulName={{.FullName}}, next is {{end}} !!!
// {{end}}{{end}}`
// 	t := template.Must(template.New("dupfiles").Funcs(funcMap).Parse(ctfn))
// 	fmt.Println("templatesssssssssssssssssssssssssssss:")
// 	err := t.Execute(os.Stdout, tbl)
// 	if err != nil {
// 		fmt.Println("executing template:", err)
// 		os.Exit(1)
// 	}
// 
// 	const cht = `<table>{{range .}}<tr><td>dir={{.Dir}}     </td><td>     rows={{length .Dups}}</td></tr>
// {{range .Dups}}<tr>{{range .}}
//   <td>{{.FullName}} </td> {{end}}
// </tr>
// {{end}}{{end}}</table>
// `
// 	th := template.Must(template.New("dupfiles2").Funcs(funcMap).Parse(cht))
// 	fmt.Println("templatesssssssssssssssssssssssssssss htttttttmmmmmmmmmllllllllllllllllllll:")
// 	err2 := th.Execute(os.Stdout, tbl)
// 	if err2 != nil {
// 		fmt.Println("executing template:", err2)
// 		os.Exit(1)
// 	}
// }

// file length type
type flen []int64
func (a flen) Len() int { return len(a) }
func (a flen) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a flen) Less(i, j int) bool { return a[i] < a[j] }

type sortFilesGroupByName FilesGroup
func (a sortFilesGroupByName) Len() int { return len(a) }
func (a sortFilesGroupByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortFilesGroupByName) Less(i, j int) bool { return a[i].FullName < a[j].FullName }

type sortFilesGroupBySiblingsGroup FilesGroup
func (a sortFilesGroupBySiblingsGroup) Len() int { return len(a) }
func (a sortFilesGroupBySiblingsGroup) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortFilesGroupBySiblingsGroup) Less(i, j int) bool { return a[i].Stats.Size() < a[j].Stats.Size() }

type sortGroupByFileSize Groups
func (a sortGroupByFileSize) Len() int { return len(a) }
func (a sortGroupByFileSize) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortGroupByFileSize) Less(i, j int) bool { return a[i][0].Stats.Size() < a[j][0].Stats.Size() }

type sortGroupByName Groups
func (a sortGroupByName) Len() int { return len(a) }
func (a sortGroupByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortGroupByName) Less(i, j int) bool { return a[i][0].FullName < a[j][0].FullName }

type sortGroupBySiblingsCount Groups
func (a sortGroupBySiblingsCount) Len() int { return len(a) }
func (a sortGroupBySiblingsCount) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortGroupBySiblingsCount) Less(i, j int) bool { return a[i][0].SiblingsCount < a[j][0].SiblingsCount }

func (fg *FilesGroup) SortCustom(sortType int) {
	switch sortType {
	case SortBySize:
		// sort.Sort(sort.Reverse(sortGroupByFileSize(*fg)))
	case SortByName:
		sort.Sort(sortFilesGroupByName(*fg))
	case SortBySiblingsCount:
		sort.Sort(sort.Reverse(sortFilesGroupBySiblingsGroup(*fg)))
	}
}
func (g *Groups) SortCustom(sortType int) {
	for _, fg := range *g { fg.SortCustom(sortType) }
	switch sortType {
	case SortBySize:
		sort.Sort(sort.Reverse(sortGroupByFileSize(*g)))
	case SortByName:
		sort.Sort(sortGroupByName(*g))
	case SortBySiblingsCount:
		sort.Sort(sort.Reverse(sortGroupBySiblingsCount(*g)))
	}
}


// // Encode via Gob to file
// func Save(path string, object interface{}) error {
// 	file, err := os.Create(path)
// 	if err == nil {
// 		encoder := gob.NewEncoder(file)
// 		encoder.Encode(object)
// 	}
// 	file.Close()
// 	return err
//  }

// // Decode Gob file
// func Load(path string, object interface{}) error {
// 	file, err := os.Open(path)
// 	if err == nil {
// 		decoder := gob.NewDecoder(file)
// 		err = decoder.Decode(object)
// 	}
// 	file.Close()
// 	return err
// }
