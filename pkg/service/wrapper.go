package service

import (
	"fmt"
	"os/exec"
	"strconv"

	"golang.org/x/sys/windows/svc"
)

// NewWrapper returns a Wrapper
func NewWrapper(exeName string, args []string) Wrapper {
	return Wrapper{exeName, args}
}

// Wrapper implements the svc.Handler interface,
// and wraps an executable allowing it to run as a service
type Wrapper struct {
	// Those will be passed to exec.Command(w.ExeName, w.Args...)
	exeName string // e.g. Launcher.exe
	args    []string
}

// GetCommand returns a slice of command+args
func (w Wrapper) GetCommand() []string {
	return append([]string{w.exeName}, w.args...)
}

// Execute will be called by the package code at the start of
// the service, and the service will exit once Execute completes.
// Inside Execute you must read service change requests from r and
// act accordingly. You must keep service control manager up to date
// about state of your service by writing into s as required.
// args contains service name followed by argument strings passed
// to the service.
// You can provide service exit code in exitCode return parameter,
// with 0 being "no error". You can also indicate if exit code,
// if any, is service specific or not by using svcSpecificEC
// parameter.
func (w *Wrapper) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	// Start App
	exitChan := make(chan bool)
	cmd := exec.Command(w.exeName, w.args...)
	// cmd := exec.Command("launcher.exe", fmt.Sprintf("--hostname=%s", os.Args[2]), fmt.Sprintf("--enroll_secret_path=%s", os.Args[3]), "--insecure")
	// if isDebug {
	// 	cmd.Stdout = os.Stdout
	// 	cmd.Stderr = os.Stderr
	// }
	go func(cmd *exec.Cmd, exitChan chan bool) {
		err := cmd.Run()
		if err != nil { // TODO
			// ilog.Printf("Error executing command:%v %v\nError:%s", cmd.Path, cmd.Args, err)
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
				kill := exec.Command("TASKKILL.exe", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
				// if isDebug {
				// 	kill.Stderr = os.Stderr
				// 	kill.Stdout = os.Stdout
				// }
				kill.Run()
				break loop
			default:
				fmt.Printf("unexpected control request #%d", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}
