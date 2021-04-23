package bootstrap

import (
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	vault "github.com/hashicorp/vault/api"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// unseal interval
// why vaultPod is not recognized
// access by using capitals - tune
// test

const (
	DefaultUnsealInterval = 180
)

var (
	unsealInterval int
)

func init() {
	if extrUnsealInterval, ok := os.LookupEnv("UNSEAL_INTERVAL"); !ok {
		log.Warn("UNSEAL_INTERVAL not set. Defaulting to ", DefaultUnsealInterval)
		unsealInterval = DefaultUnsealInterval
	} else {
		intExtrUnsealInterval, err := strconv.Atoi(extrUnsealInterval)
		if err != nil {
			log.Warnf("UNSEAL_INTERVAL must be set to an integer value. Defaulting to ", DefaultUnsealInterval)
			return
		}
		unsealInterval = intExtrUnsealInterval
	}
}
func Unseal() {

	time.Sleep(20 * time.Second)
	//k8sConfig, _ := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
	// Create clientSet for k8s client-go
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	clientsetK8s, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	// Get unseal keys
	var unsealKeys []string
	unsealKeysString, err := getValuesFromK8sSecret(clientsetK8s, vaultSecretUnseal)
	if err != nil {
		log.Fatalf("Cannot load Unseal Keys from secret %s and key %s", vaultSecretUnseal, "vaultData")
	}
	log.Debugf("Unseal keys: %s", *unsealKeysString)
	unsealKeys = strings.Split(*unsealKeysString, ";")

	// Define Vault pod
	var pod vaultPod
	insecureTLS := &vault.TLSConfig{
		Insecure: true,
	}
	pod.fqdn = "https://localhost:8200"
	pod.name = "localhost:8200"
	clientConfig := &vault.Config{
		Address: pod.fqdn,
	}
	clientConfig.ConfigureTLS(insecureTLS)

	client, err := vault.NewClient(clientConfig)
	if err != nil {
		os.Exit(1)
	}
	pod.client = client
	for {
		_ = unsealMember(pod, unsealKeys)
		time.Sleep(time.Duration(unsealInterval) * time.Second)
	}
}
