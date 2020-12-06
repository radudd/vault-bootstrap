## 0.1 (November 1st, 2020)
* Initial Version
* May the following actions: 
** Vault initialize
** Save Root Token and Unseal Keys to K8s secret or print them to STDOUT
** Unseal all members
** Enable K8s auth
* Checks if Vault and Storage Backend components are up before starting any action
* Possibility to run as a job in Kubernetes

## 0.2 (December 7th, 2020)
* Change preflight checks by verifying if Vault is up using Vault API instead of K8s API
* No need to check Storage Backend components in preflight
* Possibility to load unseal keys and token from Kubernetes secret if Vault is not initialized with this job and they are not available in memory
* Possibility to run as a CronJob with Unseal only
* Added more debug messages
* Bump Go version to 1.15.2