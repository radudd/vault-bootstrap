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

func unsealMember(client *vault.Client, unsealKeys []string) bool {
	unsealed, err := checkUnseal(client)
	if err != nil {
		log.Errorf(err.Error())
		return false
	}
	if unsealed {
		log.Info("%s: Vault already unsealed", client.Address())
		return false
	} else {
		shamirUnseal(client, unsealKeys)
		return true
	}
}

// Unseal Vault using Shamir keys
func shamirUnseal(client *vault.Client, unsealKeys []string) {
	var err error
	var sealStatus *vault.SealStatusResponse

	out:
	for {
		log.Infof("%s: Starting unsealing", client.Address())
		// Loop through the keys and unseal
		for j := 1; j <= vaultKeyThreshold; j++ {
			time.Sleep(2 * time.Second)
			sealStatus, err = client.Sys().Unseal(unsealKeys[j])
			if err != nil {
				log.Infof("%s: %s", client.Address(), err.Error())
				continue out
			}
			log.Debugf("%s: Unseal progress %s/%s", client.Address(), strconv.Itoa(sealStatus.Progress), strconv.Itoa(vaultKeyThreshold))
		}
		break
	}
	if !sealStatus.Sealed {
		log.Infof("%s: Vault was successfully unsealed using Shamir keys", client.Address())
	}
}
