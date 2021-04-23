package bootstrap

import vault "github.com/hashicorp/vault/api"

type vaultPod struct {
	name   string
	fqdn   string
	client *vault.Client
}
