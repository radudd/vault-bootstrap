package bootstrap

import (
	"context"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	DefaultVaultAddr           = "https://vault:8200"
	DefaultVaultClusterMembers = "https://vault:8200"
	DefaultVaultKeyShares      = 1
	DefaultVaultKeyThreshold   = 1
	DefaultVaultInit           = true
	DefaultVaultK8sSecret      = true
	DefaultVaultUnseal         = true
	DefaultVaultK8sAuth        = true
	DefaultVaultServiceAccount = "vault"

	VaultSecret = "vault"
)

var (
	namespace           string
	vaultAddr           string
	vaultClusterSize    int
	vaultClusterMembers string
	vaultKeyShares      int
	vaultKeyThreshold   int
	vaultInit           bool
	vaultK8sSecret      bool
	vaultUnseal         bool
	vaultK8sAuth        bool
	vaultServiceAccount string
	err                 error
	ok                  bool
)

func init() {

	// Extract namespace: https://github.com/kubernetes/kubernetes/pull/63707
	// Try to extract via Downwards API
	if namespace, ok = os.LookupEnv("NAMESPACE"); !ok {
		// Fall back to namespace of the service account
		if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
			namespace = strings.TrimSpace(string(data))
		}
	}

	if vaultAddr, ok := os.LookupEnv("VAULT_ADDR"); !ok {
		log.Warn("VAULT_ADDR not set. Defaulting to ", DefaultVaultAddr)
		vaultAddr = DefaultVaultAddr
		os.Setenv("VAULT_ADDR", vaultAddr)
	}
	vaultClusterMembers, ok = os.LookupEnv("VAULT_CLUSTER_MEMBERS")
	if !ok {
		log.Warn("VAULT_CLUSTER_MEMBERS not set. Defaulting to ", DefaultVaultClusterMembers)
		vaultClusterMembers = DefaultVaultClusterMembers
	}
	if extrVaultKeyShares, ok := os.LookupEnv("VAULT_KEY_SHARES"); !ok {
		log.Warn("VAULT_KEY_SHARES not set. Defaulting to ", DefaultVaultKeyShares)
		vaultKeyShares = DefaultVaultKeyShares
	} else {
		vaultKeyShares, err = strconv.Atoi(extrVaultKeyShares)
		if err != nil {
			log.Error("Invalid value for VAULT_KEY_SHARES" + err.Error())
		}
	}
	if extrVaultKeyThreshold, ok := os.LookupEnv("VAULT_KEY_THRESHOLD"); !ok {
		log.Warn("VAULT_KEY_THRESHOLD not set. Defaulting to ", DefaultVaultKeyThreshold)
		vaultKeyThreshold = DefaultVaultKeyThreshold
	} else {
		vaultKeyThreshold, err = strconv.Atoi(extrVaultKeyThreshold)
		if err != nil {
			log.Error("Invalid value for VAULT_KEY_THRESHOLD" + err.Error())
		}
	}
	if extrVaultInit, ok := os.LookupEnv("VAULT_ENABLE_INIT"); !ok {
		log.Warn("VAULT_ENABLE_INIT not set. Defaulting to ", DefaultVaultInit)
		vaultInit = DefaultVaultInit
	} else {
		vaultInit, err = strconv.ParseBool(extrVaultInit)
		if err != nil {
			log.Error("Invalid value for VAULT_ENABLE_INIT" + err.Error())
		}
	}
	if extrVaultK8sSecret, ok := os.LookupEnv("VAULT_ENABLE_K8SSECRET"); !ok {
		log.Warn("VAULT_ENABLE_K8SSECRET not set. Defaulting to ", DefaultVaultK8sSecret)
		vaultK8sSecret = DefaultVaultK8sSecret
	} else {
		vaultK8sSecret, err = strconv.ParseBool(extrVaultK8sSecret)
		if err != nil {
			log.Error("Invalid value for VAULT_ENABLE_K8SSECRET" + err.Error())
		}
	}
	if extrVaultUnseal, ok := os.LookupEnv("VAULT_ENABLE_UNSEAL"); !ok {
		log.Warn("VAULT_ENABLE_UNSEAL not set. Defaulting to ", DefaultVaultUnseal)
		vaultUnseal = DefaultVaultUnseal
	} else {
		vaultUnseal, err = strconv.ParseBool(extrVaultUnseal)
		if err != nil {
			log.Error("Invalid value for VAULT_ENABLE_UNSEAL" + err.Error())
		}
	}
	if extrVaultK8sAuth, ok := os.LookupEnv("VAULT_ENABLE_K8SAUTH"); !ok {
		log.Warn("VAULT_ENABLE_K8SAUTH not set. Defaulting to ", DefaultVaultK8sAuth)
		vaultK8sAuth = DefaultVaultK8sAuth
	} else {
		vaultK8sAuth, err = strconv.ParseBool(extrVaultK8sAuth)
		if err != nil {
			log.Error("Invalid value for VAULT_ENABLE_K8SAUTH" + err.Error())
		}
	}
	if extrVaultServiceAccount, ok := os.LookupEnv("VAULT_SERVICE_ACCOUNT"); !ok {
		log.Warn("VAULT_SERVICE_ACCOUNT not set. Defaulting to ", DefaultVaultServiceAccount)
		vaultServiceAccount = DefaultVaultServiceAccount
	} else {
		vaultServiceAccount = extrVaultServiceAccount
	}

}

// Run Vault bootstrap
func Run() {

	// Create clientSet for k8s client-go
	//k8sConfig, err := rest.InClusterConfig()
	//if err != nil {
	//	log.Error(err.Error())
	//	os.Exit(1)
	//}

	k8sConfig, _ := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")

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

	// Vault client
	config := vault.DefaultConfig()
	// Skip TLS verification for initialization
	insecureTLS := &vault.TLSConfig{
		Insecure: true,
	}
	config.ConfigureTLS(insecureTLS)

	client, err := vault.NewClient(config)
	if err != nil {
		os.Exit(1)
	}

	preflight()

	time.Sleep(5 * time.Second)

	var rootToken *string
	var unsealKeys *[]string

	// Start with initialization

	if vaultInit {
		init, err := checkInit(client)
		if err != nil {
			log.Debugf("Starting bootstrap")
			log.Errorf(err.Error())
			os.Exit(1)
		}
		if !init {
			rootToken, unsealKeys, err = operatorInit(client)
			if err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
			// If flag for creating k8s secret is set
			if vaultK8sSecret {
				// Check if vault secret exists
				_, _, err = getValuesFromK8sSecret(clientsetK8s)
				if err != nil {
					// if it fails because secret is not found, create the secret
					if errors.IsNotFound(err) {
						if errI := createK8sSecret(rootToken, unsealKeys, clientsetK8s); errI != nil {
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

	init, err := checkInit(client)
	if err != nil {
		log.Errorf(err.Error())
		os.Exit(1)
	}
	if !init {
		log.Errorf("Cannot proceed. Vault not initialized")
		os.Exit(1)
	}

	// Check if root token and unseal keys in memory
	// If not, load them from K8s secret
	if rootToken == nil || unsealKeys == nil {
		rootToken, unsealKeys, err = getValuesFromK8sSecret(clientsetK8s)
		log.Debug("Unseal Keys and Root Token loaded successfully")
	}

	if vaultUnseal {
		log.Info("Cluster Members: ", vaultClusterMembers)
		members := strings.Split(vaultClusterMembers, ",")
		for _, member := range members {
			clientConfig := &vault.Config{
				Address: member,
			}
			clientConfig.ConfigureTLS(insecureTLS)
			clientNode, err := vault.NewClient(clientConfig)
			unsealed, err := checkUnseal(clientNode)
			if err != nil {
				log.Errorf(err.Error())
			}
			if unsealed {
				log.Info("Vault already unsealed: ", clientConfig.Address)
			} else {
				if err := shamirUnseal(clientNode, unsealKeys); err != nil {
					log.Error(err.Error())
					os.Exit(1)
				}
			}
		}
	}

	if vaultK8sAuth {
		up := checkVaultUp(client)
		if !up {
			panic("Vault not ready. Cannot proceed with enabling K8s authentication")
		}

		// set root token
		client.SetToken(*rootToken)
		k8sAuth, err := checkK8sAuth(client)
		if err != nil {
			log.Errorf(err.Error())
			os.Exit(1)
		}
		if k8sAuth {
			log.Info("Vault Kubernetes authentication already enabled")
			return
		}
		if err := configureK8sAuth(client, clientsetK8s); err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
	}
}
