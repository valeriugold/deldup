// present a web view of deldup
package main

import (
	"fmt"
	//"html/template"
	//"io/ioutil"
	"net/http"
	// "regexp"
)

// type Page struct {
// 	Title string
// 	Body  []byte
// }

// func (p *Page) save() error {
// 	filename := p.Title + ".txt"
// 	return ioutil.WriteFile(filename, p.Body, 0600)
// }

// func loadPage(title string) (*Page, error) {
// 	filename := title + ".txt"
// 	body, err := ioutil.ReadFile(filename)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &Page{Title: title, Body: body}, nil
// }

// func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
// 	p, err := loadPage(title)
// 	if err != nil {
// 		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
// 		return
// 	}
// 	renderTemplate(w, "view", p)
// }

// func editHandler(w http.ResponseWriter, r *http.Request, title string) {
// 	p, err := loadPage(title)
// 	if err != nil {
// 		p = &Page{Title: title}
// 	}
// 	renderTemplate(w, "edit", p)
// }

// func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
// 	body := r.FormValue("body")
// 	p := &Page{Title: title, Body: []byte(body)}
// 	err := p.save()
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	http.Redirect(w, r, "/view/"+title, http.StatusFound)
// }

// var templates = template.Must(template.ParseFiles("edit.html", "view.html"))

// func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
// 	err := templates.ExecuteTemplate(w, tmpl+".html", p)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 	}
// }

// var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
// func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		m := validPath.FindStringSubmatch(r.URL.Path)
// 		if m == nil {
// 			http.NotFound(w, r)
// 			return
// 		}
// 		fn(w, r, m[2])
// 	}
// }


const templateGetDirs = `<h1>Look For Duplicates</h1>

<form action="/dups" method="POST">
Where to look for dirs?<br>
<div><input type="text" name="dir" size="60"></div>
What dirs to exclude?<br>
<div><input type="text" name="exclude" size="60"></div>
<div><input type="submit" value="GoFind"></div>
</form>`

type searchQuery struct {
	dirNames	[]string
	excludeNames	[]string
}

func findDupsHandler(w http.ResponseWriter, r *http.Request) {
	sq := searchQuery{}
	sq.dirNames = append(sq.dirNames, r.FormValue("dir"))
	sq.excludeNames = append(sq.excludeNames, r.FormValue("exclude"))
	fmt.Fprintf(w, "dir=%s, exclu=%s\n", sq.dirNames[0], sq.excludeNames[0])
}

func getDirsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s\n", templateGetDirs)
}

func main() {
	// http.HandleFunc("/view/", makeHandler(viewHandler))
	// http.HandleFunc("/edit/", makeHandler(editHandler))
	// http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/deldup", getDirsHandler)
	http.HandleFunc("/dups", findDupsHandler)

	http.ListenAndServe(":8080", nil)
}


