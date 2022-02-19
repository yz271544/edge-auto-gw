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
	klog.V(4).Info("trigger atAdd")
	at, ok := obj.(*v1.Service)
	if !ok {
		klog.Errorf("invalid type %v", obj)
		return
	}
	mgr.addAtGateway(at)
}

func (mgr *AutoGwManager) atUpdate(oldObj, newObj interface{}) {
	klog.V(4).Info("trigger atUpdate")
	at, ok := newObj.(*v1.Service)
	if !ok {
		klog.Errorf("invalid type %v", newObj)
		return
	}
	mgr.updateAtGateway(at)
}

func (mgr *AutoGwManager) atDelete(obj interface{}) {
	klog.V(4).Info("trigger atDelete")
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
	if atLables == nil {
		klog.Infof("atLables is Nil")
	}
	for key, value := range atLables {
		klog.Infof("LATBLES: %s ---> %s", key, value)
	}

	isExposeGateway, gatewayProtocol, svcPort, gatewayPort := extractGatewayConfig(atLables)
	klog.Infof("isExposeGateway:%v, gatewayProtocol:%s, svcPort:%d, gatewayPort:%d", isExposeGateway, gatewayProtocol, svcPort, gatewayPort)

	if isExposeGateway {
		ns := at.GetNamespace()
		nm := at.GetName()

		dr := GenerateDestinationRule(nm, ns)

		vs := GenerateVirtualService(nm, ns, gatewayProtocol, svcPort)

		gw := GenerateGateway(nm, ns, gatewayProtocol, gatewayPort)

		if dr == nil {
			klog.Errorf("auto add %s.%s failed, dr is nil", ns, nm)
			return
		}
		if vs == nil {
			klog.Errorf("auto add %s.%s failed, vs is nil", ns, nm)
			return
		}
		if gw == nil {
			klog.Errorf("auto add %s.%s failed, gw is nil", ns, nm)
			return
		}
		client := mgr.ifm.GetIstioClient().NetworkingV1alpha3()

		_, err = client.DestinationRules(ns).Create(context.Background(), dr, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("create destination rule failed: %v", err)
			return
		}

		_, err = client.VirtualServices(ns).Create(context.Background(), vs, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("create virtualservice rule failed: %v", err)
			return
		}
		_, err = client.Gateways(ns).Create(context.Background(), gw, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("create gateway rule failed: %v", err)
			return
		}
		klog.Infof("have created the gateway vs dr %s", nm)
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
	if atLables == nil {
		klog.Infof("atLables is Nil")
	}
	for key, value := range atLables {
		klog.Infof("LATBLES: %s ---> %s", key, value)
	}

	isExposeGateway, gatewayProtocol, svcPort, gatewayPort := extractGatewayConfig(atLables)
	klog.Infof("isExposeGateway:%v, gatewayProtocol:%s, svcPort:%d, gatewayPort:%d", isExposeGateway, gatewayProtocol, svcPort, gatewayPort)
	if isExposeGateway {
		ns := at.GetNamespace()
		nm := at.GetName()

		dr := GenerateDestinationRule(nm, ns)

		vs := GenerateVirtualService(nm, ns, gatewayProtocol, svcPort)

		gw := GenerateGateway(nm, ns, gatewayProtocol, gatewayPort)
		if dr == nil {
			klog.Errorf("auto update %s.%s failed, dr is nil", ns, nm)
			return
		}
		if vs == nil {
			klog.Errorf("auto update %s.%s failed, vs is nil", ns, nm)
			return
		}
		if gw == nil {
			klog.Errorf("auto update %s.%s failed, gw is nil", ns, nm)
			return
		}
		client := mgr.ifm.GetIstioClient().NetworkingV1alpha3()

		_, err = client.DestinationRules(ns).Update(context.Background(), dr, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("update destination rule failed: %v", err)
			return
		}

		_, err = client.VirtualServices(ns).Update(context.Background(), vs, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("update virtualservice rule failed: %v", err)
			return
		}
		_, err = client.Gateways(ns).Update(context.Background(), gw, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("update gateway rule failed: %v", err)
			return
		}
		klog.Infof("have updated the gateway vs dr %s", nm)
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

	client := mgr.ifm.GetIstioClient().NetworkingV1alpha3()

	ns := at.GetNamespace()
	nm := at.GetName()

	dr, err := client.DestinationRules(ns).Get(context.Background(), nm, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Get the DestinationRules:%s.%s failed", ns, nm)
	}
	if dr != nil {
		err = client.DestinationRules(ns).Delete(context.Background(), nm, metav1.DeleteOptions{})
		if err != nil {
			klog.Errorf("delete destinationrule failed: %v", err)
			return
		}
	}

	vs, err := client.VirtualServices(ns).Get(context.Background(), nm, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Get the VirtualServices:%s.%s failed", ns, nm)
	}
	if vs != nil {
		err = client.VirtualServices(ns).Delete(context.Background(), nm, metav1.DeleteOptions{})
		if err != nil {
			klog.Errorf("delete virtualservice failed: %v", err)
			return
		}
	}

	gw, err := client.Gateways(ns).Get(context.Background(), nm, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Get the Gateways:%s.%s failed", ns, nm)
	}
	if gw != nil {
		err = client.Gateways(ns).Delete(context.Background(), nm, metav1.DeleteOptions{})
		if err != nil {
			klog.Errorf("delete gateway failed: %v", err)
			return
		}
	}

	klog.Infof("have deleted the gateway vs dr %s", nm)
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
		} else {
			klog.V(4).Infof("gatewayPort %d is invalid scope", port)
		}
		isExposeGateway = true
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

func GenerateVirtualService(name, namespace, gatewayProtocol string, svcPort uint32) (vs *istioapi.VirtualService) {

	if gatewayProtocol == tcpProtocol {
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
			},
		}
	} else if gatewayProtocol == httpProtocol {
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
			},
		}
	} else {
		return
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
