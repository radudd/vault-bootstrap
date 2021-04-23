package bootstrap

import (
	"context"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetValuesFromK8sSecret(clientsetK8s *kubernetes.Clientset, secretName string) (*string, error) {
	secretClient := clientsetK8s.CoreV1().Secrets(namespace)
	// Check if secret exists
	secretVault, err := secretClient.Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		log.Debug("K8s Secret not found")
		return nil, err
	}
	vaultSecretData := string(secretVault.Data["vaultData"])
	//unsealKeys := strings.Split(string(secretVault.Data["unsealKeys"]), ";")
	return &vaultSecretData, nil
}

//func createK8sSecret(rootToken *string, unsealKeys *[]string, clientsetK8s *kubernetes.Clientset) error {
func createK8sSecret(clientsetK8s *kubernetes.Clientset, secretName *string, vaultSecretData *string) error {
	secretClient := clientsetK8s.CoreV1().Secrets(namespace)
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: *secretName,
		},
		Type: apiv1.SecretTypeOpaque,
		StringData: map[string]string{
			"vaultData": *vaultSecretData,
			//"unsealKeys": strings.Join(*unsealKeys, ";"),
		},
	}

	result, err := secretClient.Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Info("Created K8s secret ", result.GetObjectMeta().GetName())
	return nil
}
