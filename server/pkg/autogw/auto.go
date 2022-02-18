package autogw

import (
	"time"

	"k8s.io/klog/v2"
)

func (t *EdgeAutoGw) Run() {
	for {
		klog.Info("auto heartbeat time")
		// heartbeat time
		time.Sleep(10 * time.Second)
	}
}
