package bootstrap

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func getValuesFromK8sSecret(clientsetK8s *kubernetes.Clientset) (*string, *[]string, error) {
	secretClient := clientsetK8s.CoreV1().Secrets(namespace)
	// Check if secret exists
	secretVault, err := secretClient.Get(context.TODO(), "vault", metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	rootToken := string(secretVault.Data["rootToken"])
	unsealKeys := strings.Split(string(secretVault.Data["unsealKeys"]), ";")
	return &rootToken, &unsealKeys, nil
}

func createK8sSecret(rootToken *string, unsealKeys *[]string, clientsetK8s *kubernetes.Clientset) error {

	secretClient := clientsetK8s.CoreV1().Secrets(namespace)
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vault",
		},
		Type: apiv1.SecretTypeOpaque,
		StringData: map[string]string{
			"rootToken":  *rootToken,
			"unsealKeys": strings.Join(*unsealKeys, ";"),
		},
	}

	result, err := secretClient.Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Info("Created K8s secret ", result.GetObjectMeta().GetName())
	return nil
}
