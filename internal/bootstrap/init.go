package bootstrap

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
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
	DefaultVaultSecretRoot     = "vault-root-token"
	DefaultVaultSecretUnseal   = "vault-unseal-keys"
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
	err                 error
	ok                  bool

	vaultServiceAccount string
	vaultSecretRoot     string
	vaultSecretUnseal   string
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

	if extrVaultSecretRoot, ok := os.LookupEnv("VAULT_SECRET_ROOT"); !ok {
		log.Warn("VAULT_SECRET_ROOT not set. Defaulting to ", DefaultVaultSecretRoot)
		vaultSecretRoot = DefaultVaultSecretRoot
	} else {
		vaultSecretRoot = extrVaultSecretRoot
	}
	if extrVaultSecretUnseal, ok := os.LookupEnv("VAULT_SECRET_UNSEAL"); !ok {
		log.Warn("VAULT_SECRET_UNSEAL not set. Defaulting to ", DefaultVaultSecretUnseal)
		vaultSecretUnseal = DefaultVaultSecretUnseal
	} else {
		vaultSecretUnseal = extrVaultSecretUnseal
	}

}
