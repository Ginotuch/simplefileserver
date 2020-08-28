package backend

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"golang.org/x/sys/unix"
)

type Server interface {
	Hello(w http.ResponseWriter, req *http.Request)
	Headers(w http.ResponseWriter, req *http.Request)
	E404(w http.ResponseWriter, req *http.Request)
	Home(w http.ResponseWriter, req *http.Request)
	Walk(w http.ResponseWriter, req *http.Request)
}

type ServerStruct struct {
	RootDir string
}

func (s *ServerStruct) Hello(w http.ResponseWriter, req *http.Request) {

	fmt.Fprintf(w, "Hello\n")
}

func (s *ServerStruct) Headers(w http.ResponseWriter, req *http.Request) {

	for name, headers := range req.Header {
		for _, h := range headers {
			fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}

func (s *ServerStruct) E404(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "my404\n")
}

func (s *ServerStruct) Home(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		s.E404(w, req)
		return
	}
	fmt.Fprintf(w, "Home\n")
}

type entry struct {
	Name string
	File bool
}

type walkData struct {
	Path    string
	Entries []entry
}

func (s *ServerStruct) Walk(w http.ResponseWriter, req *http.Request) {
	const tpl = `
<!DOCTYPE html>
<html>
	<head>
		<title>{{.Path}}</title>
	</head>
	<body>
		<h1>Listing for dir: {{.Path}}</h1>
		<ul>
			<li><a href = "../">../</a></li>
			{{range .Entries}}
				{{if .File}}
					<li>{{.Name}}</li>
				{{else}}
					<li><a href="{{.Name}}/">{{.Name}}/</a></li>
				{{end}}
			{{end}}
		</ul>
	</body>
</html>`
	t, err := template.New("walkHTML").Parse(tpl)
	if err != nil {
		log.Fatal("Failed to parse template")
	}

	requestedFolder := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	absPath := path.Join(s.RootDir, requestedFolder)

	fileInfo, err := os.Stat(absPath)
	if unix.Access(absPath, unix.R_OK) != nil || err != nil || !fileInfo.IsDir() {
		w.WriteHeader(http.StatusNotFound)
		_, err = fmt.Fprintf(w, "Either the requested directory doesn't exist or access was denied")
		if err != nil {
			log.Fatal("Unable to write response")
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	requestedFolder += "/"
	absPath += "/"

	data := walkData{
		Path:    requestedFolder,
		Entries: []entry{},
	}

	files, err := ioutil.ReadDir(absPath)
	for _, f := range files {
		data.Entries = append(data.Entries, entry{Name: f.Name(), File: !f.IsDir()})
	}

	err = t.Execute(w, data)
	if err != nil {
		log.Fatal("Unable to write response")
	}
}
