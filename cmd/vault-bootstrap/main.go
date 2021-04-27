package main

import (
	"flag"
	"os"
	"runtime"
	"strings"

	"github.com/radudd/vault-bootstrap/internal/bootstrap"
	log "github.com/sirupsen/logrus"
)

func main() {

	/*
		if os.Args[1] == "job" || len(os.Args) == 0 {
			log.Info("Running in job mode...")
			bootstrap.Run()
		} else if os.Args[1] == "init-container" {
			log.Info("Running in init-container mode...")
			bootstrap.Init()
		} else {
			panic("First argument(running mode) must be 'job' or 'init-container'")
		}
		//runningMode := flag.String("mode", "job", "running mode: job or init-container")
		httpCheckUrl := flag.String("http-check", "http://www.redhat.com", "http check mode")
		flag.Parse()
		flag.Visit(func(f *flag.Flag){
			if f.Name == "http-check" {
				httpCheck.Check(*httpCheckUrl)
			}
		})
	*/
	runningMode := flag.String("mode", "job", "running mode: job or init-container")
	flag.Parse()
	if *runningMode == "job" {
		log.Info("Running in job mode...")
		bootstrap.Run()
	} else if *runningMode == "init-container" {
		log.Info("Running in init-container mode...")
		bootstrap.InitContainer()
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
