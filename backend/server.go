package backend

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	auth "github.com/abbot/go-http-auth"
	"go.uber.org/zap/zapcore"

	"go.uber.org/zap"
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

const (
	logPathDenied      = "Access to path denied"
	logUnableToRespond = "Unable to write response"
	log404             = "Error 404"
	logUnableToReadDir = "Unable to read directory"
)

type Server struct {
	mux           *http.ServeMux
	logger        *zap.SugaredLogger
	logLevel      int
	rootDir       string
	walkTemplate  *template.Template
	tempLinks     map[string]tempLink
	tempLinksLock sync.Mutex
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.logger.Infow("", "request", reqToJson(req))
	s.mux.ServeHTTP(w, req)
}

func (s *Server) closeHandlerSetup() { // silently close without printing on ctrl+c signal interrupt
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		s.logger.Infow("simplefileserver shutting down")
		_ = s.logger.Sync()
		os.Exit(0)
	}()
}

func NewServer(rootDir string, logLevel zapcore.Level) *Server {
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = append(cfg.OutputPaths, "./log.log")
	cfg.ErrorOutputPaths = append(cfg.ErrorOutputPaths, "./log.log")
	cfg.Level = zap.NewAtomicLevelAt(logLevel)
	plain, err := cfg.Build()
	if err != nil {
		log.Fatal("Unable to create zap logger")
	}
	logger := plain.Sugar()
	logger.Infow("simplefileserver starting up")

	t, err := template.New("walkHTML").Parse(walkTemplate)
	if err != nil {
		logger.Fatalw("Failed to parse walkHTML template")
	}

	newServer := &Server{mux: http.NewServeMux(), logger: logger, rootDir: rootDir, walkTemplate: t, tempLinks: make(map[string]tempLink)}

	authenticator := auth.NewBasicAuthenticator("simplefileserver", auth.HtdigestFileProvider(".htdigest"))

	newServer.mux.HandleFunc("/download/", authenticator.Wrap(newServer.Download))
	newServer.mux.HandleFunc("/gettemplink/", authenticator.Wrap(newServer.GetTempLink))
	newServer.mux.HandleFunc("/temp/", newServer.TempHandler)
	newServer.mux.HandleFunc("/walk/", authenticator.Wrap(newServer.Walk))
	newServer.mux.HandleFunc("/favicon.ico", authenticator.Wrap(newServer.Favicon))
	newServer.mux.HandleFunc("/", newServer.Home)

	newServer.closeHandlerSetup()
	logger.Infow("simpleifleserver startup complete")
	return newServer
}
