package gracefully

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	hookableSignals []os.Signal
)

func init() {
	hookableSignals = []os.Signal{
		syscall.SIGHUP,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
		syscall.SIGTSTP,
	}
}

func newSignalHookFunc() map[int]map[os.Signal][]func() {
	return map[int]map[os.Signal][]func(){
		PreSignal: {
			syscall.SIGHUP:  []func(){},
			syscall.SIGUSR1: []func(){},
			syscall.SIGUSR2: []func(){},
			syscall.SIGINT:  []func(){},
			syscall.SIGQUIT: []func(){},
			syscall.SIGTERM: []func(){},
			syscall.SIGTSTP: []func(){},
		},
		PostSignal: {
			syscall.SIGHUP:  []func(){},
			syscall.SIGUSR1: []func(){},
			syscall.SIGUSR2: []func(){},
			syscall.SIGINT:  []func(){},
			syscall.SIGQUIT: []func(){},
			syscall.SIGTERM: []func(){},
			syscall.SIGTSTP: []func(){},
		},
	}
}

// handleSignals listens for os Signals and calls any hooked in function that the
// user had registered with the signal.
func (e *endlessServer) handleSignals() {
	var sig os.Signal

	signal.Notify(
		e.sigChan,
		hookableSignals...,
	)

	pid := syscall.Getpid()
	for {
		select {
		case sig = <-e.sigChan:
		case <-e.close:
			return
		}

		e.signalHooks(PreSignal, sig)
		switch sig {
		case syscall.SIGHUP:
			signal.Stop(e.sigChan)
			close(e.sigChan)
			log.Println(pid, "[Common] endless received SIGHUP. forking...")
			if err := e.fork(); err != nil {
				log.Println("[Common] endless fork err:", err)
			}
			return
		case syscall.SIGUSR1:
			log.Println(pid, "[Common] endless received SIGUSR1.")
		case syscall.SIGUSR2:
			signal.Stop(e.sigChan)
			close(e.sigChan)
			log.Println(pid, "[Common] endless received SIGUSR2.")
			e.hammerTime(0 * time.Second)
			return
		case syscall.SIGINT:
			signal.Stop(e.sigChan)
			close(e.sigChan)
			log.Println(pid, "[Common] endless received SIGINT.")
			e.Shutdown()
			return
		case syscall.SIGQUIT:
			signal.Stop(e.sigChan)
			close(e.sigChan)
			log.Println(pid, "[Common] endless received SIGQUIT.")
			e.Shutdown()
			return
		case syscall.SIGTERM:
			signal.Stop(e.sigChan)
			close(e.sigChan)
			log.Println(pid, "[Common] endless received SIGTERM.")
			e.Shutdown()
			return
		case syscall.SIGTSTP:
			log.Println(pid, "[Common] endless received SIGTSTP.")
		default:
			log.Printf("[Common] endless received %v: nothing we care about...\n", sig)
		}
		e.signalHooks(PostSignal, sig)
	}
}

func (e *endlessListener) Close() error {
	if e.stopped {
		return syscall.EINVAL
	}

	e.stopped = true
	return e.Listener.Close()
}

func syscallKill(ppid int) error {
	return syscall.Kill(ppid, syscall.SIGTERM)
}
