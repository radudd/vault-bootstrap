package bootstrap

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	DefaultVaultAddr         = "https://vault:8200"
	DefaultVaultKeyShares    = 1
	DefaultVaultKeyThreshold = 1
	DefaultVaultInitialized  = false
	DefaultK8sRole           = "example"
	DefaultK8sRoleTokenTTL   = "2h"

	VaultServiceAccount = "vault"
)

var (
	vaultAddr         string
	vaultKeyShares    int
	vaultKeyThreshold int
)

func init() {
	if vaultAddr, ok := os.LookupEnv("VAULT_ADDR"); !ok {
		log.Warn("VAULT_ADDR not set. Defaulting to ", DefaultVaultAddr)
		vaultAddr = DefaultVaultAddr
		os.Setenv("VAULT_ADDR", vaultAddr)
	}
	if extrVaultKeyShares, ok := os.LookupEnv("VAULT_KEY_SHARES"); !ok {
		log.Warn("VAULT_KEY_SHARES not set. Defaulting to ", DefaultVaultKeyShares)
		vaultKeyShares = DefaultVaultKeyShares
		os.Setenv("VAULT_KEY_SHARES", strconv.Itoa(vaultKeyShares))
	} else {
		var err error
		vaultKeyShares, err = strconv.Atoi(extrVaultKeyShares)
		if err != nil {
			log.Error("Invalid value for VAULT_KEY_SHARES" + err.Error())
		}
	}
	if extrVaultKeyThreshold, ok := os.LookupEnv("VAULT_KEY_THRESHOLD"); !ok {
		log.Warn("VAULT_KEY_THRESHOLD not set. Defaulting to ", DefaultVaultKeyThreshold)
		vaultKeyThreshold = DefaultVaultKeyThreshold
		os.Setenv("VAULT_KEY_THRESHOLD", strconv.Itoa(vaultKeyThreshold))
	} else {
		var err error
		vaultKeyThreshold, err = strconv.Atoi(extrVaultKeyThreshold)
		if err != nil {
			log.Error("Invalid value for VAULT_KEY_THRESHOLD" + err.Error())
		}
	}
}

func operatorInit(client *vault.Client) (*string, *[]string, error) {

	initReq := &vault.InitRequest{
		SecretShares:    vaultKeyShares,
		SecretThreshold: vaultKeyThreshold,
	}
	init, err := client.Sys().InitStatus()
	if err != nil {
		return nil, nil, err
	}
	if init {
		return nil, nil, fmt.Errorf("Vault already initialiazed")
	}
	initResp, err := client.Sys().Init(initReq)
	if err != nil {
		return nil, nil, err
	}

	return &initResp.RootToken, &initResp.Keys, nil

	/*
		// Check if Vault initialized
		endpoint := vaultAddr + InitEndpoint
		resp, err := http.Get(endpoint)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}

		if err := json.Unmarshal(body, v.vaultStatus); err != nil {
			return nil, nil, err
		}
		if !v.vaultStatus.initialized {
			log.Warn("Vault not initialized")
			initParams, err := json.Marshal(v.vaultInit)
			if err != nil {
				return nil, nil, err
			}
			resp, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewBuffer(initParams))
			if err != nil {
				return nil, nil, err
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			var vr vaultInitResponse
			if err := json.Unmarshal(body, &vr); err != nil {
				return nil, nil, err
			}
			return &vr.rootToken, &vr.keys, nil

		}
		log.Info("Vault already initialized")
		// update vaultInitialize
		return nil, nil, nil
	*/
}

func createK8sSecret(rootToken string, unsealKeys []string) error {
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	secretClient := clientset.CoreV1().Secrets(apiv1.NamespaceDefault)
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vault",
		},
		Type: apiv1.SecretTypeOpaque,
		StringData: map[string]string{
			"rootToken":  rootToken,
			"unsealKeys": strings.Join(unsealKeys, ";"),
		},
	}

	log.Info("Creating K8s secret ...")
	result, err := secretClient.Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Info("Created K8s secret ", result.GetObjectMeta().GetName())
	return nil
}

// Check if auto-unseal is configured, if not returns an error
func checkUnsealed(client *vault.Client) error {

	sealed, err := client.Sys().SealStatus()

	if err != nil {
		return err
	}

	if sealed.Sealed == true {
		return fmt.Errorf("Vault is Sealed. Check your Auto-unseal mechanism")
	}
	return nil

}

func configureK8sAuth(client *vault.Client, token string) error {
	/*
			vault auth enable kubernetes
		    vault write auth/kubernetes/config token_reviewer_jwt=$JWT kubernetes_host=$KUBERNETES_HOST kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
	*/

	// Set Token
	os.Setenv("VAULT_TOKEN", token)

	// Enable K8S authentication
	err := client.Sys().EnableAuthWithOptions("kubernetes/", &vault.EnableAuthOptions{
		Type: "kubernetes",
	})

	if err != nil {
		return err
	}

	// Fetch Vault token to connect to K8S API
	// THIS IS PRESENT ON VAULT CONTAINER, not on the JOB or sidecar container
	/*
		jwt, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
		if err != nil {
			return err
		}
	*/
	/*
		k8sConfig, err := clientcmd.BuildConfigFromFlags("", "")
		if err != nil {
			return err
		}
	*/

	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	clientsetK8s, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return err
	}

	vaultServiceAccount, err := clientsetK8s.CoreV1().ServiceAccounts(apiv1.NamespaceDefault).Get(context.TODO(), VaultServiceAccount, metav1.GetOptions{})
	if err != nil {
		return err
	}
	vaultSASecret := vaultServiceAccount.Secrets[0].Name
	vaultSAK8sSecret, err := clientsetK8s.CoreV1().Secrets(apiv1.NamespaceDefault).Get(context.TODO(), vaultSASecret, metav1.GetOptions{})
	if err != nil {
		return err
	}
	vaultJwt := vaultSAK8sSecret.Data["token"]

	// Fetch CA for connecting to K8S API
	cacert, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return err
	}

	// Get K8S API URL
	k8sHost, ok := os.LookupEnv("KUBERNETES_HOST")
	if !ok {
		return fmt.Errorf("Invalid Kubernetes API config")
	}

	// Prepare payload for configuring k8s authentication
	data := map[string]interface{}{
		"kubernetes_host":    k8sHost,
		"kubernetes_ca_cert": string(cacert),
		"token_reviewer_jwt": string(vaultJwt),
	}

	// Configure K8S authentication
	_, err = client.Logical().Write("auth/kubernetes/config", data)
	if err != nil {
		return err
	}
	return nil
}

func Run() {
	config := vault.DefaultConfig()
	client, err := vault.NewClient(config)
	if err != nil {
		return
	}
	// Skip TLS verification for initialization
	os.Setenv("VAULT_SKIP_VERIFY", "true")
	pRootToken, pUnsealKeys, err := operatorInit(client)
	if err != nil {
		log.Error(err.Error())
		return
	}
	rootToken := *pRootToken
	unsealKeys := *pUnsealKeys
	if err := createK8sSecret(rootToken, unsealKeys); err != nil {
		log.Error(err.Error())
		return
	}
	if err = configureK8sAuth(client, rootToken); err != nil {
		log.Error(err.Error())
		return
	}
}

/*
// Define Role Name as Env variable
func createK8sRole(client *vault.Client, token string) error {
	//vault write auth/kubernetes/role/<rolename> \
    //    bound_service_account_names=default \
    //    bound_service_account_namespaces='<ns>' \
    //    policies=<rolename>-policy \
	//ttl=2h

	// Extract Role name from env variable
	role, err := os.LookupEnv("K8S_ROLE")
	if err != nil {
		role = DefaultK8sRole
	}

	// Extract Service Account Name (of running App)
	boundSA := nil
	// Extract Namesapce (of running App)
	boundNS := nil
	// Extract TTL from Env var
	ttl, err := os.LookupEnv("K8S_ROLE_TOKEN_TTL")
	if err != nil {
		ttl = DefaultK8sRoleTokenTTL


	// Prepare config options
	k8sRolePath := "auth/kubernetes/role/" + role
	k8sPolicy := role + "-policy"


	// Prepare payload for creating k8s role
	data := map[string]interface{} {
		bound_service_account_names: boundSA,
		bound_service_account_namespaces: boundNS,
		token_ttl: ttl,
	}

	configK8s, err := client.Logical().Write(k8sPolicy,data)
	if err != nil {
		return err
	}

	return nil
}

func createPolicy(p policy) error {
	//vault policy write -tls-skip-verify policy-example policy/policy-example.hcl
	// use client Go to extract from policy configMap
}
*/
