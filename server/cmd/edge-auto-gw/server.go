package main

import (
	"math/rand"
	"os"
	"time"

	"k8s.io/component-base/logs"

	"github.com/yz271544/edge-auto-gw/server/cmd/edge-auto-gw/app"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	command := app.NewEdgeAutoGwServerCommand()

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
