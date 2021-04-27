package bootstrap

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Init() {
	// Create clientSet for k8s client-go
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	//k8sConfig, _ := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")

	clientsetK8s, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	// Define Parameters
	podName, ok := os.LookupEnv("VAULT_K8S_POD_NAME")
	if !ok {
		panic("Cannot extract Pod name from environment variables")
	}
	// eventually parametrize http scheme and port
	podUrl := "https://" + podName + ":8200"
	namespace, ok := os.LookupEnv("VAULT_K8S_NAMESPACE")
	if !ok {
		panic("Cannot extract Namespace name from environment variables")
	}
	pod, err := clientsetK8s.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Error(err.Error())
		panic("Cannot extract Pod information from Kubernetes API")
	}
	containerImage := pod.Status.InitContainerStatuses[0].Image

	randomString := strings.Replace(uuid.New().String(), "-", "", -1)
	jobName := podName + "-usealer-" + randomString[0:4]

	// Define Job
	jobClient := clientsetK8s.BatchV1().Jobs(namespace)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      "OnFailure",
					ServiceAccountName: vaultServiceAccount,
					Containers: []corev1.Container{
						{
							Name:  "unsealer",
							Image: containerImage,
							//Command: []string{"sh", "-xc"},
							Env: []corev1.EnvVar{
								{
									Name:  "VAULT_ENABLE_INIT",
									Value: "False",
								},
								{
									Name:  "VAULT_ENABLE_K8SSECRET",
									Value: "False",
								},
								{
									Name:  "VAULT_ENABLE_UNSEAL",
									Value: "True",
								},
								{
									Name:  "VAULT_ENABLE_K8SAUTH",
									Value: "False",
								},
								{
									Name:  "VAULT_CLUSTER_MEMBERS",
									Value: podUrl,
								},
								{
									Name:  "VAULT_KEY_SHARES",
									Value: strconv.Itoa(vaultKeyShares),
								},
								{
									Name:  "VAULT_KEY_THRESHOLD",
									Value: strconv.Itoa(vaultKeyThreshold),
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := jobClient.Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		log.Error(err.Error())
		panic("Failed to create Vault Unseal Job")
	}
	log.Info("Created Vault Unseal Job ", result.GetObjectMeta().GetName())
}
