package main

import (
	"github.com/yepizrene-devoost/dflow/cmd/root"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

func main() {
	utils.HandleInterrupt()
	root.Execute()
}
