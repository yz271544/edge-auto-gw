package autogw

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"

	"github.com/yz271544/edge-auto-gw/server/common/informers"
	"github.com/yz271544/edge-auto-gw/server/common/modules"
	"github.com/yz271544/edge-auto-gw/server/pkg/autogw/config"
	"github.com/yz271544/edge-auto-gw/server/pkg/autogw/controller"
	"github.com/yz271544/edge-auto-gw/server/pkg/autogw/manager"
)

// EdgeAutoGw is a edge ingress gateway
type EdgeAutoGw struct {
	Config *config.EdgeAutoGwConfig
}

func newEdgeAutoGw(c *config.EdgeAutoGwConfig, ifm *informers.Manager) (eag *EdgeAutoGw, err error) {
	eag = &EdgeAutoGw{Config: c}
	if !c.Enable {
		return eag, nil
	}

	// new controller
	controller.Init(ifm, c)

	// // new gateway manager
	manager.NewAutoGwManager(c, ifm)

	return eag, nil
}

// Register register EdgeAutoGw to beehive modules
func Register(c *config.EdgeAutoGwConfig, ifm *informers.Manager) error {
	eag, err := newEdgeAutoGw(c, ifm)
	if err != nil {
		return fmt.Errorf("register module EdgeAutoGw error: %v", err)
	}
	core.Register(eag)
	return nil
}

// Name of EdgeAutoGw
func (eag *EdgeAutoGw) Name() string {
	return modules.EdgeAutoGwModuleName
}

// Group of EdgeAutoGw
func (eag *EdgeAutoGw) Group() string {
	return modules.EdgeAutoGwModuleName
}

// Enable indicates whether enable this module
func (eag *EdgeAutoGw) Enable() bool {
	return eag.Config.Enable
}

// Start EdgeAutoGw
func (eag *EdgeAutoGw) Start() {
	eag.Run()
}
