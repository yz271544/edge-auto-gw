package app

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/pkg/util"
	"github.com/kubeedge/kubeedge/pkg/util/flag"
	"github.com/kubeedge/kubeedge/pkg/version"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"

	"github.com/yz271544/edge-auto-gw/common/informers"
	"github.com/yz271544/edge-auto-gw/pkg/autogw"
	"github.com/yz271544/edge-auto-gw/server/cmd/edge-auto-gw/app/config"
	"github.com/yz271544/edge-auto-gw/server/cmd/edge-auto-gw/app/config/validation"
	"github.com/yz271544/edge-auto-gw/server/cmd/edge-auto-gw/app/options"
)

func NewEdgeAutoGwServerCommand() *cobra.Command {
	opts := options.NewEdgeAutoGwOptions()
	cmd := &cobra.Command{
		Use:  "edge-auto-gw",
		Long: `edge-auto-gw is a part of EdgeAutoGw, and provides auto create gateway virtualservice destinationrule for service.`,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			flag.PrintFlags(cmd.Flags())

			if errs := opts.Validate(); len(errs) > 0 {
				klog.Exit(util.SpliceErrors(errs))
			}

			serverCfg, err := opts.Config()
			if err != nil {
				klog.Exit(err)
			}

			if errs := validation.ValidateEdgeAutoGwConfiguration(serverCfg); len(errs) > 0 {
				klog.Exit(util.SpliceErrors(errs.ToAggregate().Errors()))
			}

			klog.Infof("Version: %+v", version.Get())
			if err = Run(serverCfg); err != nil {
				klog.Exit("run edgemesh-server failed: %v", err)
			}
		},
	}
	fs := cmd.Flags()
	namedFs := opts.Flags()
	verflag.AddFlags(namedFs.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFs.FlagSet("global"), cmd.Name())
	for _, f := range namedFs.FlagSets {
		fs.AddFlagSet(f)
	}

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFs, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFs, cols)
	})
	return cmd
}

//Run runs EdgeAutoGw Server
func Run(cfg *config.EdgeAutoGwConfig) error {
	trace := 1

	klog.Infof("[%d] New informers manager", trace)
	ifm, err := informers.NewManager(cfg.KubeAPIConfig)
	if err != nil {
		return err
	}
	trace++

	klog.Infof("[%d] Register beehive modules", trace)
	if errs := registerModules(cfg, ifm); len(errs) > 0 {
		return fmt.Errorf(util.SpliceErrors(errs))
	}
	trace++

	klog.Infof("[%d] Start informers manager", trace)
	ifm.Start(wait.NeverStop)
	trace++

	klog.Infof("[%d] Start all modules", trace)
	core.Run()

	klog.Infof("edge-auto-gw exited")
	return nil
}

// registerModules register all the modules started in edgemesh-server
func registerModules(c *config.EdgeAutoGwConfig, ifm *informers.Manager) []error {
	var errs []error
	if err := autogw.Register(c.Modules.EdgeAutoConfig, ifm); err != nil {
		errs = append(errs, err)
	}
	return errs
}
