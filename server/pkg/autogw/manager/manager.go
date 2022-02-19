package manager

import (
	"context"
	"strconv"
	"strings"
	"sync"

	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/yz271544/edge-auto-gw/server/common/constants"
	"github.com/yz271544/edge-auto-gw/server/common/informers"
	"github.com/yz271544/edge-auto-gw/server/pkg/autogw/config"
	"github.com/yz271544/edge-auto-gw/server/pkg/autogw/controller"
)

const (
	tcpProtocol    = "TCP"
	httpProtocol   = "HTTP"
	minGatewayPort = 30000
	maxGatewayPort = 65535
)

var (
	Protocols = []string{tcpProtocol, httpProtocol}
)

// Manager is gateway manager
type AutoGwManager struct {
	lock sync.Mutex
	ifm  *informers.Manager
}

func NewAutoGwManager(c *config.EdgeAutoGwConfig, ifm *informers.Manager) *AutoGwManager {
	mgr := &AutoGwManager{
		ifm: ifm,
	}
	klog.V(4).Infof("start get ips which need listen...")
	// set edge-auto-gateway-manager event handler funcs
	controller.APIConn.SetAutoGatewayEventHandlers("edge-auto-gateway-manager", cache.ResourceEventHandlerFuncs{
		AddFunc: mgr.atAdd, UpdateFunc: mgr.atUpdate, DeleteFunc: mgr.atDelete})
	return mgr
}

func (mgr *AutoGwManager) atAdd(obj interface{}) {
	at, ok := obj.(*v1.Service)
	if !ok {
		klog.Errorf("invalid type %v", obj)
		return
	}
	mgr.addAtGateway(at)
}

func (mgr *AutoGwManager) atUpdate(oldObj, newObj interface{}) {
	at, ok := newObj.(*v1.Service)
	if !ok {
		klog.Errorf("invalid type %v", newObj)
		return
	}
	mgr.updateAtGateway(at)
}

func (mgr *AutoGwManager) atDelete(obj interface{}) {
	at, ok := obj.(*v1.Service)
	if !ok {
		klog.Errorf("invalid type %v", obj)
		return
	}
	mgr.deleteAtGateway(at)
}

// addGateway add a gateway server
func (mgr *AutoGwManager) addAtGateway(at *v1.Service) {

	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	var err error
	if at == nil {
		klog.Errorf("gateway is nil")
		return
	}

	atLables := at.GetLabels()

	isExposeGateway, gatewayProtocol, svcPort, gatewayPort := extractGatewayConfig(atLables)

	if isExposeGateway {
		ns := at.GetNamespace()
		nm := at.GetName()

		dr := GenerateDestinationRule(nm, ns)

		vs := GenerateVirtualService(nm, ns, svcPort)

		gw := GenerateGateway(nm, ns, gatewayProtocol, gatewayPort)

		clent := mgr.ifm.GetIstioClient().NetworkingV1alpha3()

		_, err = clent.DestinationRules(ns).Create(context.Background(), dr, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("create destination rule failed: %v", err)
			return
		}

		_, err = clent.VirtualServices(ns).Create(context.Background(), vs, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("create virtualservice rule failed: %v", err)
			return
		}
		_, err = clent.Gateways(ns).Create(context.Background(), gw, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("create gateway rule failed: %v", err)
			return
		}
	}

}

// updateGateway update a gateway server
func (mgr *AutoGwManager) updateAtGateway(at *v1.Service) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	var err error
	if at == nil {
		klog.Errorf("gateway is nil")
		return
	}

	atLables := at.GetLabels()

	isExposeGateway, gatewayProtocol, svcPort, gatewayPort := extractGatewayConfig(atLables)

	if isExposeGateway {
		ns := at.GetNamespace()
		nm := at.GetName()

		dr := GenerateDestinationRule(nm, ns)

		vs := GenerateVirtualService(nm, ns, svcPort)

		gw := GenerateGateway(nm, ns, gatewayProtocol, gatewayPort)

		clent := mgr.ifm.GetIstioClient().NetworkingV1alpha3()

		_, err = clent.DestinationRules(ns).Update(context.Background(), dr, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("update destination rule failed: %v", err)
			return
		}

		_, err = clent.VirtualServices(ns).Update(context.Background(), vs, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("update virtualservice rule failed: %v", err)
			return
		}
		_, err = clent.Gateways(ns).Update(context.Background(), gw, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("update gateway rule failed: %v", err)
			return
		}
	}
}

// deleteGateway delete a gateway server
func (mgr *AutoGwManager) deleteAtGateway(at *v1.Service) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	var err error
	if at == nil {
		klog.Errorf("gateway is nil")
		return
	}

	clent := mgr.ifm.GetIstioClient().NetworkingV1alpha3()

	ns := at.GetNamespace()
	nm := at.GetName()

	err = clent.DestinationRules(ns).Delete(context.Background(), nm, metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("delete destination rule failed: %v", err)
		return
	}

	err = clent.VirtualServices(ns).Delete(context.Background(), nm, metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("delete virtualservice rule failed: %v", err)
		return
	}

	err = clent.Gateways(ns).Delete(context.Background(), nm, metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("delete gateway rule failed: %v", err)
		return
	}
}

// extractGatewayConfig Extract the Gateway Protocol and Port
func extractGatewayConfig(atLables map[string]string) (isExposeGateway bool, gatewayProtocol string, svcPort uint32, gatewayPort uint32) {

	if _, ok := atLables[controller.LabelEdgemeshGatewayConfig]; !ok {
		klog.V(4).Infof("not have %s label in the service.", controller.LabelEdgemeshGatewayConfig)
		return false, "", 0, 0
	}

	if _, ok := atLables[controller.LabelEdgemeshGatewayPort]; !ok {
		klog.V(4).Infof("not have %s label in the service.", controller.LabelEdgemeshGatewayPort)
		return false, "", 0, 0
	}

	for _, protocol := range Protocols {
		if protocol == atLables[controller.LabelEdgemeshGatewayConfig] {
			gatewayProtocol = protocol
		}
	}

	edgemeshGatewayPort := atLables[controller.LabelEdgemeshGatewayPort]
	sets := strings.Split(edgemeshGatewayPort, "-")

	if len(sets) >= 2 {
		// extract Servcie port
		port, err := strconv.Atoi(sets[0])
		if err != nil {
			klog.V(4).Infof("svcPort %s not convert integer: %v", sets[0], err)
			return false, "", 0, 0
		}
		if port > 0 && port <= maxGatewayPort {
			svcPort = uint32(port)
		} else {
			klog.V(4).Infof("svcPort %d is invalid scope", port)
			return false, "", 0, 0
		}
		// extract Gateway expose port
		port, err = strconv.Atoi(sets[1])
		if err != nil {
			klog.V(4).Infof("gatewayPort %s not convert integer: %v", sets[1], err)
			return false, "", 0, 0
		}
		if port >= minGatewayPort && port <= maxGatewayPort {
			gatewayPort = uint32(port)
			return
		} else {
			klog.V(4).Infof("gatewayPort %d is invalid scope", port)
		}

	} else {
		klog.V(4).Infof("gateway-port %s is invalid format: SvcPort-GatewayPort", edgemeshGatewayPort)
		return false, "", 0, 0
	}

	return false, "", 0, 0
}

// GenerateDestinationRule generate DestinationRule
func GenerateDestinationRule(name, namespace string) (dr *istioapi.DestinationRule) {
	dr = &istioapi.DestinationRule{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DestinationRule",
			APIVersion: "networking.istio.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1alpha3.DestinationRule{
			Host: name,
			TrafficPolicy: &networkingv1alpha3.TrafficPolicy{
				LoadBalancer: &networkingv1alpha3.LoadBalancerSettings{
					LbPolicy: &networkingv1alpha3.LoadBalancerSettings_Simple{
						Simple: networkingv1alpha3.LoadBalancerSettings_RANDOM,
					},
				},
			},
		},
	}
	return
}

func GenerateVirtualService(name, namespace string, svcPort uint32) (vs *istioapi.VirtualService) {
	vs = &istioapi.VirtualService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: "networking.istio.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1alpha3.VirtualService{
			Hosts:    []string{"*"},
			Gateways: []string{name},
			Http: []*networkingv1alpha3.HTTPRoute{
				&networkingv1alpha3.HTTPRoute{
					Match: []*networkingv1alpha3.HTTPMatchRequest{
						&networkingv1alpha3.HTTPMatchRequest{
							Uri: &networkingv1alpha3.StringMatch{
								MatchType: &networkingv1alpha3.StringMatch_Prefix{
									Prefix: "/",
								},
							},
						},
					},
					Route: []*networkingv1alpha3.HTTPRouteDestination{
						&networkingv1alpha3.HTTPRouteDestination{
							Destination: &networkingv1alpha3.Destination{
								Host: name,
								Port: &networkingv1alpha3.PortSelector{
									Number: svcPort,
								},
							},
						},
					},
				},
			},
			Tcp: []*networkingv1alpha3.TCPRoute{
				&networkingv1alpha3.TCPRoute{
					Route: []*networkingv1alpha3.RouteDestination{
						&networkingv1alpha3.RouteDestination{
							Destination: &networkingv1alpha3.Destination{
								Host: name,
								Port: &networkingv1alpha3.PortSelector{
									Number: svcPort,
								},
							},
						},
					},
				},
			},
			Tls: []*networkingv1alpha3.TLSRoute{},
		},
	}
	return
}

func GenerateGateway(name, namespace, gatewayProtocol string, gatewayPort uint32) (gw *istioapi.Gateway) {
	gw = &istioapi.Gateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gateway",
			APIVersion: "networking.istio.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1alpha3.Gateway{
			Servers: []*networkingv1alpha3.Server{
				&networkingv1alpha3.Server{
					Hosts: []string{"*"},
					Port: &networkingv1alpha3.Port{
						Number:   gatewayPort,
						Protocol: gatewayProtocol,
						Name:     strings.ToLower(gatewayProtocol) + "-0",
					},
				},
			},
			Selector: map[string]string{
				constants.SelectorForEdgeMeshGatewayKey: constants.SelectorForEdgeMeshGatewayValue,
			},
		},
	}
	return
}