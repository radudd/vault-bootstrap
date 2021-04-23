package main

import (
	"flag"
	"os"
	"runtime"
	"strings"

	"github.com/radudd/vault-bootstrap/internal/bootstrap"
	"github.com/radudd/vault-bootstrap/internal/unsealer"
	log "github.com/sirupsen/logrus"
)

func main() {

	runningMode := flag.String("mode", "job", "running mode: job or sidecar")
	flag.Parse()

	if *runningMode == "job" {
		log.Info("Running in job mode...")
		bootstrap.Run()
	} else if *runningMode == "sidecar" {
		log.Info("Running in sidecar mode...")
		unsealer.Run()
	} else {
		panic("Running mode must be 'sidecar' or 'job'")
	}
}

func init() {

	const DefaultLogLevel = "Info"

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
