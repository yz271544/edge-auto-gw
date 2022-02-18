package config

import (
	"io/ioutil"
	"path"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	autogwconfig "github.com/yz271544/edge-auto-gw/cloud/pkg/gateway/config"
)

const (
	GroupName  = "edgeautogw.config.kubeedge.io"
	APIVersion = "v1alpha1"
	Kind       = "EdgeAutoGw"
)

// EdgeAutoGwConfig indicates the config of edgeAutoGw which get from edgeAutoGw config file
type EdgeAutoGwConfig struct {
	metav1.TypeMeta
	// CommonConfig indicates common config for all modules
	// +Required
	CommonConfig *CommonConfig `json:"commonConfig,omitempty"`
	// KubeAPIConfig indicates the kubernetes cluster info which edgeMeshAgent will connected
	// +Required
	KubeAPIConfig *v1alpha1.KubeAPIConfig `json:"kubeAPIConfig,omitempty"`
	// Modules indicates edgeMeshAgent modules config
	// +Required
	Modules *Modules `json:"modules,omitempty"`
}

// CommonConfig defines some common configuration items
type CommonConfig struct {
	// ConfigMapName indicates the configmap mounted by edgemesh-agent,
	// which contains all the configuration information of edgemesh-agent
	// default edgemesh-agent-cfg
	ConfigMapName string `json:"configMapName,omitempty"`
}

// Modules indicates the modules of edgeMeshAgent will be use
type Modules struct {
	// EdgeDNSConfig indicates edgedns module config
	EdgeAutoConfig *autogwconfig.EdgeAutoGwConfig `json:"edgeDNS,omitempty"`
}

// NewEdgeMeshAgentConfig returns a full EdgeMeshAgentConfig object
func NewEdgeAutoGwConfig() *EdgeAutoGwConfig {
	c := &EdgeAutoGwConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		CommonConfig: &CommonConfig{
			ConfigMapName: DefaultConfigMapName,
		},
		KubeAPIConfig: &v1alpha1.KubeAPIConfig{
			Master:      "",
			ContentType: runtime.ContentTypeJSON,
			QPS:         constants.DefaultKubeQPS,
			Burst:       constants.DefaultKubeBurst,
			KubeConfig:  "",
		},
		Modules: &Modules{
			EdgeAutoConfig: autogwconfig.NewEdgeAutoGwConfig(),
		},
	}

	return c
}

// Parse unmarshal config file into *EdgeAutoGwConfig
func (c *EdgeAutoGwConfig) Parse(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		klog.Errorf("Failed to read config file %s: %v", filename, err)
		return err
	}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		klog.Errorf("Failed to unmarshal config file %s: %v", filename, err)
		return err
	}
	return nil
}
