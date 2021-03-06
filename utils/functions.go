package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	sshconf "github.com/dorzheh/infra/comm/common"
	"github.com/dorzheh/infra/comm/ssh"
)

type ConnFuncAlias func(*sshconf.Config) (*ssh.SshConn, error)

func ConnFunc(config *sshconf.Config) func() (*ssh.SshConn, error) {
	return func() (*ssh.SshConn, error) {
		c, err := ssh.NewSshConn(config)
		if err != nil {
			return nil, err
		}
		return c, nil
	}
}

// RunFunc is a generic solution for running appropriate commands
// on local or remote host
func RunFunc(config *sshconf.Config) func(string) (string, error) {
	if config == nil {
		return func(command string) (string, error) {
			var stderr bytes.Buffer
			var stdout bytes.Buffer
			c := exec.Command("/bin/bash", "-c", command)
			c.Stderr = &stderr
			c.Stdout = &stdout
			if err := c.Start(); err != nil {
				return "", err
			}
			if err := c.Wait(); err != nil {
				return "", fmt.Errorf("executing %s  : %s [%s]", command, stderr.String(), err)
			}
			return strings.TrimSpace(stdout.String()), nil
		}
	}
	return func(command string) (string, error) {
		c, err := ssh.NewSshConn(config)
		if err != nil {
			return "", err
		}
		defer c.ConnClose()
		outstr, errstr, err := c.Run("sudo " + command)
		if err != nil {
			return "", fmt.Errorf("executing %s : %s [%s]", command, errstr, err)
		}
		return strings.TrimSpace(outstr), nil
	}
}

// InterruptHandler is trying to release appropriate image
// in case SIGHUP, SIGINT or SIGTERM signal received
func InterruptHandler(fn func() error) {
	//create a channel for interrupt handler
	interrupt := make(chan os.Signal, 1)
	// create an interrupt handler
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	// run a seperate goroutine.It's  job is handling events in case a signal has been sent
	go func() {
		for {
			select {
			case <-interrupt:
				fn()
			}
		}
	}()
}
