package vviews

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"regexp"

	"github.com/valeriugold/deldup/dupfinder"
)

// load templates
// define generic json printing functions
// define how functions interract

// global var holding all templates
// var t *template.Template
var views = make(map[string]*View)
var nameOfBaseTmpl = "base"

// var dir = "/Users/valeriug/dev/go/src/github.com/valeriugold/vket/vviews/vtemplates"
var dir = "/Users/valeriug/dev/go/src/github.com/valeriugold/deldup/dupfinder/webver/vviews/vtemplates"

var funcMap = template.FuncMap{
	"length": func(x dupfinder.FilesGroup) int { return len(x) },
	"dir":    func(x dupfinder.FilesGroup) string { return filepath.Dir(x[0].FullName) },
	"size":   func(x dupfinder.FilesGroup) int64 { return x[0].Stats.Size() },
	"escapeBackSlash": func(s string) string {
		r, _ := regexp.Compile(`\\+`)
		return r.ReplaceAllLiteralString(s, `\`)
	},
}

// Init should be called automatically when this package is used
func Init() {
	names := []string{"choosedirs", "duplicates"}
	for _, n := range names {
		views[n] = CreateView(n, nameOfBaseTmpl, n)
	}
	// ajax templates do not start from base, they are alone!
	ajaxNames := []string{"ajaxdirbrowser"}
	for _, n := range ajaxNames {
		views[n] = CreateView(n, n, []string{}...)
	}
}

func GetJSONRepresentation(args ...interface{}) string {
	// b, err := json.Marshal(args)
	b, err := json.MarshalIndent(args, "", "    ")
	if err != nil {
		fmt.Println(err, " args=", args)
		return "error marshaling args"
	}
	return string(b)
}

type View struct {
	name     string
	base     string
	files    []string
	template *template.Template
}

func CreateView(name string, baseName string, files ...string) *View {
	fls := []string{baseName + ".tmpl"}
	for _, f := range files {
		fls = append(fls, f+".tmpl")
	}
	v := &View{name: name, base: baseName, files: fls}
	v.Init()
	return v
}
func UseTemplate(w http.ResponseWriter, name string, data interface{}) {
	if v, ok := views[name]; ok {
		v.Render(w, data)
		return
	}
	// http.Error(w, "Did not find template name for data: %v", data)
	fmt.Printf("Did not find template name !%s! for data: %v\n", name, data)
}
func (v *View) Init() {
	paths := make([]string, 0, len(v.files))
	for _, f := range v.files {
		fmt.Printf("d=%s, f=%v\n", dir, f)
		paths = append(paths, filepath.Join(dir, f))
	}
	fmt.Println("l=", len(paths), " paths = ", paths)
	fmt.Printf("0=%s!\n", paths[0])
	v.template = template.Must(template.New("base").Funcs(funcMap).ParseFiles(paths...))
}
func (v *View) Render(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := v.template.ExecuteTemplate(w, v.base, data)
	if err != nil {
		fmt.Printf("err executing template %s: %v", v.name, err)
		return
	}
}

// type VView struct {
// 	name string
// }
//
// func GetVView() VView {
// 	return VView{}
// }

func ChooseDirs(w http.ResponseWriter) {
	UseTemplate(w, "choosedirs", nil)
}

func AjaxDirBrowser(w http.ResponseWriter, data interface{}) {
	UseTemplate(w, "ajaxdirbrowser", data)
}

func Duplicates(w http.ResponseWriter, data interface{}) {
	UseTemplate(w, "duplicates", data)
}

func Error(w http.ResponseWriter, fields ...string) {
	UseTemplate(w, "error", fields)
}

func Register(w http.ResponseWriter, fields ...string) {
	UseTemplate(w, "register", fields)
}
