package main

import (
	"github.com/yepizrene-devoost/dflow/cmd/root"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

var version = "dev"

func main() {
	utils.SetVersion(version)
	utils.HandleInterrupt()
	root.Execute()
}
