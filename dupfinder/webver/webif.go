// present a web view of deldup
package main

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/valeriugold/deldup/dupfinder"
	"github.com/valeriugold/deldup/dupfinder/webver/vviews"
)

var cacheFileName string
var baseDir string = ``
var startDir string = ``

func handlerExitNow(w http.ResponseWriter, r *http.Request) {
	os.Exit(0)
}

func handlerPrintPostData(w http.ResponseWriter, r *http.Request) {
	fmt.Print("handlerPrintPostData !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n")
	r.ParseForm()
	for k, v := range r.PostForm {
		fmt.Printf("k = %s, v=%q\n", k, v)
		for _, x := range v {
			fmt.Printf("    v= %s\n", x)
		}
	}
}
func handlerDeletePostFiles(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("handlerDeletePostFiles handler:\n")
	r.ParseForm()

	if v, ok := r.PostForm["deleteTheseFiles[]"]; ok {
		var buffer bytes.Buffer
		for _, f := range v {
			s := fmt.Sprintf("%s\n", f)
			os.Remove(f)
			buffer.WriteString(s)
		}
		fmt.Printf("POST files are:\n%s", buffer.String())
	}
	handlerFindDuplicates(w, r)
}

func handlerFindDuplicates(w http.ResponseWriter, r *http.Request) {
	var sq struct {
		dirNames     []string
		excludeNames map[string]bool
	}
	r.ParseForm()
	sq.excludeNames = make(map[string]bool)
	if v, ok := r.PostForm["dir"]; ok {
		for _, d := range v {
			sq.dirNames = append(sq.dirNames, d)
		}
	}
	if v, ok := r.PostForm["exclude"]; ok {
		for _, d := range v {
			sq.excludeNames[d] = true
		}
	}
	fmt.Printf("handlerFindDuplicates: dirs:%v\nexclude:%v\n", sq.dirNames, sq.excludeNames)
	dups := dupfinder.GetDups(&sq.dirNames, &sq.excludeNames, cacheFileName)

	// dups.SortCustom(dupfinder.SortBySize)
	// dups.SortCustom(dupfinder.SortByName)
	dups.SortCustom(dupfinder.SortBySiblingsCount)
	// printDuplicates(&dups)
	getHTMLTableFromGroups(&dups, w)
}

// display plain form for dir and exclude - not used now...
func handlerGetDirsPlainForm(w http.ResponseWriter, r *http.Request) {
	const templateGetDirs = `<h1>Look For Duplicates</h1>
<form action="/dups" method="POST">
Where to look for dirs?<br>
<div><input type="text" name="dir" size="60"></div>
What dirs to exclude?<br>
<div><input type="text" name="exclude" size="60"></div>
<div><input type="submit" value="GoFind"></div>
</form>`
	fmt.Fprintf(w, "%s\n", templateGetDirs)
}

// display the page containing the dir browser; it will be refreshed by Ajax calling handlerDirBrowserAjax
func handlerDirBrowserPage(w http.ResponseWriter, r *http.Request) {
	// rootDir := "/" // linux and OSX
	// if len(startDir) > 0 {
	// 	rootDir = startDir
	// } else if runtime.GOOS == "windows" {
	// 	rootDir = `c:/`
	// }
	// switch runtime.GOOS {
	// case "windows":
	// case "linux":
	// case "darwin":
	// default:
	// }

	// 	htmlDirBrowser := `<!DOCTYPE html>
	// <html lang="en">
	//     <head><title>Choose dir</title>
	//         <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
	// ` + cssRules + `
	//         <script src="https://code.jquery.com/jquery-1.9.1.min.js"></script>
	//         <script>
	// 	var dirsSearch = new Object();
	// 	var dirsExclude = new Object();
	// 	var currentDir = "";
	// 	function startDelDup() {
	// 	    var form = $('<form></form>');

	// 	    form.attr("method", "post");
	// 	    form.attr("action", "/dups");	// dups	// printPost

	//    	    for(var key in dirsSearch) {
	// 		    var field = $('<input></input>');
	// 		    field.attr("type", "hidden");
	// 		    field.attr("name", "dir");
	// 		    field.attr("value", key);
	// 		    form.append(field);
	// 	    }
	//    	    for(var key in dirsExclude) {
	// 		    var field = $('<input></input>');
	// 		    field.attr("type", "hidden");
	// 		    field.attr("name", "exclude");
	// 		    field.attr("value", key);
	// 		    form.append(field);
	// 	    }

	// 	    // The form needs to be a part of the document in
	// 	    // order for us to be able to submit it.
	// 	    $(document.body).append(form);
	// 	    form.submit();
	// 	}
	// 	function addSearchDir() {
	// 	    dirsSearch[currentDir] = 1;
	// 	    displaySearchDir();
	// 	}
	// 	function addExcludeDir() {
	// 	    dirsExclude[currentDir] = 1;
	// 	    displayExcludeDir();
	// 	}
	// 	function removeSearchDir(d) {
	// 	    dirsSearch[d] = 0;
	// 	    displaySearchDir();
	// 	}
	// 	function removeExcludeDir(d) {
	// 	    dirsExclude[d] = 0;
	// 	    displayExcludeDir();
	// 	}
	// 	function displaySearchDir() {
	// 	    var s = "";
	// 	    for(var key in dirsSearch) {
	// 		// alert("key = " + key + " value = " + dirsSearch[key])
	// 		if (dirsSearch[key] == 1) {
	// 		    s += "<li class='liDir' onclick='removeSearchDir(\"" + key + "\")'>" + key + "</li>";
	// 		}
	// 	    }
	// 	    // alert("s=" + s);
	// 	    document.getElementById("liSelectedSearchDir").innerHTML = s;
	// 	}
	// 	function displayExcludeDir() {
	// 	    var s = "";
	// 	    for(key in dirsExclude) {
	// 		if (dirsExclude[key] == 1) {
	// 		    s += "<li class='liDir' onclick='removeExcludeDir(\"" + key + "\")'>" + key + "</li>";
	// 		}
	// 	    }
	// 	    document.getElementById("liSelectedExcludeDir").innerHTML = s;
	// 	}
	// 	$(document).ready(function(){
	// 	    // alert("ready");
	// 	    $.post("/searchDir", { dir: "` + rootDir + `" },
	// 		    function(data, status){
	// 			document.getElementById("searchDir").innerHTML = data;
	// 			// alert("Data: " + data + "\nStatus: " + status);
	// 		});
	// 	});
	// 	function loadDir(dirName) {
	// 	    // alert("send req for " + dirName)
	// 	    $.post("/searchDir", { dir: dirName },
	// 		    function(data, status){
	// 			currentDir = dirName
	// 			document.getElementById("searchDir").innerHTML = data;
	// 			// alert("Data: " + data + "\nStatus: " + status);
	// 		});
	// 	}
	// 	</script>
	//     </head>
	//     <body>
	//     <button class="buttonAddDirs" onclick=addSearchDir()>Add to search dirs</button>
	//     <button class="buttonAddDirs" onclick=addExcludeDir()>Add to exclude dirs</button>
	//     <button class="buttonStart" onclick=startDelDup()>Start looking for duplicates</button>
	//     <button class="buttonExit" onclick=exitProgram()>Exit</button>
	//     <h3>Search dirs:</h3>
	//     <div id="liSelectedSearchDir">
	//     </div>
	//     <h3>Exclude dirs:</h3>
	//     <div id="liSelectedExcludeDir">
	//     </div>
	//     <h1>Choose dir<br />----------------------------</h1>
	//     <div id="searchDir">
	//     </div>

	//     </body>
	// </html>
	// `
	// w.Write([]byte(htmlDirBrowser))
	vviews.ChooseDirs(w)
}

func handlerDirBrowserAjax(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("already in handlerDirBrowserAjax\n")
	url := html.UnescapeString(r.FormValue("dir"))
	urlPath := path.Join(baseDir, url)

	fmt.Printf("current path is %s, base=%s, url=%s\n", urlPath, baseDir, url)
	if fi, err := os.Stat(urlPath); err != nil {
		reportError(w, err)
		return
	} else {
		if !fi.Mode().IsDir() {
			reportError(w, errors.New("not a dir"))
			return
		}
		// apparently Stat traverses symlinks, so they appear as a directory
		// else if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		// 	fmt.Printf("new dir is a symlink !!!! %s \n", urlPath)
		// 	linkTarget, err := os.Readlink(urlPath)
		// 	if err != nil {
		// 		reportError(w, err)
		// 		return
		// 	}
		// 	fi, err := os.Stat(linkTarget)
		// 	if err != nil {
		// 		reportError(w, err)
		// 		return
		// 	}
		// 	if !fi.IsDir() {
		// 		reportError(w, errors.New("symlink does not point to a dir"))
		// 		return
		// 	}
		// 	urlPath = linkTarget
		// 	baseDir = ""
		// 	url = linkTarget
		// 	fmt.Printf("set basedir to null\n")
		// }
	}

	// type DirDesc struct {
	// 	ParentDir  string
	// 	CurrentDir string
	// 	Dirs       []string
	// 	Files      []string
	// }
	// var d DirDesc
	var d struct {
		ParentDir  string
		CurrentDir string
		Dirs       []string
		Files      []string
	}

	d.CurrentDir = url
	d.ParentDir = filepath.Dir(d.CurrentDir)
	fmt.Printf("parent dir = %s, current=%s\n", d.ParentDir, d.CurrentDir)

	// Have enough info to figure out what to send back
	files, err := ioutil.ReadDir(urlPath)
	if err != nil {
		reportError(w, err)
		return
	}

	for _, element := range files {
		if element.IsDir() {
			d.Dirs = append(d.Dirs, element.Name())
		} else if element.Mode()&os.ModeSymlink == os.ModeSymlink {
			fmt.Printf("check symlink = %s, from urlPath = %s\n", element.Name(), urlPath)
			linkTarget, err := os.Readlink(path.Join(urlPath, element.Name()))
			if err != nil {
				reportError(w, err)
				return
			}
			fi, err := os.Stat(linkTarget)
			if err == nil {
				if fi.IsDir() {
					d.Dirs = append(d.Dirs, element.Name())
				} else {
					d.Files = append(d.Files, element.Name())
				}
			}
		} else {
			d.Files = append(d.Files, element.Name())
		}
	}
	fmt.Print("here comes dddddd \n")
	fmt.Print(d)
	fmt.Printf("curreent=%s, urlPath=%s\n", d.CurrentDir, urlPath)
	fmt.Printf("len=%d, dirs=%q\n", len(d.Dirs), d.Dirs)
	// funcMap := template.FuncMap{
	// 	"escapeBackSlash": func(s string) string {
	// 		r, _ := regexp.Compile(`\\+`)
	// 		return r.ReplaceAllLiteralString(s, `\`)
	// 	},
	// }
	// 	const htmlDirBrowser = `
	//         {{ $d := .CurrentDir }}
	//         <h2>current dir: {{html $d}}</h2>
	//         <ul>
	//         	<li class="liDir" onclick="loadDir('{{ escapeBackSlash .ParentDir | html }}')">[d] ..</li>
	//         	{{range .Dirs}}
	//         	    <li class="liDir" onclick='loadDir("{{if ne $d "/"}}{{ escapeBackSlash $d | html }}{{end}}/{{html .}}")'>[d] {{.}}</li>
	//         	{{end}}
	//         	{{range .Files}}
	//         	    <li>[ ]{{html .}}</li>
	//         	{{end}}
	//         </ul>
	// `

	vviews.AjaxDirBrowser(w, d)
	// th := template.Must(template.New("dupfiles2").Funcs(funcMap).Parse(htmlDirBrowser))
	// //th := template.Must(template.New("dupfiles2").Parse(htmlDirBrowser))
	// err = th.Execute(w, d)
	// if err != nil {
	// 	fmt.Fprintln(w, "executing template:", err)
	// 	os.Exit(1)
	// }
}

func main() {
	dir, ef := filepath.Abs(filepath.Dir(os.Args[0]))
	if ef != nil {
		fmt.Print("the current dir could not be found, use /tmp/")
		cacheFileName = "/tmp/dupfinder.cache"
	} else {
		cacheFileName = path.Join(dir, "dupfinder.cache")
	}
	if len(os.Args) > 1 {
		startDir = os.Args[1]
	}
	vviews.Init()
	http.Handle("/bootstrap/css/", http.StripPrefix("/bootstrap/css/", http.FileServer(http.Dir("bootstrap-3.3.7-dist/css"))))
	http.Handle("/bootstrap/js/", http.StripPrefix("/bootstrap/js/", http.FileServer(http.Dir("bootstrap-3.3.7-dist/js"))))
	// http.Handle("/bootstrap/css/", http.StripPrefix("/bootstrap/css/", http.FileServer(http.Dir("/Users/valeriug/dev/go/src/github.com/valeriugold/deldup/dupfinder/webver/bootstrap-3.3.7-dist")))
	// http.Handle("/bcss/", http.FileServer(http.Dir("/Users/valeriug/dev/go/src/github.com/valeriugold/deldup/dupfinder/webver/bootstrap-3.3.7-dist/css")))
	// http.Handle("/x", http.FileServer(http.Dir("bootstrap-3.3.7-dist/css")))
	// http.Handle("/", http.FileServer(http.Dir("bootstrap-3.3.7-dist/css")))
	http.Handle("/y/", http.StripPrefix("/y/", http.FileServer(http.Dir("bootstrap-3.3.7-dist/css"))))
	// http.HandleFunc("/y", http.FileServer(http.Dir("bootstrap-3.3.7-dist/css")))
	// http.HandleFunc("/", http.FileServer(http.Dir("bootstrap-3.3.7-dist/css")))
	http.HandleFunc("/deldup", handlerGetDirsPlainForm)
	http.HandleFunc("/dups", handlerFindDuplicates)
	http.HandleFunc("/delfiles", handlerDeletePostFiles)
	http.HandleFunc("/dirs", handlerDirBrowserPage)
	http.HandleFunc("/searchDir", handlerDirBrowserAjax)
	http.HandleFunc("/printPost", handlerPrintPostData)
	http.HandleFunc("/exitNow", handlerExitNow)

	go startBrowserPage(8080)
	err := http.ListenAndServe(":8080", nil)
	log.Fatal(err)
}

func getHTMLTableFromGroups(g *dupfinder.Groups, w http.ResponseWriter) {
	// funcMap := template.FuncMap{
	// 	"length": func(x dupfinder.FilesGroup) int { return len(x) },
	// 	"dir":    func(x dupfinder.FilesGroup) string { return filepath.Dir(x[0].FullName) },
	// 	"size":   func(x dupfinder.FilesGroup) int64 { return x[0].Stats.Size() },
	// }
	// 	const cht = `
	// <!DOCTYPE html>
	// <html>
	// <head>
	// ` + cssRules + `
	// <style>
	// table {
	//     border-collapse: collapse;
	//     width: 100%;
	// }

	// th, td {
	//     text-align: left;
	//     padding: 8px;
	// }

	// tr:nth-child(even){background-color: #f2f2f2}

	// th {
	//     background-color: #4CAF50;
	//     color: white;
	// }

	// tr:hover {
	//     background-color: #ffff99;
	// }

	// td:hover {
	//     background-color: #33FFFC;
	// }
	// </style>
	// <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.1.1/jquery.min.js"></script>
	// <script>
	// var mapFiles = new Map()
	// function toggleDelete(el, fileName) {
	//     if (mapFiles.has(fileName)) {
	// 	el.style.color = 'black';
	// 	mapFiles.delete(fileName);
	//     } else {
	// 	el.style.color = 'red';
	// 	mapFiles.set(fileName, 1);
	//     }
	// }
	// function apply() {
	//     var files = []
	//     for (var key of mapFiles.keys()) {
	// 	console.log(key);
	// 	files.push(key)
	//     }
	//     // window.alert(files)
	//     $.post('/delfiles', {'deleteTheseFiles': files});
	//     location.reload();
	// }
	// function cancel() {
	//     mapFiles.clear();
	//     location.reload();
	// }
	// </script>
	// </head>
	// <body>

	// <h2>Duplicate files</h2>

	// <table border=3>
	// <tr><th>size</th><th colspan="2">identical files</th></tr>
	// {{range .}}
	// <tr><td>{{size .}}</td>{{range .}}<td onclick="toggleDelete(this, '{{.FullName}}')">{{.FullName}} </td>{{end}}</tr>
	// {{end}}</table>

	// <button class="buttondel" onclick="apply()">Delete Files</button>
	// <button class="buttoncancel" onclick="cancel()">Cancel</button>
	// <button class="buttonExit" onclick=exitProgram()>Exit</button>
	// </body>
	// </html>
	// `
	vviews.Duplicates(w, g)
	// th := template.Must(template.New("dupfiles2").Funcs(funcMap).Parse(cht))
	// // err2 := th.Execute(os.Stdout, g)
	// err2 := th.Execute(w, g)
	// if err2 != nil {
	// 	fmt.Fprintln(w, "executing template:", err2)
	// 	os.Exit(1)
	// }
}

func printDuplicates(dups *dupfinder.Groups) {
	fmt.Println("here are duplicates")
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

func reportError(w http.ResponseWriter, err error) {
	fmt.Fprintf(w, "Error during operation: %s", err)
}

func startBrowserPage(port int) {
	time.Sleep(2 * time.Second)
	var err error
	s := "http://localhost:" + strconv.Itoa(port) + "/dirs"
	fmt.Printf("start browser here: %s\n", s)
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", s).Start()
	case "windows":
		err = exec.Command("explorer", s).Start()
	case "darwin":
		err = exec.Command("open", s).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Printf("%q", err)
	}
}
