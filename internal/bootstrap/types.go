package bootstrap

import vault "github.com/hashicorp/vault/api"

type VaultPod struct {
	Name   string
	Fqdn   string
	Client *vault.Client
}
