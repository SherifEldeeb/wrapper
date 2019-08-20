package service

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/SherifEldeeb/wrapper/pkg/tools"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/mgr"
)

// GetByName returns a mgr.Sevive given a service name
// returned mgr.Service object has to be closed to release resources.
func GetByName(name string) (s *mgr.Service, err error) {
	m, err := mgr.Connect()
	if err != nil {
		return
	}
	defer m.Disconnect()

	return m.OpenService(name)
}

// Run starts executing the program
func Run(name string, args ...string) {
	run := svc.Run
	// interactive?
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}
	if !isIntSess {
		run = debug.Run
	}

	var w = NewWrapper(args[0], args[1:])

	err = run(name, w)
	if err != nil {
		fmt.Printf("%s service failed: %s", name, err)
		return
	}
}

// Remove removes service by name
func Remove(name string) error {
	s, err := GetByName(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()
	s.Control(svc.Stop)
	time.Sleep(2 * time.Second)
	err = s.Delete()
	if err != nil {
		return err
	}
	return nil
}

// Install creates a service
// - startType can be one of: "auto", "manual" or "disabled"
func Install(name, desc, startType string, args ...string) error {
	// func Install(name, desc, startType, hostname, secretPath string) error {
	fmt.Printf("Installing service: Name: '%s', Descritpion: '%s'", name, desc)

	fmt.Print("connecting to service manager...")
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	fmt.Print("Getting executable path...")
	exepath, err := tools.GetExePath()
	if err != nil {
		return err
	}
	fmt.Printf("exepath: %s", exepath)
	// service exists?
	fmt.Print("Checking if service already exists...")
	fmt.Printf("Trying to open service: %s", name)
	s, err := m.OpenService(name)
	if err == nil {
		fmt.Printf("service %s already exists, removing it...", name)
		s.Control(svc.Stop)
		time.Sleep(2 * time.Second)
		err = s.Delete()
		if err != nil {
			return fmt.Errorf("Error removing service: %s", err)
		}
		s.Close()
	} else {
		fmt.Printf("Serive doesn't exist, proceeding with install...")
	}
	fmt.Printf("Creating Service: %s", name)

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
		fmt.Printf("Unknown Service Start type: %s; Only auto, manual and disabled are supported", startType)
		os.Exit(1)
	}
	// create service
	fullArgs := append([]string{name, "run"}, args...)
	s, err = m.CreateService(name, exepath, mgr.Config{
		DisplayName: name,
		Description: desc,
		StartType:   stype,
	},
		fullArgs...,
	// "run",
	// hostname,
	// secretPath,
	)
	if err != nil {
		return err
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		return err
	}
	fmt.Println("service installed and started:", name)
	return nil
}
