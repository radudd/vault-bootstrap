package main

import (
	"os"
	"runtime"
	"strings"

	"github.com/radudd/vault-bootstrap/internal/bootstrap"
	log "github.com/sirupsen/logrus"
)

func main() {
	bootstrap.Run()
}

func init() {

	const DefaultLogLevel = "Debug"

	logLevel, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logLevel = DefaultLogLevel
	}
	level, err := log.ParseLevel(strings.Title(logLevel))
	if err != nil {
		return
	}

	// Output everything including stderr to stdout
	log.SetOutput(os.Stdout)

	// set level
	log.SetLevel(level)
	log.Info("LogLevel set to " + level.String())

	log.Info(runtime.Version())
}
