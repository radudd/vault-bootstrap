package bootstrap

import (
	"crypto/tls"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

// codes defined by /sys/health
// if any of those codes, Vault is up

func preflight(vaultPods []vaultPod) {
	c := make(chan string, len(vaultPods))
	for _, pod := range vaultPods {
		log.Debugf("Starting goroutine for %s", pod.name)
		go checkVaultStatus(pod, c)
	}
	for range vaultPods {
		log.Infof("%s is Running", <-c)
	}
}

func checkVaultStatus(pod vaultPod, c chan string) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	for {
		resp, err := http.Get(pod.fqdn + "/v1/sys/health")
		if err != nil {
			log.Debugf("%s: %s", pod.name, err.Error())
			time.Sleep(3 * time.Second)
			continue
		} else if !find(vaultReadyStatusCodes, resp.StatusCode) {
			log.Debugf("%s: HTTP Status %s", pod.name, strconv.Itoa(resp.StatusCode))
			time.Sleep(3 * time.Second)
			continue
		}
		c <- pod.name
		break
	}
}
