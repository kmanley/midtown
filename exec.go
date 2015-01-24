package midtown

import (
	"errors"
	"github.com/golang/glog"
	"os"
	"os/exec"
	"time"
)

var ErrTimeout = errors.New("Timeout elapsed")

type Done chan error

type GracefulCmd struct {
	*exec.Cmd
	done Done
}

func GracefulCommand(c *exec.Cmd) *GracefulCmd {
	return &GracefulCmd{c, nil}
}

func (this *GracefulCmd) Start() error {
	err := this.Cmd.Start()
	if err != nil {
		return err
	}
	this.done = make(Done, 1)
	go func() {
		this.done <- this.Cmd.Wait()
	}()
	return nil
}

func (this *GracefulCmd) Wait(timeout time.Duration) error {
	var err error
	if timeout == 0 {
		err = <-this.done
		return err
	} else {
		select {
		case <-time.After(timeout):
			{
				return ErrTimeout
			}
		case err = <-this.done:
			{
				return err
			}
		}
	}
}

func (this *GracefulCmd) Kill(waitTime time.Duration) error {
	err := this.Cmd.Process.Signal(os.Interrupt)
	if err != nil {
		glog.V(1).Infof("sending SIGINT to %s failed: %s", this.Cmd.Process.Pid, err)
		// fall through
	} else {
		glog.Infof("sent SIGINT to %d; waiting up to %s for exit...", this.Cmd.Process.Pid, waitTime.String())
		select {
		case err = <-this.done:
			{
				// process exited
				return err
			}
		case <-time.After(waitTime):
			{
				// SIGINT timeout elapsed
				break
			}
		}
	}
	glog.Infof("pid %d still running; sending SIGKILL", this.Cmd.Process.Pid)
	err = this.Cmd.Process.Kill()
	if err != nil {
		return err
	}
	err = <-this.done
	return err
}
