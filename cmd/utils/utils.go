package utils

import (
	"fmt"
)

var version = "dev"

const banner = `
        ██████╗ ███████╗██╗   ██╗ ██████╗  ██████╗ ███████╗████████╗
        ██╔══██╗██╔════╝██║   ██║██╔═══██╗██╔═══██╗██╔════╝╚══██╔══╝
        ██║  ██║█████╗  ██║   ██║██║   ██║██║   ██║███████╗   ██║   
        ██║  ██║██╔══╝  ╚██╗ ██╔╝██║   ██║██║   ██║╚════██║   ██║   
        ██████╔╝███████╗ ╚████╔╝ ╚██████╔╝╚██████╔╝███████║   ██║   
        ╚═════╝ ╚══════╝  ╚═══╝   ╚═════╝  ╚═════╝ ╚══════╝   ╚═╝   

                dflow %s - Git branching made simple
        
`

func SetVersion(v string) {
	version = v
}

func PrintBanner() {
	fmt.Printf(banner, version)
}
