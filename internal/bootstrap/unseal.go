package bootstrap

import (
	"fmt"
	"strconv"

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

// Unseal Vault using Shamir keys
func shamirUnseal(client *vault.Client, unsealKeys *[]string) error {

	var err error
	var sealStatus *vault.SealStatusResponse

	// Loop through the keys and unseal
	for j := 0; j < vaultKeyThreshold; j++ {
		sealStatus, err = client.Sys().Unseal((*unsealKeys)[j])
		if err != nil {
			return err
		}
		log.Debugf("%s: Unseal progress %s/%s", client.Address(), strconv.Itoa(sealStatus.Progress), strconv.Itoa(vaultKeyThreshold))
	}
	if !sealStatus.Sealed {
		log.Info("Vault was successfully unsealed using Shamir keys: ", client.Address())
		return nil
	}
	respSealStatus, err := client.Sys().SealStatus()
	if err != nil {
		return err
	}
	respStatus := strconv.Itoa(respSealStatus.Progress)

	return fmt.Errorf("Failed unsealing with Shamir keys: ", respStatus)
}
