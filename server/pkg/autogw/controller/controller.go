package controller

import (
	"sync"

	istiolisters "istio.io/client-go/pkg/listers/networking/v1alpha3"
	k8slisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/yz271544/edge-auto-gw/server/common/informers"
	"github.com/yz271544/edge-auto-gw/server/pkg/autogw/config"
)

var (
	APIConn *GatewayController
	once    sync.Once
)

type GatewayController struct {
	secretLister k8slisters.SecretLister
	vsLister     istiolisters.VirtualServiceLister
	gwInformer   cache.SharedIndexInformer
	gwManager    *Manager
}

func Init(ifm *informers.Manager, cfg *config.EdgeAutoGwConfig) {
	once.Do(func() {
		APIConn = &GatewayController{
			secretLister: ifm.GetKubeFactory().Core().V1().Secrets().Lister(),
			vsLister:     ifm.GetIstioFactory().Networking().V1alpha3().VirtualServices().Lister(),
			gwInformer:   ifm.GetIstioFactory().Networking().V1alpha3().Gateways().Informer(),
			gwManager:    NewGatewayManager(cfg),
		}
		ifm.RegisterInformer(APIConn.gwInformer)
		ifm.RegisterSyncedFunc(APIConn.onCacheSynced)
	})
}

func (c *GatewayController) onCacheSynced() {
	// set informers event handler
	// c.gwInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
	// 	AddFunc: c.gwAdd, UpdateFunc: c.gwUpdate, DeleteFunc: c.gwDelete})
}

type Manager struct {
}

func NewGatewayManager(c *config.EdgeGatewayConfig) *Manager {
	mgr := &Manager{}

	return mgr
}
