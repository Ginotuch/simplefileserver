package backend

import (
	"fmt"
	"log"
	"time"

	auth "github.com/abbot/go-http-auth"
)

const (
	LogDebug   = 0
	LogInfo    = 1
	LogWarning = 2
	LogError   = 3
)

var LogNames = [4]string{"DEBUG", "INFO", "WARNING", "ERROR"}

func (s *Server) logger(level int, req *auth.AuthenticatedRequest, caller string) {
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
