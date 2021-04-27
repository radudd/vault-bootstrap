## 0.1 (November 1st, 2020)
* Initial Version
* Can perform the following actions: 
** Vault initialize
** Save Root Token and Unseal Keys to K8s secret or print them to STDOUT
** Unseal all members
** Enable K8s auth
* Checks if Vault and Storage Backend components are up before starting any action
* Possibility to run as a job in Kubernetes

## 0.2 (January 15th, 2021)
* Change preflight checks by verifying if Vault is up using Vault API instead of K8s API
* No need to check Storage Backend components in preflight
* Possibility to load unseal keys and token from Kubernetes secret if Vault is not initialized with this job and they are not available in memory
* Possibility to run as a CronJob with Unseal only
* Added more debug messages
* When HA: Working with RAFT storage, as well as Consul
* When HA: Initialize a specific Vault pod instead of using the load-balancer (when using RAFT, same pod needs to be both initialized and initially unsealed)
* Bump Go version to 1.15.2

## 0.3 (April 27th, 2021)
* Separate secrets with configurable names for Vault root token and Vault unseal keys
* Addded a different mode of operation: `init-container`. This mode should be used to run this tool as an init container. This init container will spawn up a new `vault-bootstrap` job that can perform unsealing.