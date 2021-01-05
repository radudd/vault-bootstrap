package bootstrap

import (
	"strconv"
	"time"

	vault "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
)

func checkUnseal(client *vault.Client) (bool, error) {
	sealed, err := client.Sys().SealStatus()
	if err != nil {
		return false, err
	}
	if sealed.Sealed {
		return false, nil
	}
	return true, nil
}

func prepareUnseal(members []string, unsealKeys []string) {
	c := make(chan string, len(members))
	for _, member := range members {
		clientConfig := &vault.Config{
			Address: member,
		}
		insecureTLS := &vault.TLSConfig{
			Insecure: true,
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
			go shamirUnseal(clientNode, unsealKeys, c)
		}
		for range members {
			log.Infof("%s is Unsealed", <-c)
		}
	}
}

// Unseal Vault using Shamir keys
func shamirUnseal(client *vault.Client, unsealKeys []string, c chan string) {
	var err error
	var sealStatus *vault.SealStatusResponse

	// Loop through the keys and unseal
	for j := 1; j <= vaultKeyThreshold; j++ {
		sealStatus, err = client.Sys().Unseal(unsealKeys[j])
		if err != nil {
			panic("Cannot unseal " + client.Address() + ". Exiting..")
		}
		log.Debugf("%s: Unseal progress %s/%s", client.Address(), strconv.Itoa(sealStatus.Progress), strconv.Itoa(vaultKeyThreshold))
		time.Sleep(2 * time.Second)

	}
	if !sealStatus.Sealed {
		log.Info("Vault was successfully unsealed using Shamir keys: ", client.Address())
	}
	c <- client.Address()
}
