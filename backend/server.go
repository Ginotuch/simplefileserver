package backend

import (
	"archive/zip"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	auth "github.com/abbot/go-http-auth"

	"golang.org/x/sys/unix"
)

const walkTemplate = `
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
					<li><a href="{{.Name}}/">{{.Name}}/</a>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<a href="/{{.DownloadPath}}">zip download</a></li>
				{{end}}
			{{end}}
		</ul>
	</body>
</html>`

type Server interface {
	E404(w http.ResponseWriter, req *http.Request)
	Home(w http.ResponseWriter, req *http.Request)
	Walk(w http.ResponseWriter, req *auth.AuthenticatedRequest)
	Favicon(w http.ResponseWriter, req *auth.AuthenticatedRequest)
	Download(w http.ResponseWriter, req *auth.AuthenticatedRequest)
	DownloadFile(w http.ResponseWriter, req *auth.AuthenticatedRequest, absPath string)
	DownloadFolder(w http.ResponseWriter, absPath string)
}

type ServerStruct struct {
	logFile      *os.File
	logLevel     int
	rootDir      string
	walkTemplate *template.Template
}

func (s *ServerStruct) E404(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "404\n")
}

func (s *ServerStruct) Home(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		s.E404(w, req)
		return
	}
	homeHTML := `
<html>
   <head>
      <style>body{width:9px;height:9px;position:absolute;top:0;bottom:0;left:0;right:0;margin:auto;}</style>
      <title>hey</title>
   </head>
   <body><a href="/walk/">walk</a></body>
</html>
`
	fmt.Fprint(w, homeHTML)
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

func (s *ServerStruct) Favicon(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
	w.Header().Set("Content-Type", "image/x-icon")
	file, err := os.Open("favicon.ico")
	if err != nil {
		s.logger(LogWarning, req, "Favicon")
		return
	}

	var ftime time.Time
	fileStat, err := os.Stat("favicon.ico")

	if err != nil {
		ftime = time.Time{}
	} else {
		ftime = fileStat.ModTime() // doesn't seem to actually set file dates
	}

	r := http.Request{
		Method:           req.Method,
		URL:              req.URL,
		Proto:            req.Proto,
		ProtoMajor:       req.ProtoMajor,
		ProtoMinor:       req.ProtoMinor,
		Header:           req.Header,
		Body:             req.Body,
		GetBody:          req.GetBody,
		ContentLength:    req.ContentLength,
		TransferEncoding: req.TransferEncoding,
		Close:            req.Close,
		Host:             req.Host,
		Form:             req.Form,
		PostForm:         req.PostForm,
		MultipartForm:    req.MultipartForm,
		Trailer:          req.Trailer,
		RemoteAddr:       req.RemoteAddr,
		RequestURI:       req.RequestURI,
		TLS:              req.TLS,
		Cancel:           req.Cancel,
		Response:         req.Response,
	}

	http.ServeContent(w, &r, "favicon.ico", ftime, file)
}

func (s *ServerStruct) Walk(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
	s.logger(LogInfo, req, "Walk")
	requestedFolder := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	absPath := path.Join(s.rootDir, requestedFolder)

	fileInfo, err := os.Stat(absPath)
	if unix.Access(absPath, unix.R_OK) != nil || err != nil || !fileInfo.IsDir() {
		s.logger(LogWarning, req, "PathAccessDenied")
		w.WriteHeader(http.StatusNotFound)
		_, err = fmt.Fprintf(w, "Either the requested directory doesn't exist or access was denied")
		if err != nil {
			s.logger(LogError, req, "UnableToWriteResponse")
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

	files, err := ioutil.ReadDir(absPath) // todo: race condition with above unix.Access check (folder permission removal)
	for _, f := range files {
		data.Entries = append(data.Entries, entry{
			Name:         f.Name(),
			File:         !f.IsDir(),
			DownloadPath: path.Join("download", requestedFolder, f.Name()),
		})
	}

	err = s.walkTemplate.Execute(w, data)
	if err != nil {
		s.logger(LogError, req, "Walk-UnableToWriteResponse")
	}
}

func (s *ServerStruct) Download(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
	s.logger(LogInfo, req, "Download")
	requestedThing := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	absPath := path.Join(s.rootDir, requestedThing)

	fileInfo, err := os.Stat(absPath)
	if unix.Access(absPath, unix.R_OK) != nil || err != nil {
		s.logger(LogWarning, req, "PathAccessDenied")
		w.WriteHeader(http.StatusNotFound)
		_, err = fmt.Fprintf(w, "Either the requested item doesn't exist or access was denied")
		if err != nil {
			s.logger(LogError, req, "Download-UnableToWriteResponse")
		}
		return
	}

	if fileInfo.IsDir() {
		s.DownloadFolder(w, absPath)
	} else {
		s.DownloadFile(w, req, absPath)
	}
}

func (s *ServerStruct) DownloadFolder(w http.ResponseWriter, absPath string) {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", path.Base(absPath)))
	zipWriter := zip.NewWriter(w)

	walkErr := filepath.Walk(absPath, func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		zipPath := path.Join(strings.Split(filePath, "/")[len(strings.Split(absPath, "/")):]...)
		fmt.Printf("[%s][Zipping] folder: \"%s\" file: \"%s\"\n", time.Now().Format("2006-01-02 15:04:05"), absPath, zipPath)
		fileWriter, err := zipWriter.CreateHeader(&zip.FileHeader{Name: zipPath, Method: zip.Store})
		if err != nil {
			log.Println("couldn't make file in zip")
			log.Println(err)
		}
		fileReader, err := os.Open(filePath)
		_, err = io.Copy(fileWriter, fileReader)
		if err != nil {
			log.Println("(most likely download stopped)")
			log.Println(err)
		}
		return nil
	})
	if walkErr != nil {
		log.Println("walk error")
		log.Println(walkErr)
	}
	err := zipWriter.Close()
	if err != nil {
		log.Println("failed to close zip file")
		log.Println(err)
	}
}

func (s *ServerStruct) DownloadFile(w http.ResponseWriter, req *auth.AuthenticatedRequest, absPath string) {
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
		ftime = fileStat.ModTime() // doesn't seem to actually set file dates
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", path.Base(req.URL.Path)))
	r := http.Request{
		Method:           req.Method,
		URL:              req.URL,
		Proto:            req.Proto,
		ProtoMajor:       req.ProtoMajor,
		ProtoMinor:       req.ProtoMinor,
		Header:           req.Header,
		Body:             req.Body,
		GetBody:          req.GetBody,
		ContentLength:    req.ContentLength,
		TransferEncoding: req.TransferEncoding,
		Close:            req.Close,
		Host:             req.Host,
		Form:             req.Form,
		PostForm:         req.PostForm,
		MultipartForm:    req.MultipartForm,
		Trailer:          req.Trailer,
		RemoteAddr:       req.RemoteAddr,
		RequestURI:       req.RequestURI,
		TLS:              req.TLS,
		Cancel:           req.Cancel,
		Response:         req.Response,
	}
	http.ServeContent(w, &r, path.Base(req.URL.Path), ftime, file)
	_ = file.Close()
}

const (
	LogDebug   = 0
	LogInfo    = 1
	LogWarning = 2
	LogError   = 3
)

var LogNames = [4]string{"DEBUG", "INFO", "WARNING", "ERROR"}

func (s *ServerStruct) logger(level int, req *auth.AuthenticatedRequest, caller string) {
	if level >= s.logLevel {
		logString := fmt.Sprintf("[%s][%s][%s][%s] user:\"%s\" URL:\"%s\"\n", time.Now().Format("2006-01-02 15:04:05"), LogNames[level], caller, req.RemoteAddr, req.Username, req.URL.Path)
		fmt.Print(logString)
		_, err := s.logFile.WriteString(logString)
		if err != nil {
			log.Fatal("Unable to write to log file")
		}
		err = s.logFile.Sync()
		if err != nil {
			log.Fatal("Unable to write to log file")
		}
	}
}

func NewServer(rootDir string, logLevel int) Server {
	t, err := template.New("walkHTML").Parse(walkTemplate)
	if err != nil {
		log.Fatal("Failed to parse template")
	}
	logFile, err := os.OpenFile("logs.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		log.Fatal(err)
	}
	_, err = logFile.WriteString(fmt.Sprintf("=====New Server created on %s=====\n", time.Now().Format("2006-01-02 15:04:05")))
	if err != nil {
		log.Fatal("Unable to write to log file")
	}
	newServer := &ServerStruct{logFile: logFile, logLevel: logLevel, rootDir: rootDir, walkTemplate: t}

	return newServer
}
