/*
TODO: catch sigint, etc.
orderly shutdown on signal
call ReallocateTask on server if we're going down
call CheckTask during task run, graceful kill if CheckTask returns False
*/

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/kmanley/midtown"
	"io"
	_ "io/ioutil"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Worker struct {
	name       string
	serverName string
	serverPort int
	conn       *rpc.Client
}

func NewWorker(serverName string, serverPort int) (*Worker, error) {
	hostname, err := os.Hostname()
	if err != nil {
		glog.Error("can't get hostname: %s", err)
		return nil, err
	}
	name := fmt.Sprintf("%s:%d", hostname, os.Getpid())
	worker := &Worker{name, serverName, serverPort, nil}
	glog.V(1).Infof("created worker %s", name)
	return worker, nil
}

func (this *Worker) Connect() error {
	distributor := fmt.Sprintf("%s:%d", this.serverName, this.serverPort)
	glog.V(1).Infof("attempting to connect to %s", distributor)
	conn, err := rpc.Dial("tcp", distributor)
	if err != nil {
		glog.Errorf("can't connect to distributor %s: %s", distributor, err)
		this.conn = nil
		return err
	}
	this.conn = conn
	return nil
}

// TODO: check shutdown channel
func (this *Worker) ConnectRetry() error {
	backoff := []time.Duration{1, 2, 5, 10, 20, 30}
	ctr := 0
	for {
		err := this.Connect()
		if err == nil {
			return nil
		}
		if shouldReconnect(err) {
			if ctr >= len(backoff) {
				ctr = len(backoff) - 1
			}
			time.Sleep(backoff[ctr] * time.Second)
		}
		ctr += 1
	}
}

// Returns true if based on the error we should try reconnecting to the distributor
func shouldReconnect(err error) bool {
	if err == rpc.ErrShutdown || err == io.EOF || err == io.ErrUnexpectedEOF {
		return true
	} else if _, ok := err.(*net.OpError); ok {
		// e.g. connection refused...
		return true
	} else {
		return false
	}
}

func (this *Worker) GetNextTask() *midtown.WorkerTask {
	var task midtown.WorkerTask
	err := this.conn.Call("WorkerApi.GetWorkerTask", this.name, &task)
	if err != nil {
		if shouldReconnect(err) {
			this.ConnectRetry()
			return nil
		}
		glog.Errorf("failed to get task: %s", err)
		return nil
	}
	if task.Job == "" {
		glog.V(1).Infof("no task")
		return nil
	} else {
		glog.Infof("got task %s:%d", task.Job, task.Seq)
		if glog.V(1) {
			glog.Info(spew.Sdump(task))
		}
		return &task
	}
}

func (this *Worker) SetTaskDone(taskResult *midtown.TaskResult) error {
	var ok bool
	err := this.conn.Call("WorkerApi.SetTaskDone", taskResult, &ok)
	if err != nil {
		// TODO:
		return err
	}
	glog.V(1).Infof("set task %s:%d done", taskResult.Job, taskResult.Seq)
	return nil
}

func (this *Worker) RunTask(task *midtown.WorkerTask) *midtown.TaskResult {
	taskResult := &midtown.TaskResult{WorkerName: this.name, Job: task.Job, Seq: task.Seq}
	cmd := exec.Command(task.Cmd, task.Args...)
	cmd.Dir = task.Dir

	input := []interface{}{task.Job, task.Seq, task.Data, task.Ctx}
	var indata bytes.Buffer
	enc := json.NewEncoder(&indata)
	err := enc.Encode(input)
	if err != nil {
		glog.Errorf("failed to encode input data %v", input)
		taskResult.Error = err // TODO: wrap in a better error?
		return taskResult
	}
	cmd.Stdin = strings.NewReader(indata.String())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	fmt.Println(cmd) // TODO:

	if err := cmd.Run(); err != nil {
		glog.Errorf("failed to run cmd: %v", err)
		taskResult.Error = err
		return taskResult
	}

	fmt.Println("stdout: ", stdout.String())
	fmt.Println("stderr: ", stderr.String())

	var outdata interface{}
	if err := json.Unmarshal(stdout.Bytes(), &outdata); err != nil {
		glog.Errorf("failed to decode stdout: %v", err)
		taskResult.Error = err
		return taskResult
	}
	fmt.Println("outdata: ", outdata)

	taskResult.Stderr = stderr.String()
	taskResult.Result = outdata
	return taskResult
}

func main() {

	//flag.Usage = func() {
	//	fmt.Fprintln(os.Stderr, "Usage: midtownw <distributor hostname>")
	//	flag.PrintDefaults()
	//}

	var svr = flag.String("svr", "localhost", "distributor hostname")
	var basePort = flag.Int("port", 6877, "distributor base port")
	flag.Parse()

	/*
		if flag.NArg() != 1 {
			flag.Usage()
			os.Exit(1)
		}
	*/

	worker, _ := NewWorker(*svr, *basePort+2)
	worker.ConnectRetry()

	for {
		task := worker.GetNextTask()
		if task != nil {
			// TODO: distinguish between failure to run task and an error returned by
			// the task subproc?
			taskResult := worker.RunTask(task)
			worker.SetTaskDone(taskResult) // TODO: err handling

		} else {
			// TODO: orderly shutdown, signal handler etc.
			// TODO: exponential backoff
			time.Sleep(1 * time.Second)
		}
	}

}
