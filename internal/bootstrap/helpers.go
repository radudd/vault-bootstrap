package bootstrap

import (
	"strings"
	apiv1 "k8s.io/api/core/v1"
)

var vaultReadyStatusCodes = []int{200, 501, 503, 429, 472, 473}

func find(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func getPodName(p *apiv1.Pod) string {
	if p.ObjectMeta.Name != "" {
		return p.ObjectMeta.Name
	}
	return strings.TrimSuffix(p.ObjectMeta.GenerateName, "-")
}
