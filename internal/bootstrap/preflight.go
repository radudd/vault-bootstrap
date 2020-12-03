package bootstrap

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func preflight(clientsetK8s *kubernetes.Clientset) {
	pods, _ := clientsetK8s.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	for _, avPod := range pods.Items {
		log.Debug(avPod.ObjectMeta.GenerateName)
	}
	podsUrls := strings.Split(vaultClusterMembers, ",")
	if storageClusterMembers != "" {
		storagePodsUrls := strings.Split(storageClusterMembers, ",")
		podsUrls = append(podsUrls, storagePodsUrls...)
	}
	log.Debugf("Namespace: %s", namespace)
	c := make(chan string, len(podsUrls))
	for _, podUrl := range podsUrls {
		podFqdn, err := url.Parse(podUrl)
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
		podName := strings.Split(podFqdn.Hostname(), ".")[0]
		log.Debugf("Starting goroutine for %s", podName)
		go checkPodPhase(podName, clientsetK8s, c)
	}

	for range podsUrls {
		log.Infof("%s is Running", <-c)
		log.Debugf("Current buffer size is %s", strconv.Itoa(len(c)))
	}

}
func checkPodPhase(podName string, clientsetK8s *kubernetes.Clientset, c chan string) {
	for {
		podClient, _ := clientsetK8s.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if podClient.Status.Phase != apiv1.PodRunning {
			log.Infof("%s NOT ready. Waiting...", podName)
			time.Sleep(3 * time.Second)
			continue
		}
		c <- podName
		break
	}
}
