package validation

import (
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/yz271544/edge-auto-gw/cloud/cmd/edge-auto-gw/app/config"
)

func ValidateEdgeAutoGwConfiguration(c *config.EdgeAutoGwConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validation.ValidateKubeAPIConfig(*c.KubeAPIConfig)...)
	return allErrs
}
