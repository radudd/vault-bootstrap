package bootstrap

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	clientFirstMember := vaultPods[0].client
	preflight(vaultPods)

	time.Sleep(5 * time.Second)

	var rootToken *string
	var unsealKeys *[]string

	// Start with initialization

	if vaultInit {
		init, err := checkInit(clientFirstMember)
		if err != nil {
			log.Debugf("Starting bootstrap")
			log.Errorf(err.Error())
			os.Exit(1)
		}
		if !init {
			rootToken, unsealKeys, err = operatorInit(clientFirstMember)
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

	// Check if root token and unseal keys in memory
	// If not, load them from K8s secret
	if rootToken == nil || unsealKeys == nil {
		rootToken, unsealKeys, err = getValuesFromK8sSecret(clientsetK8s)
		log.Debug("Unseal Keys and Root Token loaded successfully")
	}

	if vaultUnseal {
		unsealed := unsealMember(clientFirstMember, *unsealKeys)
		if unsealed {
			log.Debugf("Waiting 15 seconds after unsealing first member...")
			time.Sleep(15 * time.Second)
		}
		for _, vaultPod := range vaultPods[1:] {
			unsealMember(vaultPod.client, *unsealKeys)
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
