package constants

// Resources
const (
	// Common
	EdgeAutoGwNamespace      = "kubeedge"
	EdgeAutoGwConfigFileName = "edge-auto-gw.yaml"

	SecretNamespace = EdgeAutoGwNamespace

	SelectorForEdgeMeshGatewayKey   = "kubeedge"
	SelectorForEdgeMeshGatewayValue = "edgemesh-gateway"

	// env
	MY_NODE_NAME = "MY_NODE_NAME"

	DefaultConfigDir = "/etc/kubeedge/config/"
)
