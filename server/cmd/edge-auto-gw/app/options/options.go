package options

import (
	"fmt"
	"path"

	"k8s.io/apimachinery/pkg/util/validation/field"
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/kubeedge/edgemesh/agent/cmd/edgemesh-agent/app/config"
	meshConstants "github.com/kubeedge/edgemesh/common/constants"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/util/validation"
)

type EdgeAutoGwOptions struct {
	ConfigFile string
}

func NewEdgeAutoGwOptions() *EdgeAutoGwOptions {
	return &EdgeAutoGwOptions{
		ConfigFile: path.Join(constants.DefaultConfigDir, meshConstants.EdgeMeshAgentConfigFileName),
	}
}

func (o *EdgeAutoGwOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("global")
	fs.StringVar(&o.ConfigFile, "config-file", o.ConfigFile, "The path to the configuration file. Flags override values in this file.")
	return
}

func (o *EdgeAutoGwOptions) Validate() []error {
	var errs []error
	if !validation.FileIsExist(o.ConfigFile) {
		errs = append(errs, field.Required(field.NewPath("config-file"),
			fmt.Sprintf("config file %v not exist", o.ConfigFile)))
	}
	return errs
}

// Config generates *config.EdgeMeshAgentConfig
func (o *EdgeAutoGwOptions) Config() (*config.EdgeMeshAgentConfig, error) {
	cfg := config.NewEdgeMeshAgentConfig()
	if err := cfg.Parse(o.ConfigFile); err != nil {
		return nil, err
	}
	return cfg, nil
}
