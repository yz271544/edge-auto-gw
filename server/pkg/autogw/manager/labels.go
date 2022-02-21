/*
Copyright 2021 The Gridsum Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package manager

import (
	"fmt"
	"strings"

	"github.com/spf13/cast"
	"github.com/yz271544/edge-auto-gw/server/pkg/autogw/controller"
)

const GatewayPortSeparate = "-"
const GroupSparate = "."

type LabelAnnotation struct {
	ServicePort     []uint32
	ServiceProtocol []string
	GatewayPort     []uint32
	GateWayProtocol []string
}

type Labels map[string]string

// extractLabels return k8s service label information.
func (l Labels) extractLabels() (*LabelAnnotation, error) {
	gatewayProtocols, ok := l[controller.LabelEdgemeshGatewayConfig]
	if !ok {
		return nil, fmt.Errorf("not have %s label in the service", controller.LabelEdgemeshGatewayConfig)
	}
	gatewayPorts, ok := l[controller.LabelEdgemeshGatewayPort]
	if !ok {
		return nil, fmt.Errorf("not have %s label in the service", controller.LabelEdgemeshGatewayPort)
	}

	gatewayProtocolGroups := strings.Split(gatewayProtocols, GroupSparate)
	if len(gatewayProtocolGroups) == 0 {
		return nil, fmt.Errorf("not config the protocol values, example:%s", "TCP.TCP or TCP.HTTP")
	}
	gatewayPortGroups := strings.Split(gatewayPorts, GroupSparate)
	if len(gatewayPortGroups) == 0 {
		return nil, fmt.Errorf("not config the port values, example:%s", "TCP.TCP or TCP.HTTP")
	}

	if len(gatewayProtocolGroups) != len(gatewayPortGroups) {
		return nil, fmt.Errorf("config error: protocol groups [%d] not equals with port groups [%d]", len(gatewayProtocolGroups), len(gatewayPortGroups))
	}

	servicePortBox := make([]uint32, 0)
	gatewayPortBox := make([]uint32, 0)
	ServiceProtocolBox := make([]string, 0)
	gatewayProtocolBox := make([]string, 0)

	for _, gatewayProtocol := range gatewayProtocolGroups {

		gatewayProtocolBox = append(gatewayProtocolBox, strings.ToUpper(gatewayProtocol))
		ServiceProtocolBox = append(ServiceProtocolBox, strings.ToLower(gatewayProtocol))

		ports := strings.Split(gatewayPorts, GatewayPortSeparate)
		if len(ports) != 2 {
			return nil, fmt.Errorf("%s must containes from servicePort to gatewayPort", controller.LabelEdgemeshGatewayPort)
		}

		servicePort := cast.ToUint32(ports[0])
		if ok := ValidateServicePort(servicePort); !ok {
			return nil, fmt.Errorf("service port must >0 and < 65535", servicePort)
		}

		gatewayPort := cast.ToUint32(ports[1])
		if ok := ValidateGatewayPort(gatewayPort); !ok {
			return nil, fmt.Errorf("gateway port must > 30000 and < 65535", gatewayPort)
		}

		servicePortBox = append(servicePortBox, servicePort)
		gatewayPortBox = append(gatewayPortBox, gatewayPort)

	}

	return &LabelAnnotation{
		ServicePort:     servicePortBox,
		GatewayPort:     gatewayPortBox,
		ServiceProtocol: ServiceProtocolBox,
		GateWayProtocol: gatewayProtocolBox,
	}, nil
}

func ValidateServicePort(p uint32) bool {
	if p > 0 && p <= maxGatewayPort {
		return true
	}
	return false
}

func ValidateGatewayPort(p uint32) bool {
	if p >= minGatewayPort && p <= maxGatewayPort {
		return true
	}
	return false
}
