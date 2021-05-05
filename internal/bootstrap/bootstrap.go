package bootstrap

import (
	"context"
	"net/url"
	"os"
	"strings"
	"time"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Run Vault bootstrap
func Run() {

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

	podsList, err := clientsetK8s.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	var pdList []string
	for _, pd := range podsList.Items {
		pdList = append(pdList, getPodName(&pd))
	}
	log.Debugf("Pods list: %s", strings.Join(pdList, ";"))

	// Define Vault client for Vault LB
	clientConfigLB := vault.DefaultConfig()
	// Skip TLS verification for initialization
	insecureTLS := &vault.TLSConfig{
		Insecure: true,
	}
	clientConfigLB.ConfigureTLS(insecureTLS)

	clientLB, err := vault.NewClient(clientConfigLB)
	if err != nil {
		os.Exit(1)
	}

	// Slice of maps containing vault pods details
	var vaultPods []vaultPod
	vaultMembersUrls := strings.Split(vaultClusterMembers, ",")
	// Generate the slice from Env variable
	for _, member := range vaultMembersUrls {
		var pod vaultPod
		podFqdn, _ := url.Parse(member)
		pod.fqdn = member
		pod.name = strings.Split(podFqdn.Hostname(), ".")[0]
		// Define main client (vault-0) which will be used for initialization
		// When using integrated RAFT storage, the vault cluster member that is initialized
		// needs to be first one which is unsealed
		// In the unseal part we'll always start with the first member
		clientConfig := &vault.Config{
			Address: pod.fqdn,
		}
		clientConfig.ConfigureTLS(insecureTLS)

		client, err := vault.NewClient(clientConfig)
		if err != nil {
			os.Exit(1)
		}
		pod.client = client
		vaultPods = append(vaultPods, pod)
	}
	// Define main client (vault-0) which will be used for initialization
	// When using integrated RAFT storage, the vault cluster member that is initialized
	// needs to be first one which is unsealed
	// In the unseal part we'll always start with the first member
	vaultFirstPod := vaultPods[0]
	preflight(vaultPods)

	time.Sleep(5 * time.Second)

	var rootToken *string
	var unsealKeys *[]string

	pVaultSecretRoot := &vaultSecretRoot
	pVaultSecretUnseal := &vaultSecretUnseal

	// Start with initialization

	if vaultInit {
		init, err := checkInit(vaultFirstPod)
		if err != nil {
			log.Debugf("Starting bootstrap")
			log.Errorf(err.Error())
			os.Exit(1)
		}
		if !init {
			rootToken, unsealKeys, err = operatorInit(vaultFirstPod)
			if err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
			// If flag for creating k8s secrets is set
			if vaultK8sSecret {
				// Check if vault secret root exists
				_, err = getValuesFromK8sSecret(clientsetK8s, pVaultSecretRoot)
				if err != nil {
					// if it fails because secret is not found, create the secret
					if errors.IsNotFound(err) {
						if errI := createK8sSecret(clientsetK8s, &vaultSecretRoot, rootToken); errI != nil {
							log.Error(errI.Error())
							os.Exit(1)
						}
					} else {
						log.Error(err.Error())
						os.Exit(1)
					}
				}
				// Check if vault secret unseal exists
				_, err = getValuesFromK8sSecret(clientsetK8s, pVaultSecretRoot)
				if err != nil {
					// if it fails because secret is not found, create the secret
					if errors.IsNotFound(err) {
						unsealKeysString := strings.Join(*unsealKeys, ";")
						if errI := createK8sSecret(clientsetK8s, &vaultSecretUnseal, &unsealKeysString); errI != nil {
							log.Error(errI.Error())
							os.Exit(1)
						}
					} else {
						log.Error(err.Error())
						os.Exit(1)
					}
				}
			} else {
				logTokens(rootToken, unsealKeys)
			}
		} else {
			log.Info("Vault already initialized")
		}
	}

	// Check if unseal keys in memory and if not load them
	if unsealKeys == nil {
		unsealKeysString, err := getValuesFromK8sSecret(clientsetK8s, pVaultSecretUnseal)
		if err != nil {
			panic("Cannot load Unseal Keys")
		}
		npUnsealKeys := strings.Split(*unsealKeysString, ";")
		unsealKeys = &npUnsealKeys
		log.Debug("Unseal Keys loaded successfully")
	}

	if vaultUnseal && !vaultAutounseal {
		unsealed := unsealMember(vaultFirstPod, *unsealKeys)
		if unsealed {
			log.Debugf("Waiting 15 seconds after unsealing first member...")
			time.Sleep(15 * time.Second)
		}
		for _, vaultPod := range vaultPods[1:] {
			unsealMember(vaultPod, *unsealKeys)
		}
	}

	if vaultK8sAuth {
		up := checkVaultUp(clientLB)
		if !up {
			panic("K8s authentication: Vault not ready. Cannot proceed")
		}

		// set root token
		clientLB.SetToken(*rootToken)
		k8sAuth, err := checkK8sAuth(clientLB)
		if err != nil {
			log.Errorf(err.Error())
			os.Exit(1)
		}
		if k8sAuth {
			log.Info("K8s authentication: Already enabled")
			return
		}
		if err := configureK8sAuth(clientLB, clientsetK8s); err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
	}
}
