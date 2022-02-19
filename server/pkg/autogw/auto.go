package autogw

import (
	"time"
)

func (t *EdgeAutoGw) Run() {
	for {
		// klog.Info("auto heartbeat time")
		// heartbeat time
		time.Sleep(10 * time.Second)
	}
}
