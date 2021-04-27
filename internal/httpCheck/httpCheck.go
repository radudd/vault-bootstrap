package httpCheck

import (
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

func Check(url string) {
	for {
		resp, err := http.Get(url)
		if err != nil {
			panic(err.Error())
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 399 {
			log.Info("Service UP. Status code: ", resp.StatusCode)
			os.Exit(0)
		}
		log.Error("Service DOWN. Status code: ", resp.StatusCode)
		os.Exit(99)
	}
}
