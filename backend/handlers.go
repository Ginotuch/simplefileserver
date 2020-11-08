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

	auth "github.com/abbot/go-http-auth"
	"github.com/google/uuid"
)

func (s *Server) E404(w http.ResponseWriter, req *http.Request) {
	s.logger.Warnw(log404, "request", reqToSafeStruct(req))
	w.WriteHeader(http.StatusNotFound)
	_, err := fmt.Fprintf(w, "404\n")
	if err != nil {
		s.logger.Errorw(logUnableToRespond, "request", reqToSafeStruct(req))
	}
}

func (s *Server) Home(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		s.E404(w, req)
		return
	}
	_, err := fmt.Fprint(w, homeHTML)
	if err != nil {
		s.logger.Errorw(logUnableToRespond, "request", reqToSafeStruct(req))
	}
}

func (s *Server) Favicon(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
	w.Header().Set("Content-Type", "image/x-icon")
	file, err := os.Open("favicon.ico")
	if err != nil {
		s.logger.Warnw("Favicon Missing", "request", reqToSafeStruct(authReqToReq(req)))
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
	timeStamp time.Time
}

func (s *Server) Walk(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
	requestedFolder := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	absPath, _, err := s.checkThing(w, req)
	if err != nil {
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
		s.logger.Errorw(logUnableToReadDir, "request", reqToSafeStruct(authReqToReq(req)), "error", err)

		w.WriteHeader(http.StatusNotFound)
		_, err = fmt.Fprintf(w, "Either the requested directory doesn't exist or access was denied")
		if err != nil {
			s.logger.Errorw(logUnableToRespond, "request", reqToSafeStruct(authReqToReq(req)))
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
		s.logger.Errorw(logUnableToRespond, "request", reqToSafeStruct(authReqToReq(req)))
	}
}

func (s *Server) GetTempLink(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
	_, _, err := s.checkThing(w, req)
	if err != nil {
		return
	}
	fileUUID := uuid.New().String()
	timeStamp := time.Now().Add(time.Hour * time.Duration(s.tempLinksHours))
	filePath := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	s.tempLinksLock.Lock()
	s.tempLinks[fileUUID] = tempLink{
		Path:      filePath,
		timeStamp: timeStamp,
	}
	s.tempLinksLock.Unlock()
	_, err = fmt.Fprintf(w, "File: %s\nTemporary link: https://%s\n\n\nOnly valid for %d hours", filePath, path.Join(req.Host, "temp", fileUUID), s.tempLinksHours)
	if err != nil {
		s.logger.Errorw(logUnableToRespond, "request", reqToSafeStruct(authReqToReq(req)))
	}
	go s.linkClean()
}

func (s *Server) TempHandler(w http.ResponseWriter, req *http.Request) {
	requestedUUID := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	s.tempLinksLock.RLock()
	linkInfo, ok := s.tempLinks[requestedUUID]
	s.tempLinksLock.RUnlock()
	if !ok || linkInfo.timeStamp.Before(time.Now()) {
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
		if v.timeStamp.Before(time.Now()) {
			delete(s.tempLinks, k)
		}
	}
	s.tempLinksLock.Unlock()
}

func (s *Server) checkThing(w http.ResponseWriter, req *auth.AuthenticatedRequest) (string, os.FileInfo, error) {
	requestedThing := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	absPath := path.Join(s.rootDir, requestedThing)

	fileInfo, statErr := os.Stat(absPath)
	fileHandle, openErr := os.Open(absPath)
	if openErr != nil || statErr != nil {
		if fileHandle != nil {
			_ = fileHandle.Close()
		}
		s.logger.Warnw(logPathDenied,
			"request", reqToSafeStruct(authReqToReq(req)), "statErr", statErr, "openErr", openErr)
		w.WriteHeader(http.StatusNotFound)
		_, respErr := fmt.Fprintf(w, "Either the requested item doesn't exist or access was denied")
		if respErr != nil {
			s.logger.Errorw(logUnableToRespond,
				"request", reqToSafeStruct(authReqToReq(req)))
		}
		return absPath, nil, statErr
	}
	err := fileHandle.Close()
	if err != nil {

	}
	return absPath, fileInfo, nil
}

func (s *Server) Download(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
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
