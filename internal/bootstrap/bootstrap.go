package bootstrap

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	DefaultVaultAddr             = "https://vault:8200"
	DefaultVaultClusterMembers   = "vault"
	DefaultStorageClusterMembers = ""
	DefaultVaultKeyShares        = 1
	DefaultVaultKeyThreshold     = 1
	DefaultVaultInit             = true
	DefaultVaultK8sSecret        = true
	DefaultVaultUnseal           = true
	DefaultVaultK8sAuth          = true

	VaultSecret         = "vault"
	VaultServiceAccount = "vault"
)

var (
	vaultAddr           string
	vaultClusterSize    int
	vaultClusterMembers string
	storageClusterMembers string
	vaultKeyShares      int
	vaultKeyThreshold   int
	vaultInit           bool
	vaultK8sSecret      bool
	vaultUnseal         bool
	vaultK8sAuth        bool
	err                 error
	ok                  bool
)

func init() {

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
	storageClusterMembers, ok = os.LookupEnv("VAULT_STORAGE_CLUSTER_MEMBERS")
	if !ok {
		log.Warn("VAULT_STORAGE_CLUSTER_MEMBERS not set. Defaulting to ", DefaultStorageClusterMembers)
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
		log.Warn("VAULT_ENABLE_K8SSSECRET not set. Defaulting to ", DefaultVaultK8sSecret)
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
}

// Run Vault bootstrap
func Run() {
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

	// Get current namespace
	namespaceBs, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Error("Cannot extract namespace" + err.Error())
		os.Exit(1)
	}
	namespace := string(namespaceBs)

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

	preflight(clientsetK8s, namespace)
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
				_, _, err = getValuesFromK8sSecret(clientsetK8s, namespace)
				if err != nil {
					// if it fails because secret is not found, create the secret
					if errors.IsNotFound(err) {
						if errI := createK8sSecret(rootToken, unsealKeys, clientsetK8s, namespace); errI != nil {
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
			rootToken, unsealKeys, err = getValuesFromK8sSecret(clientsetK8s, namespace)
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
		time.Sleep(10 * time.Second)
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
		if err := configureK8sAuth(client, clientsetK8s, namespace); err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
	}
}
