package bootstrap

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// codes defined by /sys/health
// if any of those codes, Vault is up

func preflight() {
	podsUrls := strings.Split(vaultClusterMembers, ",")
	c := make(chan string, len(podsUrls))
	for _, podUrl := range podsUrls {
		log.Debugf("Starting goroutine for %s", podUrl)
		go checkVaultStatus(podUrl, c)
		log.Debugf("Before: Current buffer size is %s", strconv.Itoa(len(c)))
	}
	for range podsUrls {
		log.Infof("%s is Running", <-c)
		log.Debugf("After: Current buffer size is %s", strconv.Itoa(len(c)))
	}
}

func checkVaultStatus(podUrl string, c chan string) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	podFqdn, _ := url.Parse(podUrl)
	podName := strings.Split(podFqdn.Hostname(), ".")[0]
	for {
		resp, err := http.Get(podUrl + "/v1/sys/health")
		if err != nil {
			log.Debugf("%s: %s", podName, err.Error())
			time.Sleep(3 * time.Second)
			continue
		} else if !find(vaultReadyStatusCodes, resp.StatusCode) {
			log.Debugf("%s: HTTP Status %s", podName, strconv.Itoa(resp.StatusCode))
			time.Sleep(3 * time.Second)
			continue
		}
		c <- podUrl
		break
	}
}
