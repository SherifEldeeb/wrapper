package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/mgr"
)

type myservice struct{}

func main() {
	// flag
	var svcName string
	flag.StringVar(&svcName, "name", "GO_SERVIFY", "name of the service")
	var desc string
	flag.StringVar(&svcName, "desc", "GO_SERVIFY Description", "description of the service")
	var cmd string
	flag.StringVar(&cmd, "cmd", "list", "Command (install, remove, list")

	flag.Parse()
	//
	var err error
	// isIntSess, err := svc.IsAnInteractiveSession()
	// if err != nil {
	// 	log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	// }

	switch cmd {
	// case "debug":
	// 	runService(svcName, true)
	// 	return
	case "install":
		err = installService(svcName, desc)
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
		fmt.Printf("invalid command %s", cmd)
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
	}
	return
}

func (m *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	fasttick := time.Tick(500 * time.Millisecond)
	slowtick := time.Tick(2 * time.Second)
	tick := fasttick
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case <-tick:
			fmt.Println("beep")
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				tick = slowtick
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				tick = fasttick
			default:
				fmt.Printf("unexpected control request #%d", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func installService(name, desc string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	exepath, err := exePath()
	if err != nil {
		return err
	}
	// service exists?
	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", name)
	}

	// create service
	s, err = m.CreateService(name, exepath, mgr.Config{DisplayName: name, Description: desc})
	if err != nil {
		return err
	}
	s.Close()
	fmt.Println("service installed:", name)
	return nil
}

func runService(name string, isDebug bool) {
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err := run(name, &myservice{})
	if err != nil {
		fmt.Println("%s service failed: %v", name, err)
		return
	}
	fmt.Println("%s service stopped", name)
}

func removeService(name string) error {
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
