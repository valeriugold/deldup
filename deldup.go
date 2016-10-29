// Delete duplicates from given directory list
package main

import (
	"bufio"
	"crypto/md5"
	// "encoding/gob"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	// "flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var cmdLineDirsIn = flag.String("dirs", ".", "search for duplicates in these dirs")
var cmdLineDirsExclude = flag.String("exclude", "", "omit seraching in these dirs")

type flen []int64
func (a flen) Len() int { return len(a) }
func (a flen) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a flen) Less(i, j int) bool { return a[i] < a[j] }

type filestats struct {
	fullName    string
	stats	    os.FileInfo
	md5sum	    [md5.Size]byte
}

func (st *filestats) String() string {
	return fmt.Sprintf("fn:%s name:%s size:%d modtime:%s",
		st.fullName, st.stats.Name(), st.stats.Size(),
		st.stats.ModTime().Format("2006-01-02 15:04:05"))
}

type cachedMd5Sum struct {
	md5sum	    [md5.Size]byte
	fullName    string
}
var tokens = make(chan struct{}, 20)

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
	
	lenToNames, lenDuplicates := parseDirStructure(&roots, &exclude)
	
	// // just for debug print lenToNames
	// for _, v := range lenToNames {
	// 	fmt.Printf("elements: ")
	// 	for _, x := range v {
	// 		fmt.Printf(" %v, ex: %v", x, x.fullName)
	// 	}
	// 	fmt.Printf("\n")
	// }
	fmt.Println("add md5sums")

	// get md5sum for all potential duplicate files
	fillMd5SumField(&lenToNames, &lenDuplicates, cacheFileName)

	// construct a map of length, that on each value has a map of md5sums,
	// that on each value has a slice of identical files
	len5 := make(map[int64]*map[[md5.Size]byte][]*filestats)
	
	// for length := range lenMulti {
	for _, length := range lenDuplicates {
		if m5 := getMapOfMd5Duplicates(lenToNames[length]); m5 != nil {
			len5[length] = m5
		}
	}
	dupSlice := make(flen, len(len5), len(len5))
	for length := range len5 {
		dupSlice = append(dupSlice, length)
	}
	sort.Sort(sort.Reverse(dupSlice))
	
	fmt.Println("here are duplicates")
	for _, length := range dupSlice {
		if length == 0 {
			continue
		}
		fmt.Printf("length %d:\n", length)
		for k, v := range *(len5[length]) {
			fmt.Printf("    sum:%v\n", k)
			for _, fs := range v {
				fmt.Printf("        %s\n", fs.fullName)
			}
		}
	}

	
	// 	var sf []string
	// 	for _, f := range lenToNames[length] {
	// 		sf = append(sf, f.fullName)
	// 	}
	// 	var d string
	// 	var sb []string
	// 	if len(roots) < 2 {
	// 		l := len(roots[0])
	// 		if ! (roots[0][l-1:] == "/") {
	// 			l++
	// 		}
	// 		d, sb = removeRootDir(l, sf)
	// 	} else {
	// 		d, sb = getCommonDir(sf)
	// 	}
	// 	fmt.Printf("dir: %s len %d : ", d, length)
	// 	for _, f := range sb {
	// 		fmt.Printf("%s, ", f)
	// 	}
	// 	// e := lenToNames[length][0]
	// 	// fmt.Printf("dir: %s len %d : ",
	// 	// 	e.fullName[:len(e.fullName) - len(e.stats.Name())],
	// 	// 	length)
	// 	// for _, f := range lenToNames[length] {
	// 	// 	fmt.Printf(" %s", f.stats.Name())
	// 	// }
	// 	fmt.Printf("\n")
	// }
}

// parse dir structure
func parseDirStructure(roots *([]string), exclude *(map[string]bool)) (map[int64][]*filestats, flen)  {
	reportFiles := make(chan *filestats)
	var n sync.WaitGroup
	for _, dir := range *roots {
		n.Add(1)
		go addDir(dir, &n, reportFiles, exclude)
	}
	go func() {
		n.Wait()
		close(reportFiles)
	}()
	lenToNames := make(map[int64][]*filestats)
	lenDuplicates := make(flen, 256, 256)
	for fs := range reportFiles {
		if fs.stats.IsDir() || fs.stats.Name() == ".DS_Store" {
			continue
		}
		if _, ok := lenToNames[fs.stats.Size()]; ok {
			lenDuplicates = append(lenDuplicates, fs.stats.Size())
		}
		lenToNames[fs.stats.Size()] = append(lenToNames[fs.stats.Size()], fs)
	}
	return lenToNames, lenDuplicates
}

func addDir(dir string, n *sync.WaitGroup, reportFiles chan<- *filestats, exclude *map[string]bool) {
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
		fullName := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			n.Add(1)
			go addDir(fullName, n, reportFiles, exclude)
		}
		reportFiles <- &filestats{fullName, entry, [md5.Size]byte{}}
	}
}

// get md5sum for all potential duplicate files
// modify lenToNames
func fillMd5SumField(lenToNames *(map[int64][]*filestats), lenDuplicates *flen, cacheFileName string) {
	// load known md4sum values from cache
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
	// loop only for lenghts that feet multiple files, possible duplicates
	for _, length := range *lenDuplicates {
		if length == 0 {
			continue
		}
		for _, f := range (*lenToNames)[length] {
			if n, ok := map5[f.fullName]; ok {
				// fmt.Printf("_load md5 form cache %s\n", f.fullName)
				f.md5sum = n
			// } else if !f.stats.IsDir() && f.stats.Name() != ".DS_Store" {
			} else {
				n2.Add(1)
				go func(fs *filestats) {
					defer n2.Done()
					tokens <- struct{}{}
					defer func() { <- tokens }()
					fs.md5sum, _ = getMd5FromFile(fs.fullName)
					chSaveMd5Sums <- cachedMd5Sum{fs.md5sum, fs.fullName}
				}(f)
			}
		}
	}
	n2.Wait()
	close(chSaveMd5Sums)
	
	// // add all known md5sums to map5
	// for _, length := range *lenDuplicates {
	// 	for _, f := range (*lenToNames)[length] {
	// 		map5[f.fullName] = f.md5sum
	// 	}
	// }
	return
}

// return map of [md5sum] -> []*filestats
func getMapOfMd5Duplicates(files []*filestats) (*map[[md5.Size]byte][]*filestats) {
	m5 := make(map[[md5.Size]byte][]*filestats)
	for _, f := range files {
		m5[f.md5sum] = append(m5[f.md5sum], f)
	}
	// remove md5sums and files that don't have duplicates in m5
	for k, v := range m5 {
		if len(v) <= 1 {
			// this file is not duplicate
			delete(m5, k)
		}
	}
	if len(m5) > 0 {
		return &m5
	}
	return nil
}


func removeRootDir(l int, sf []string) (d string, sb []string) {
	for _, s := range sf {
		sb = append(sb, s[l:])
		// fmt.Printf("%s ,", s[l:])
	}
	// fmt.Printf("\n22222 i=%d, j=%d, d=%s, from %v\n", i, j, d, sf)
	return
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

	// fmt.Printf("00000 i=%d, j=%d, d=%s, from %v\n", i, j, d, sf)
	for j = i - 1; j >= 0; j-- {
		if sf[0][j] == '/' {
			break
		}
	}
	j++
	// fmt.Printf("11111 i=%d, j=%d, d=%s, return: ", i, j, d)
	d = sf[0][:j]
	for _, s := range sf {
		sb = append(sb, s[j:])
		// fmt.Printf("%s ,", s[j:])
	}
	// fmt.Printf("\n22222 i=%d, j=%d, d=%s, from %v\n", i, j, d, sf)
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
