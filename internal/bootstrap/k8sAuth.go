package bootstrap

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

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
		return fmt.Errorf("Cant't get vault service account - ", err.Error())
	}

	secretSaVaultName := saClientVault.Secrets[0].Name
	log.Info("Token secret for vault: ", secretSaVaultName)

	secretSaVault, err := clientsetK8s.CoreV1().Secrets(namespace).Get(context.TODO(), secretSaVaultName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Cant't get secret for vault service account - ", err.Error())
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
		return fmt.Errorf("Invalid Kubernetes API config")
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
	log.Info("Vault K8S authentication enabled")
	return nil
}
