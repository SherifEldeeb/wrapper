// +build windows

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/mgr"
)

type myservice struct{}

var ilog *log.Logger // info logger
var dlog *log.Logger // debug logger
var fullCmd string   // full command

func main() {
	// flag
	var svcName string
	flag.StringVar(&svcName, "name", "GO_SERVIFY", "name of the service")

	var desc string
	flag.StringVar(&desc, "desc", "GO_SERVIFY Description", "description of the service")

	var cmd string
	flag.StringVar(&cmd, "cmd", "list", "Command (install, remove, list")

	var startType string
	flag.StringVar(&startType, "start", "auto", "Service Start Type (auto, manual or disabled")

	flag.StringVar(&fullCmd, "wrap", "", "Full command to be wrapped as a service")

	flag.Parse()
	// /flag
	var err error
	// isIntSess, err := svc.IsAnInteractiveSession()
	// if err != nil {
	// 	log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	// }
	ilog = log.New(os.Stderr, "[INFO] ", 0)

	switch cmd {
	// case "debug":
	// 	runService(svcName, true)
	// 	return
	case "install":
		if fullCmd == "" {
			ilog.Fatal("Install without 'wrap' won't work!")
		}
		err = installService(svcName, desc, startType, fullCmd)
	case "remove":
		err = removeService(svcName)
	case "list":
		err = printServices()
	// case "start":
	// 	err = startService(svcName)
	// case "stop":
	// 	err = controlService(svcName, svc.Stop, svc.Stopped)
	// case "pause":
	// 	err = controlService(svcName, svc.Pause, svc.Paused)
	// case "continue":
	// 	err = controlService(svcName, svc.Continue, svc.Running)
	default:
		ilog.Printf("invalid command %s", cmd)
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
	}
	return
}

func (m *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	// Start App
	exitChan := make(chan bool)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	go func(cmd *exec.Cmd, exitChan chan bool) {
		err := cmd.Run()
		if err != nil {
			ilog.Printf("Error executing command:%v %v\nError:%s", cmd.Path, cmd.Args, err)
		}
		exitChan <- true
	}(cmd, exitChan)
	//
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case <-exitChan: // we done?
			break loop
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				// changes <- c.CurrentStatus
				// // Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				// time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			default:
				fmt.Printf("unexpected control request #%d", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func installService(name, desc, startType, fullCmd string) error {
	ilog.Printf("Installing service: Name: '%s', Descritpion: '%s'", name, desc)

	ilog.Print("connecting to service manager...")
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	ilog.Print("Getting executable path...")
	exepath, err := exePath()
	if err != nil {
		return err
	}
	ilog.Printf("exepath: %s", exepath)
	// service exists?
	ilog.Print("Checking if service already exists...")
	ilog.Printf("Trying to open service: %s", name)
	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", name)
	}
	ilog.Printf("Serive doesn't exist, proceeding with install...")
	ilog.Printf("Creating Service: %s", name)

	// svcstart type
	var stype uint32
	switch startType {
	case "auto":
		stype = mgr.StartAutomatic
	case "manual":
		stype = mgr.StartManual
	case "disabled":
		stype = mgr.StartDisabled
	default:
		ilog.Fatalf("Unknown Service Start type: %s; Only auto, manual and disabled are supported", startType)
	}
	// create service
	s, err = m.CreateService(name, exepath, mgr.Config{
		DisplayName: name,
		Description: desc,
		StartType:   stype,
	},
		strings.Split(fullCmd, " ")...,
	)
	if err != nil {
		return err
	}
	s.Close()
	ilog.Println("service installed:", name)
	return nil
}

func runService(name string, isDebug bool) {
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err := run(name, &myservice{})
	if err != nil {
		fmt.Printf("%s service failed: %s", name, err)
		return
	}
	fmt.Printf("%s service stopped", name)
}

func removeService(name string) error {
	ilog.Printf("Removing service: '%s'", name)

	ilog.Print("connecting to service manager...")
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}
	ilog.Printf("Service removed.")

	return nil
}

func listServices() ([]string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, err
	}
	defer m.Disconnect()
	return m.ListServices()
}

func printServices() error {
	svcs, err := listServices()
	if err != nil {
		return err
	}
	j, _ := json.MarshalIndent(svcs, "", "    ")
	fmt.Println(string(j))
	return nil
}

func exePath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}
	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}
	return "", err
}
