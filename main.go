// +build windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/mgr"
)

type launcherSVC struct {
	HostName         string
	EnrollSecretFile string
}

var ilog *log.Logger // info logger

func main() {
	ilog = log.New(os.Stderr, "[INFO] ", 0)
	var err error

	if len(os.Args) != 4 {
		ilog.Fatal("Number of args has to be 3 (cmd host enroll_secret_path)")
	}

	// flag
	var svcName = "EBRIGADE_Kolide_Wrapper"
	// flag.StringVar(&svcName, "name", "GO_SERVIFY", "name of the service")

	var desc = "EBRIGADE windows service wrapper for Kolide Launcher"
	// flag.StringVar(&desc, "desc", "GO_SERVIFY Description", "description of the service")

	var cmd = os.Args[1]
	// flag.StringVar(&cmd, "cmd", "list", "Command (install, remove, list")

	var startType = "auto"
	// flag.StringVar(&startType, "start", "auto", "Service Start Type (auto, manual or disabled")

	var hostname = os.Args[2]
	// flag.StringVar(&hostname, "hostname", "127.0.0.1:8080", "The hostname of the gRPC server.")

	var secretPath = os.Args[3]
	// flag.StringVar(&secretPath, "secret", ".\\enroll", "the path to the enrollment secret file")

	// flag.Parse()
	// /flag

	// interactive?
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		ilog.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}
	if !isIntSess {
		runService(svcName, false)
		return
	}

	switch cmd {
	case "install":
		err = installService(svcName, desc, startType, hostname, secretPath)
	case "remove":
		err = removeService(svcName)
	case "run":
		runService(svcName, true)
		return
	default:
		ilog.Printf("invalid command %s", cmd)
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
	}
	return
}

func (m *launcherSVC) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	ilog.Printf("SVC Execute Args: %#v", os.Args)

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	// Start App
	exitChan := make(chan bool)
	cmd := exec.Command("launcher.exe", fmt.Sprintf("--hostname=%s", os.Args[2]), fmt.Sprintf("--enroll_secret_path=%s", os.Args[3]), "--insecure")
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

func installService(name, desc, startType, hostname, secretPath string) error {
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
		"run",
		hostname,
		secretPath,
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
	err := run(name, &launcherSVC{})
	if err != nil {
		ilog.Printf("%s service failed: %s", name, err)
		return
	}
	ilog.Printf("%s service stopped", name)
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
