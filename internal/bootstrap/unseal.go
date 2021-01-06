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

func unsealMember(pod vaultPod, unsealKeys []string) bool {
	unsealed, err := checkUnseal(pod.client)
	if err != nil {
		log.Errorf(err.Error())
		return false
	}
	if unsealed {
		log.Info("%s: Vault already unsealed", pod.name)
		return false
	} else {
		shamirUnseal(pod, unsealKeys)
		return true
	}
}

// Unseal Vault using Shamir keys
func shamirUnseal(pod vaultPod, unsealKeys []string) {
	var err error
	var sealStatus *vault.SealStatusResponse

	out:
	for {
		log.Infof("%s: Starting unsealing", pod.name)
		// Loop through the keys and unseal
		for j := 1; j <= vaultKeyThreshold; j++ {
			time.Sleep(2 * time.Second)
			sealStatus, err = pod.client.Sys().Unseal(unsealKeys[j])
			if err != nil {
				log.Infof("%s: %s", pod.name, err.Error())
				continue out
			}
			log.Infof("%s: Unseal progress %s/%s", pod.name, strconv.Itoa(sealStatus.Progress), strconv.Itoa(vaultKeyThreshold))
		}
		break
	}
	if !sealStatus.Sealed {
		log.Infof("%s: Vault was successfully unsealed using Shamir keys", pod.name)
	}
}
