package backend

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

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

func (s *Server) E404(w http.ResponseWriter, req *http.Request) {
	s.logger.Warnw("Error 404", "request", reqToSafeStruct(req))
	w.WriteHeader(http.StatusNotFound)
	_, err := fmt.Fprintf(w, "404\n")
	if err != nil {
		s.logger.Errorw("Unable to respond", "request", reqToSafeStruct(req))
	}
}

func (s *Server) Home(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		s.E404(w, req)
		return
	}
	_, err := fmt.Fprint(w, homeHTML)
	if err != nil {
		s.logger.Errorw("Unable to respond", "request", reqToSafeStruct(req))
	}
}

func (s *Server) Favicon(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "image/x-icon")
	file, err := os.Open("favicon.ico")
	if err != nil {
		s.logger.Warnw("Favicon Missing", "request", reqToSafeStruct(req))
		return
	}
	defer file.Close()

	fileStat, err := os.Stat("favicon.ico")
	var ftime time.Time
	if err != nil {
		ftime = time.Time{}
	} else {
		ftime = fileStat.ModTime()
	}

	http.ServeContent(w, req, "favicon.ico", ftime, file)
}

func (s *Server) Walk(w http.ResponseWriter, req *http.Request) {
	_, absPath, err := s.checkThing(w, req)
	if err != nil {
		return
	}

	// requestedFolder for display
	requestedFolder := path.Join(strings.Split(req.URL.Path, "/")[2:]...) + "/"

	data := walkData{
		Path:    requestedFolder,
		Entries: []entry{},
	}

	files, err := ioutil.ReadDir(absPath + "/")
	if err != nil {
		s.logger.Errorw("Unable to read directory", "request", reqToSafeStruct(req), "error", err)
		w.WriteHeader(http.StatusNotFound)
		_, respErr := fmt.Fprintf(w, "Either the requested directory doesn't exist or access was denied")
		if respErr != nil {
			s.logger.Errorw("Unable to respond", "request", reqToSafeStruct(req))
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
		s.logger.Errorw("Unable to respond", "request", reqToSafeStruct(req))
	}
}

func (s *Server) GetTempLink(w http.ResponseWriter, req *http.Request) {
	_, _, err := s.checkThing(w, req)
	if err != nil {
		return
	}
	fileUUID := uuid.New().String()
	timeStamp := time.Now().Add(time.Hour * time.Duration(s.tempLinksHours)).Unix()
	filePath := path.Join(strings.Split(req.URL.Path, "/")[2:]...)

	s.tempLinksLock.Lock()
	s.tempLinks[fileUUID] = tempLink{
		Path:      filePath,
		timeStamp: timeStamp,
	}
	s.tempLinksLock.Unlock()

	// Build full URL for the user
	tempLinkURL := "https://" + path.Join(req.Host, strings.TrimPrefix(s.cfg.TempLinkBase, "/"), fileUUID)
	_, err = fmt.Fprintf(w, "File: %s\nTemporary link: %s\n\n\nOnly valid for %d hours",
		filePath, tempLinkURL, s.tempLinksHours)
	if err != nil {
		s.logger.Errorw("Unable to respond", "request", reqToSafeStruct(req))
	}
	go s.linkClean()
}

func (s *Server) TempHandler(w http.ResponseWriter, req *http.Request) {
	requestedUUID := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	s.tempLinksLock.RLock()
	linkInfo, ok := s.tempLinks[requestedUUID]
	s.tempLinksLock.RUnlock()
	if !ok || time.Unix(linkInfo.timeStamp, 0).Before(time.Now()) {
		s.E404(w, req)
		return
	}
	req.URL.Path = "/temp/" + linkInfo.Path
	s.Download(w, req)
	go s.linkClean()
}

func (s *Server) linkClean() { // remove out of date links
	s.tempLinksLock.Lock()
	for k, v := range s.tempLinks {
		if time.Unix(v.timeStamp, 0).Before(time.Now()) {
			delete(s.tempLinks, k)
		}
	}
	s.tempLinksLock.Unlock()
}

func (s *Server) checkThing(w http.ResponseWriter, req *http.Request) (string, string, error) {
	requestedThing := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	absPath := path.Join(s.rootDir, requestedThing)

	_, statErr := os.Stat(absPath)
	if statErr != nil {
		s.logger.Warnw("Access to path denied or path does not exist",
			"request", reqToSafeStruct(req), "error", statErr)
		w.WriteHeader(http.StatusNotFound)
		_, respErr := fmt.Fprintf(w, "Either the requested item doesn't exist or access was denied")
		if respErr != nil {
			s.logger.Errorw("Unable to respond", "request", reqToSafeStruct(req))
		}
		return absPath, "", statErr
	}

	return requestedThing, absPath, nil
}

func (s *Server) Download(w http.ResponseWriter, req *http.Request) {
	_, absPath, err := s.checkThing(w, req)
	if err != nil {
		return
	}
	fileInfo, _ := os.Stat(absPath)
	if fileInfo.IsDir() {
		s.downloadFolder(w, absPath)
	} else {
		s.downloadFile(w, req, absPath)
	}
}
