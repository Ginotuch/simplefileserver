package backend

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	auth "github.com/abbot/go-http-auth"
)

func (s *Server) downloadFolder(w http.ResponseWriter, absPath string) {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", path.Base(absPath)))
	zipWriter := zip.NewWriter(w)

	walkErr := filepath.Walk(absPath, func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		zipPath := path.Join(strings.Split(filePath, "/")[len(strings.Split(absPath, "/"))-1:]...)
		s.logger.Debugw("Zipping file to folder", "folder", absPath, "file", zipPath)
		fileWriter, err := zipWriter.CreateHeader(&zip.FileHeader{Name: zipPath, Method: zip.Store})
		if err != nil {
			s.logger.Errorw("Zipping file to folder failed", "folder", absPath, "file", zipPath, "error", err)
		}
		fileReader, err := os.Open(filePath)
		_, err = io.Copy(fileWriter, fileReader)
		if err != nil {
			s.logger.Errorw("Zip write failed (most likely download stopped by user)", "folder", absPath, "file", zipPath, "error", err)
		}
		err = fileReader.Close()
		if err != nil {
			s.logger.Errorw("Closing file being added to zip failed", "folder", absPath, "file", zipPath, "error", err)
		}
		return nil
	})
	if walkErr != nil {
		s.logger.Errorw("Walk error", "folder", absPath, "error", walkErr)
	}
	err := zipWriter.Close()
	if err != nil {
		s.logger.Errorw("Failed to close zip file", "folder", absPath, "error", walkErr)
	}
}

func (s *Server) downloadFile(w http.ResponseWriter, req *auth.AuthenticatedRequest, absPath string) {
	file, err := os.Open(absPath)
	if err != nil {
		s.logger.Warnw("", "request", reqToSafeStruct(authReqToReq(req)), "error", err)
		_, err = fmt.Fprintf(w, "Unable to get file")
		if err != nil {
			s.logger.Errorw(logUnableToRespond, "request", reqToSafeStruct(authReqToReq(req)))
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

	http.ServeContent(w, authReqToReq(req), path.Base(req.URL.Path), ftime, file)
	_ = file.Close()
}
