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
	"time"

	"golang.org/x/sys/unix"
)

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
					<li><a href="/{{.DownloadPath}}">{{.Name}}</a></li>
				{{else}}
					<li><a href="{{.Name}}/">{{.Name}}/</a></li>
				{{end}}
			{{end}}
		</ul>
	</body>
</html>`

type Server interface {
	Hello(w http.ResponseWriter, req *http.Request)
	Headers(w http.ResponseWriter, req *http.Request)
	E404(w http.ResponseWriter, req *http.Request)
	Home(w http.ResponseWriter, req *http.Request)
	Walk(w http.ResponseWriter, req *http.Request)
	Download(w http.ResponseWriter, req *http.Request)
}

type ServerStruct struct {
	rootDir      string
	walkTemplate *template.Template
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
	Name         string
	File         bool
	DownloadPath string
}

type walkData struct {
	Path    string
	Entries []entry
}

func (s *ServerStruct) Walk(w http.ResponseWriter, req *http.Request) {
	requestedFolder := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	absPath := path.Join(s.rootDir, requestedFolder)

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
		data.Entries = append(data.Entries, entry{Name: f.Name(), File: !f.IsDir(), DownloadPath: path.Join("download", requestedFolder, f.Name())})
	}

	err = s.walkTemplate.Execute(w, data)
	if err != nil {
		log.Fatal("Unable to write response")
	}
}

func (s *ServerStruct) Download(w http.ResponseWriter, req *http.Request) {
	requestedThing := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	absPath := path.Join(s.rootDir, requestedThing)

	fileInfo, err := os.Stat(absPath)
	if unix.Access(absPath, unix.R_OK) != nil || err != nil || fileInfo.IsDir() {
		w.WriteHeader(http.StatusNotFound)
		_, err = fmt.Fprintf(w, "Either the requested directory doesn't exist or access was denied")
		if err != nil {
			log.Fatal("Unable to write response")
		}
		return
	}

	file, err := os.Open(absPath)
	if err != nil {
		_, err = fmt.Fprintf(w, "Unable to get file")
		if err != nil {
			log.Fatal("Unable to write response")
		}
		return
	}

	var ftime time.Time
	fileStat, err := os.Stat(absPath)

	if err != nil {
		ftime = time.Time{}
	} else {
		ftime = fileStat.ModTime()
	}

	w.Header().Set("Content-Disposition:", fmt.Sprintf("attachment; filename=\"%s\"", path.Base(req.URL.Path)))

	http.ServeContent(w, req, path.Base(req.URL.Path), ftime, file)
	_ = file.Close()
}

func NewServer(rootDir string) Server {
	t, err := template.New("walkHTML").Parse(tpl)
	if err != nil {
		log.Fatal("Failed to parse template")
	}
	newServer := &ServerStruct{rootDir: rootDir, walkTemplate: t}

	return newServer
}
