package bootstrap

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func preflight(clientsetK8s *kubernetes.Clientset, namespace string) {
	vaultPodsUrls := strings.Split(vaultClusterMembers, ",")
	storagePodsUrls := strings.Split(storageClusterMembers, ",")
	podsUrls := append(vaultPodsUrls, storagePodsUrls...)
	c := make(chan string, len(podsUrls))
	for _, podUrl := range podsUrls {
		podFqdn, err := url.Parse(podUrl)
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
		podName := strings.Split(podFqdn.Hostname(), ".")[0]
		log.Debugf("Starting goroutine for %s", podName)
		go checkPodPhase(podName, clientsetK8s, namespace, c)
	}

	for range podsUrls {
		log.Infof("%s is Running", <-c)
		log.Debugf("Current buffer size is %s", strconv.Itoa(len(c)))
	}

}
func checkPodPhase(podName string, clientsetK8s *kubernetes.Clientset, namespace string, c chan string) {
	for {
		podClient, _ := clientsetK8s.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if podClient.Status.Phase != "Running" {
			log.Infof("%s NOT READY. Waiting...", podName)
			time.Sleep(3 * time.Second)
			continue
		}
		c <- podName
		break
	}
}
