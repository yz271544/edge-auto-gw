package options

import (
	"fmt"
	"path"

	"github.com/kubeedge/kubeedge/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/yz271544/edge-auto-gw/server/cmd/edge-auto-gw/app/config"
	autoConstants "github.com/yz271544/edge-auto-gw/server/common/constants"
)

type EdgeAutoGwOptions struct {
	ConfigFile string
}

func NewEdgeAutoGwOptions() *EdgeAutoGwOptions {
	return &EdgeAutoGwOptions{
		ConfigFile: path.Join(autoConstants.DefaultConfigDir, autoConstants.EdgeAutoGwConfigFileName),
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

// Config generates *config.EdgeAutoGwConfig
func (o *EdgeAutoGwOptions) Config() (*config.EdgeAutoGwConfig, error) {
	cfg := config.NewEdgeAutoGwConfig()
	if err := cfg.Parse(o.ConfigFile); err != nil {
		return nil, err
	}
	return cfg, nil
}
