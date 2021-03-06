package bootstrap

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func checkVaultUp(client *vault.Client) (bool) {
	for i := 0; i < 5; i++ {
		hr, err := client.Sys().Health()
		if err != nil {
			log.Warn(err.Error(), "K8s authentication: Retrying in 3 seconds...")
			time.Sleep(3 * time.Second)
			continue
		}
		if !hr.Initialized || hr.Sealed {
			log.Warn("K8s authentication: Vault not Initialized/Unsealed. Retrying ...")
			time.Sleep(3 * time.Second)
			continue
		}
		return true
	}
	return false
}
func checkK8sAuth(client *vault.Client) (bool, error) {
	auths, err := client.Logical().Read("sys/auth")
	if err != nil {
		return false, err
	}
	if k8sAuth, _ := auths.Data["kubernetes/"]; k8sAuth != nil {
		return true, nil
	}
	return false, nil
}
func configureK8sAuth(client *vault.Client, clientsetK8s *kubernetes.Clientset) error {

	// Enable K8S authentication
	err := client.Sys().EnableAuthWithOptions("kubernetes/", &vault.EnableAuthOptions{
		Type: "kubernetes",
	})

	if err != nil {
		return err
	}

	saClient := clientsetK8s.CoreV1().ServiceAccounts(namespace)
	saClientVault, err := saClient.Get(context.TODO(), vaultServiceAccount, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("K8s authentication: Cant't get vault service account - ", err.Error())
	}

	secretSaVaultName := saClientVault.Secrets[0].Name
	log.Info("Token secret for vault: ", secretSaVaultName)

	secretSaVault, err := clientsetK8s.CoreV1().Secrets(namespace).Get(context.TODO(), secretSaVaultName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("K8s authentication: Cant't get secret for vault service account - ", err.Error())
	}
	vaultJwt := secretSaVault.Data["token"]

	// Fetch CA for connecting to K8S API
	cacert, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return err
	}

	// Get K8S API URL
	k8sApiHost, ok := os.LookupEnv("KUBERNETES_PORT_443_TCP_ADDR")
	if !ok {
		return fmt.Errorf("K8s authentication: Invalid Kubernetes API config")
	}

	k8sApiUrl := fmt.Sprintf("https://%s:443", k8sApiHost)

	// Prepare payload for configuring k8s authentication
	data := map[string]interface{}{
		"kubernetes_host":    k8sApiUrl,
		"kubernetes_ca_cert": string(cacert),
		"token_reviewer_jwt": string(vaultJwt),
	}

	// Configure K8S authentication
	_, err = client.Logical().Write("auth/kubernetes/config", data)
	if err != nil {
		return err
	}
	log.Info("K8s authentication: Successfully enabled")
	return nil
}
