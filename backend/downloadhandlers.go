package backend

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	auth "github.com/abbot/go-http-auth"
)

func (s *ServerStruct) downloadFolder(w http.ResponseWriter, absPath string) {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", path.Base(absPath)))
	zipWriter := zip.NewWriter(w)

	walkErr := filepath.Walk(absPath, func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		zipPath := path.Join(strings.Split(filePath, "/")[len(strings.Split(absPath, "/"))-1:]...)
		fmt.Printf("[%s][Zipping] folder: \"%s\" file: \"%s\"\n", time.Now().Format("2006-01-02 15:04:05"), absPath, zipPath)
		fileWriter, err := zipWriter.CreateHeader(&zip.FileHeader{Name: zipPath, Method: zip.Store})
		if err != nil {
			fmt.Printf("[%s][ERROR][Zipping] FAILED folder: \"%s\" file: \"%s\"\n", time.Now().Format("2006-01-02 15:04:05"), absPath, zipPath)
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

func (s *ServerStruct) downloadFile(w http.ResponseWriter, req *auth.AuthenticatedRequest, absPath string) {
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

	http.ServeContent(w, authReqToReq(req), path.Base(req.URL.Path), ftime, file)
	_ = file.Close()
}
