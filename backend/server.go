package backend

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type tempLink struct {
	Path      string
	timeStamp int64 // store as Unix time for simplicity
}

type Server struct {
	mux            *http.ServeMux
	logger         *zap.SugaredLogger
	logLevel       zapcore.Level
	rootDir        string
	walkTemplate   *template.Template
	tempLinks      map[string]tempLink
	tempLinksHours int
	tempLinksLock  sync.RWMutex

	cfg *Config
}

func NewServer(cfg *Config) (*Server, error) {
	zcfg := zap.NewProductionConfig()
	zcfg.OutputPaths = append(zcfg.OutputPaths, "./log.log")
	zcfg.ErrorOutputPaths = append(zcfg.ErrorOutputPaths, "./log.log")
	zcfg.Level = zap.NewAtomicLevelAt(cfg.LogLevel)
	plain, err := zcfg.Build()
	if err != nil {
		log.Fatal("Unable to create zap logger")
	}
	logger := plain.Sugar()
	logger.Infow("simplefileserver starting up")

	t, err := template.New("walkHTML").Parse(walkTemplateString)
	if err != nil {
		logger.Fatalw("Failed to parse walkHTML template")
	}

	newServer := &Server{
		mux:            http.NewServeMux(),
		logger:         logger,
		rootDir:        cfg.RootDir,
		walkTemplate:   t,
		tempLinks:      make(map[string]tempLink),
		tempLinksHours: cfg.ExpireHours,
		cfg:            cfg,
	}

	newServer.registerHandlers()

	newServer.closeHandlerSetup()
	logger.Infow("simplefileserver startup complete")
	return newServer, nil
}

func (s *Server) registerHandlers() {
	// Wrap handlers with auth if needed
	s.mux.Handle("/download/", s.authWrapper(http.HandlerFunc(s.Download)))
	s.mux.Handle("/gettemplink/", s.authWrapper(http.HandlerFunc(s.GetTempLink)))
	s.mux.Handle(s.cfg.TempLinkBase+"/", http.HandlerFunc(s.TempHandler))
	s.mux.Handle("/walk/", s.authWrapper(http.HandlerFunc(s.Walk)))
	s.mux.Handle("/favicon.ico", s.authWrapper(http.HandlerFunc(s.Favicon)))
	s.mux.Handle("/", http.HandlerFunc(s.Home))
}

func (s *Server) authWrapper(next http.Handler) http.Handler {
	// Apply Basic Auth only if configured and path is protected
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isProtectedPath(r.URL.Path) {
			BasicAuthMiddleware(s.cfg.BasicUser, s.cfg.BasicPass, next).ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.logger.Infow("", "request", reqToSafeStruct(req))
	s.mux.ServeHTTP(w, req)
}

func (s *Server) closeHandlerSetup() { // silently close without printing on ctrl+c signal interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		s.logger.Infow("simplefileserver shutting down")
		_ = s.logger.Sync()
		os.Exit(0)
	}()
}
