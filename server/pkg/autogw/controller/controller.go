package controller

import (
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/yz271544/edge-auto-gw/server/common/informers"
	"github.com/yz271544/edge-auto-gw/server/pkg/autogw/config"
)

const (
	labelEdgeMeshServiceProxyName = "service.edgemesh.kubeedge.io/service-proxy-name"
	labelNoProxyEdgeMesh          = "noproxy"

	LabelEdgemeshGatewayConfig = "kubeedge.io/edgemesh-gateway"
	LabelEdgemeshGatewayPort   = "kubeedge.io/edgemesh-gateway-port"
)

var (
	APIConn *AutoGatewayController
	once    sync.Once
)

type AutoGatewayController struct {
	sync.RWMutex
	atInformer      cache.SharedIndexInformer
	atEventHandlers map[string]cache.ResourceEventHandlerFuncs // key: gateway event handler name
}

func Init(ifm *informers.Manager, cfg *config.EdgeAutoGwConfig) {
	once.Do(func() {

		configSyncPeriod := metav1.Duration{Duration: 15 * time.Minute}

		noProxyName, err := labels.NewRequirement(labelNoProxyEdgeMesh, selection.DoesNotExist, nil)
		if err != nil {
			klog.Errorf("set selector label %s for request failed: %v", labelNoProxyEdgeMesh, err)
		}

		noEdgeMeshProxyName, err := labels.NewRequirement(labelEdgeMeshServiceProxyName, selection.DoesNotExist, nil)
		if err != nil {
			klog.Errorf("set selector label %s for request failed: %v", labelEdgeMeshServiceProxyName, err)
		}

		hasGateway, err := labels.NewRequirement(LabelEdgemeshGatewayConfig, selection.Exists, nil)
		if err != nil {
			klog.Errorf("set selector label %s for request failed: %v", LabelEdgemeshGatewayConfig, err)
		}

		labelSelector := labels.NewSelector()
		labelSelector = labelSelector.Add(*noProxyName, *noEdgeMeshProxyName, *hasGateway)

		client := ifm.GetKubeClient()

		informerFactory := k8sinformers.NewSharedInformerFactoryWithOptions(client, configSyncPeriod.Duration,
			k8sinformers.WithTweakListOptions(func(options *metav1.ListOptions) {
				options.LabelSelector = labelSelector.String()
			}))

		APIConn = &AutoGatewayController{
			atInformer:      informerFactory.Core().V1().Services().Informer(),
			atEventHandlers: make(map[string]cache.ResourceEventHandlerFuncs),
		}
		ifm.RegisterInformer(APIConn.atInformer)
		ifm.RegisterSyncedFunc(APIConn.onCacheSynced)
	})
}

func (c *AutoGatewayController) onCacheSynced() {

	for name, funcs := range c.atEventHandlers {
		klog.V(4).Infof("enable edge-auto-gw event handler funcs: %s", name)
		c.atInformer.AddEventHandler(funcs)
	}

	// set informers event handler
	// c.gwInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
	// 	AddFunc: c.gwAdd, UpdateFunc: c.gwUpdate, DeleteFunc: c.gwDelete})
}

func (c *AutoGatewayController) SetAutoGatewayEventHandlers(name string, handlerFuncs cache.ResourceEventHandlerFuncs) {
	c.Lock()
	if _, exist := c.atEventHandlers[name]; exist {
		klog.Warningf("edge-auto-gw event handler %s already exists, it will be overwritten!", name)
	}
	c.atEventHandlers[name] = handlerFuncs
	c.Unlock()
}
