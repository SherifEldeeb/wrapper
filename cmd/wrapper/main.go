//go:generate goversioninfo
// +build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SherifEldeeb/wrapper/pkg/service"
)

func printUsage() {
	fmt.Println(`
##################
# SvcWrapper 0.1 #
##################
twitter.com/0xdeeb
    
    - For windows programs to run as services, they have to be developed in a certain way to respond to 
      certain messages from the Windows OS ... otherwise windows kills them after 30 seconds.
    - SvcWrapper wraps "normal" (non-service) Windows executables enabling them to run as a service.
    - It enables "installing", "removing" and "running" applications as services.
    
    Usage: wrapper.exe SERVICE_NAME COMMAND [ARGS]
        - SERVICE_NAME is the name of the service
        - COMMAND is either 'install', 'remove' or 'run'

            Example 01:
            - > wrapper.exe kolide_launcher_service install launcher.exe --host=192.168.0.10 --insecure
                This will create a service 'kolide_launcher_service' that will execute the 'launcher.exe'
                with the arguments that has been provided

            Example 02:
            - > wrapper.exe kolide_launcher_service remove
				This will stop and remove 'kolide_launcher_service' service
`)
	os.Exit(1)
}

func main() {
	var err error
	if len(os.Args) < 3 {
		printUsage()
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
		fmt.Printf("invalid command %s", cmd)
		printUsage()
	}

	if err != nil {
		panic(fmt.Sprintf("failed to %s %s: %v", cmd, svcName, err))
	}
	return
}
