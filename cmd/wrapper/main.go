//go:generate goversioninfo
// +build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/SherifEldeeb/wrapper/pkg/service"
)

func main() {
	var err error
	if len(os.Args) < 3 {
		fmt.Println("go home")
		os.Exit(1)
	}
	var svcName = os.Args[1]
	var cmd = os.Args[2]

	// change dir to where the exe is
	exename, _ := os.Executable()
	os.Chdir(filepath.Dir(exename))

	switch cmd {
	case "install":
		if len(os.Args) > 3 { // exe + svcname + cmd + (something has to be here)
			err = service.Install(svcName, "SvcWrapper", "auto", os.Args[3:]...)
		}
	case "remove":
		if len(os.Args) > 2 { // exe + svcname + cmd
			err = service.Remove(svcName)
		}
	case "run":
		if len(os.Args) > 3 { // exe + svcname + cmd + (something has to be here)
			service.Run(svcName, os.Args[3:]...)
		}
		return
	default:
		log.Printf("invalid command %s", cmd)
	}

	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
	}
	return
}
