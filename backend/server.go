package backend

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	auth "github.com/abbot/go-http-auth"

	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

const walkTemplate = `
<!DOCTYPE html>
<html>
	<head>
      <link id="favicon" rel="shortcut icon" type="image/png" href="data:image/png;base64,AAABAAEAEBAQAAEABAAoAQAAFgAAACgAAAAQAAAAIAAAAAEABAAAAAAAgAAAAAAAAAAAAAAAEAAAAAAAAACPj48Ax8fHAOPj4wA7OzsAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAhERERERERACEREREREREAIRMRETMzEQAhExERERExACETERERETEAIRMRERERMQAhEzMREzMRACETERExEREAIRMRETEREQAhExERMRERACETMzMTMzEAIREREREREQAhERERERERACEREREREREAIiIiIiIiIiAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA">
      <title>{{.Path}}</title>
	</head>
	<body>
		<h1>Listing for dir: {{.Path}}</h1>
		<ul>
			<li><a href = "../">../</a></li>
			{{range .Entries}}
				<li>
				{{if .File}}
					<a href="/{{.DownloadPath}}">{{.Name}}</a>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
				{{else}}
					<a href="{{.Name}}/">{{.Name}}/</a>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<a href="/{{.DownloadPath}}">zip download</a>&nbsp;&nbsp;
				{{end}}
				<a href="/{{.GenTempLink}}">temp link</a></li>
				</li>
			{{end}}
		</ul>
	</body>
</html>`

const homeHTML = `<!doctype html><link id=favicon rel="shortcut icon" type=image/png href=data:image/png;base64,AAABAAEAEBAQAAEABAAoAQAAFgAAACgAAAAQAAAAIAAAAAEABAAAAAAAgAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAXl1cAP///wArKysAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMhEREREREREyAAAAAAAAATIAAAAAAAABMgAAAAAAAAEyACAAAiIgATIAIAAAACABMgAgAAAAIAEyACIiAiIgATIAIAACAAABMgAgAAIAAAEyACIiAiIgATIAAAAAAAABMgAAAAAAAAEyAAAAAAAAATIiIiIiIiIiMzMzMzMzMzMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA><style>body{width:9px;height:9px;position:absolute;top:0;bottom:0;left:0;right:0;margin:auto}</style><title>&#65279;</title><a href=/walk/>walk</a>`

type Server struct {
	logFile       *os.File
	logLevel      int
	rootDir       string
	walkTemplate  *template.Template
	tempLinks     map[string]tempLink
	tempLinksLock sync.Mutex
}

func (s *Server) E404(w http.ResponseWriter, req *http.Request) {
	s.logger(LogWarning, reqToAuthReq(req), "E404")
	w.WriteHeader(http.StatusNotFound)
	_, err := fmt.Fprintf(w, "404\n")
	if err != nil {
		s.logger(LogError, reqToAuthReq(req), "E404-UnableToWriteResponse")
	}
}

func (s *Server) Home(w http.ResponseWriter, req *http.Request) {
	s.logger(LogInfo, reqToAuthReq(req), "Home")
	if req.URL.Path != "/" {
		s.E404(w, req)
		return
	}
	_, err := fmt.Fprint(w, homeHTML)
	if err != nil {
		s.logger(LogError, reqToAuthReq(req), "Home-UnableToWriteResponse")
	}
}

func (s *Server) Favicon(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
	s.logger(LogInfo, req, "Favicon")
	w.Header().Set("Content-Type", "image/x-icon")
	file, err := os.Open("favicon.ico")
	if err != nil {
		s.logger(LogWarning, req, "FaviconMissing")
		return
	}

	var ftime time.Time
	fileStat, err := os.Stat("favicon.ico")

	if err != nil {
		ftime = time.Time{}
	} else {
		ftime = fileStat.ModTime() // doesn't seem to actually set file dates
	}

	http.ServeContent(w, authReqToReq(req), "favicon.ico", ftime, file)
}

type entry struct {
	Name         string
	File         bool
	DownloadPath string
	GenTempLink  string
}

type walkData struct {
	Path    string
	Entries []entry
}

type tempLink struct {
	Path      string
	timeStamp int64
}

func (s *Server) Walk(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
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

	requestedFolder += "/"
	absPath += "/"

	data := walkData{
		Path:    requestedFolder,
		Entries: []entry{},
	}

	files, err := ioutil.ReadDir(absPath)
	if err != nil {
		s.logger(LogError, req, "Walk-readDir")
		log.Println(err)

		w.WriteHeader(http.StatusNotFound)
		_, err = fmt.Fprintf(w, "Either the requested directory doesn't exist or access was denied")
		if err != nil {
			s.logger(LogError, req, "UnableToWriteResponse")
		}
		return
	}
	for _, f := range files {
		data.Entries = append(data.Entries, entry{
			Name:         f.Name(),
			File:         !f.IsDir(),
			DownloadPath: path.Join("download", requestedFolder, f.Name()),
			GenTempLink:  path.Join("gettemplink", requestedFolder, f.Name()),
		})
	}

	sort.SliceStable(data.Entries, func(i, j int) bool {
		return strings.ToUpper(data.Entries[i].Name) < strings.ToUpper(data.Entries[j].Name)
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = s.walkTemplate.Execute(w, data)
	if err != nil {
		s.logger(LogError, req, "Walk-UnableToWriteResponse")
	}
}

func (s *Server) GetTempLink(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
	s.logger(LogInfo, req, "GetTempLink")
	_, _, err := s.checkThing(w, req)
	if err != nil {
		return
	}
	fileUUID := uuid.New().String()
	timeStamp := time.Now().Unix() + 60*60*48 // adds 48 hours to the link
	filePath := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	s.tempLinksLock.Lock()
	s.tempLinks[fileUUID] = tempLink{
		Path:      filePath,
		timeStamp: timeStamp,
	}
	s.tempLinksLock.Unlock()
	_, err = fmt.Fprintf(w, "File: %s\nTemporary link: https://%s\n\n\nOnly valid for 48 hours", filePath, path.Join(req.Host, "temp", fileUUID))
	if err != nil {
		s.logger(LogError, req, "GetTempLink-UnableToWriteResponse")
	}
	go s.linkClean()
}

func (s *Server) TempHandler(w http.ResponseWriter, req *http.Request) {
	s.logger(LogInfo, reqToAuthReq(req), "tempHandler")
	requestedUUID := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	s.tempLinksLock.Lock()
	linkInfo, ok := s.tempLinks[requestedUUID]
	s.tempLinksLock.Unlock()
	if !ok || linkInfo.timeStamp < time.Now().Unix() {
		s.E404(w, req)
		return
	}
	req.URL.Path = "/temp/" + linkInfo.Path
	s.Download(w, reqToAuthReq(req))
	go s.linkClean()
}

func (s *Server) linkClean() { // remove out of date links
	s.tempLinksLock.Lock()
	for k, v := range s.tempLinks {
		if v.timeStamp < time.Now().Unix() {
			delete(s.tempLinks, k)
		}
	}
	s.tempLinksLock.Unlock()
}

func (s *Server) checkThing(w http.ResponseWriter, req *auth.AuthenticatedRequest) (string, os.FileInfo, error) {
	requestedThing := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	absPath := path.Join(s.rootDir, requestedThing)

	fileInfo, statErr := os.Stat(absPath)
	if unix.Access(absPath, unix.R_OK) != nil || statErr != nil {
		s.logger(LogWarning, req, "PathAccessDenied")
		w.WriteHeader(http.StatusNotFound)
		_, respErr := fmt.Fprintf(w, "Either the requested item doesn't exist or access was denied")
		if respErr != nil {
			s.logger(LogError, req, "checkThing-UnableToWriteResponse")
		}
		return absPath, nil, statErr
	}
	return absPath, fileInfo, nil
}

func (s *Server) Download(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
	s.logger(LogInfo, req, "Download")
	absPath, fileInfo, err := s.checkThing(w, req)
	if err != nil {
		return
	}
	if fileInfo.IsDir() {
		s.downloadFolder(w, absPath)
	} else {
		s.downloadFile(w, req, absPath)
	}
}

func NewServer(rootDir string, logLevel int) *Server {
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
	newServer := &Server{logFile: logFile, logLevel: logLevel, rootDir: rootDir, walkTemplate: t, tempLinks: make(map[string]tempLink)}

	return newServer
}
