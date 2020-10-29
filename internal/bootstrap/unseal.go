package bootstrap

import (
	"fmt"

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
	var sealed *vault.SealStatusResponse

	// Loop through the keys and unseal
	for j := 0; j < vaultKeyThreshold; j++ {
		sealed, err = client.Sys().Unseal((*unsealKeys)[j])
		if err != nil {
			return err
		}
	}
	if !sealed.Sealed {
		log.Info("Vault was successfully unsealed using Shamir keys: ", client.Address())
		return nil
	}
	return fmt.Errorf("Failed unsealing with Shamir keys")
}
