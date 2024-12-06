package backend

import (
	"flag"
	"os"
	"strconv"

	"go.uber.org/zap/zapcore"
)

type Config struct {
	RootDir      string
	Port         string
	CertFile     string
	KeyFile      string
	ExpireHours  int
	BasicUser    string
	BasicPass    string
	TempLinkBase string
	LogLevel     zapcore.Level
}

func LoadConfig() (*Config, error) {
	// Environment variables with fallback to flags
	rootDirEnv := getEnv("ROOT_DIR", "./temp")
	portEnv := getEnv("PORT", "8090")
	certEnv := getEnv("CERT_FILE", "./localhost.crt")
	keyEnv := getEnv("KEY_FILE", "./localhost.key")
	expireEnv := getEnv("TEMP_LINK_EXPIRE", "48")
	userEnv := os.Getenv("BASIC_USER")
	passEnv := os.Getenv("BASIC_PASS")
	tempBaseEnv := getEnv("TEMP_LINK_BASE", "/temp")

	expireHours, err := strconv.Atoi(expireEnv)
	if err != nil {
		expireHours = 48
	}

	// If you want flags as well:
	rootDirFlag := flag.String("root", rootDirEnv, "The root directory for the hosted files.")
	portFlag := flag.String("port", portEnv, "Port number to listen on.")
	certFlag := flag.String("cert", certEnv, "Cert file for TLS")
	keyFlag := flag.String("key", keyEnv, "Key file for TLS")
	expireFlag := flag.Int("expire", expireHours, "Hours until temporary links expire")
	flag.Parse()

	cfg := &Config{
		RootDir:      *rootDirFlag,
		Port:         *portFlag,
		CertFile:     *certFlag,
		KeyFile:      *keyFlag,
		ExpireHours:  *expireFlag,
		BasicUser:    userEnv,
		BasicPass:    passEnv,
		TempLinkBase: tempBaseEnv,
		LogLevel:     zapcore.DebugLevel,
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
