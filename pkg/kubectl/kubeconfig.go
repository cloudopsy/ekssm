package kubectl

import (
	"fmt"

	"github.com/cloudopsy/ekssm/internal/logging"
)

func GenerateKubeconfig(clusterName, endpoint string) string {
	logging.Debugf("Generating kubeconfig for cluster %s with endpoint %s", clusterName, endpoint)

	return fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    server: %s
    insecure-skip-tls-verify: true
  name: %s
contexts:
- context:
    cluster: %s
    user: aws
  name: %s
current-context: %s
kind: Config
preferences: {}
users:
- name: aws
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
      args:
        - eks
        - get-token
        - --cluster-name
        - %s
`, endpoint, clusterName, clusterName, clusterName, clusterName, clusterName)
}
