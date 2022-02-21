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
	"github.com/spf13/cast"
	"github.com/yz271544/edge-auto-gw/server/pkg/autogw/controller"
	"strings"
)

const GatewayPortSeparate = "-"

type LabelAnnotation struct {
	ServicePort     uint32
	ServiceProtocol string
	GatewayPort     uint32
	GateWayProtocol string
}

type Labels map[string]string

// extractLabels return k8s service label information.
func (l Labels) extractLabels() (*LabelAnnotation, error) {
	gatewayProtocal, ok := l[controller.LabelEdgemeshGatewayConfig]
	if !ok {
		return nil, fmt.Errorf("not have %s label in the service", controller.LabelEdgemeshGatewayConfig)
	}
	gatewayPorts, ok := l[controller.LabelEdgemeshGatewayPort]
	if !ok {
		return nil, fmt.Errorf("not have %s label in the service", controller.LabelEdgemeshGatewayPort)
	}

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

	return &LabelAnnotation{
		ServicePort:     servicePort,
		GatewayPort:     gatewayPort,
		GateWayProtocol: gatewayProtocal,
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
