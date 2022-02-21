package manager

import (
	"context"
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

	labelAn, err := Labels(at.GetLabels()).extractLabels()
	if err != nil {
		klog.Errorf("get labels extract %s", err)
		return
	}

	ns := at.GetNamespace()
	nm := at.GetName()

	dr := GenerateDestinationRule(nm, ns)

	vs := GenerateVirtualService(nm, ns, labelAn.GateWayProtocol, labelAn.ServicePort)

	gw := GenerateGateway(nm, ns, labelAn.GateWayProtocol, labelAn.GatewayPort)

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

// updateGateway update a gateway server
func (mgr *AutoGwManager) updateAtGateway(at *v1.Service) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	var err error
	if at == nil {
		klog.Errorf("gateway is nil")
		return
	}

	labelAn, err := Labels(at.GetLabels()).extractLabels()
	if err != nil {
		klog.Errorf("get labels extract %s", err)
		return
	}

	ns := at.GetNamespace()
	nm := at.GetName()

	dr := GenerateDestinationRule(nm, ns)

	vs := GenerateVirtualService(nm, ns, labelAn.GateWayProtocol, labelAn.ServicePort)

	gw := GenerateGateway(nm, ns, labelAn.GateWayProtocol, labelAn.GatewayPort)
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

// GenerateDestinationRule generate DestinationRule
func GenerateDestinationRule(name, namespace string) (dr *istioapi.DestinationRule) {
	dr = &istioapi.DestinationRule{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DestinationRule",
			APIVersion: istioapi.SchemeGroupVersion.String(),
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

func GenerateVirtualService(name, namespace string, gatewayProtocols []string, svcPorts []uint32) (vs *istioapi.VirtualService) {

	tcpRoutes := make([]*networkingv1alpha3.TCPRoute, 0)
	httpRoutes := make([]*networkingv1alpha3.HTTPRoute, 0)

	for i, gatewayProtocol := range gatewayProtocols {
		if gatewayProtocol == tcpProtocol {
			tcpRoute := &networkingv1alpha3.TCPRoute{
				Route: []*networkingv1alpha3.RouteDestination{
					{
						Destination: &networkingv1alpha3.Destination{
							Host: name,
							Port: &networkingv1alpha3.PortSelector{
								Number: svcPorts[i],
							},
						},
					},
				},
			}
			tcpRoutes = append(tcpRoutes, tcpRoute)
		} else if gatewayProtocol == httpProtocol {
			httpRoute := &networkingv1alpha3.HTTPRoute{
				Match: []*networkingv1alpha3.HTTPMatchRequest{
					{
						Uri: &networkingv1alpha3.StringMatch{
							MatchType: &networkingv1alpha3.StringMatch_Prefix{
								Prefix: "/",
							},
						},
					},
				},
				Route: []*networkingv1alpha3.HTTPRouteDestination{
					{
						Destination: &networkingv1alpha3.Destination{
							Host: name,
							Port: &networkingv1alpha3.PortSelector{
								Number: svcPorts[i],
							},
						},
					},
				},
			}
			httpRoutes = append(httpRoutes, httpRoute)
		}
	}

	vs = &istioapi.VirtualService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: istioapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1alpha3.VirtualService{
			Hosts:    []string{"*"},
			Gateways: []string{name},
			Tcp:      tcpRoutes,
			Http:     httpRoutes,
		},
	}

	return
}

func GenerateGateway(name, namespace string, gatewayProtocols []string, gatewayPorts []uint32) (gw *istioapi.Gateway) {

	servers := make([]*networkingv1alpha3.Server, 0)

	for i, gatewayProtocol := range gatewayProtocols {
		server := &networkingv1alpha3.Server{
			Hosts: []string{"*"},
			Port: &networkingv1alpha3.Port{
				Number:   gatewayPorts[i],
				Protocol: gatewayProtocol,
				Name:     strings.ToLower(gatewayProtocol) + "-0",
			},
		}

		servers = append(servers, server)

	}

	gw = &istioapi.Gateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gateway",
			APIVersion: istioapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1alpha3.Gateway{
			Servers: servers,
			Selector: map[string]string{
				constants.SelectorForEdgeMeshGatewayKey: constants.SelectorForEdgeMeshGatewayValue,
			},
		},
	}
	return
}
