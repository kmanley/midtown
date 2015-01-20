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
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type WorkerApp struct {
	n          int
	name       string
	serverName string
	serverPort int
	quitChan   *chan bool
	conn       *rpc.Client
}

func NewWorkerApp(n int, serverName string, serverPort int, quitChan *chan bool) (*WorkerApp, error) {
	hostname, err := os.Hostname()
	if err != nil {
		glog.Error("can't get hostname: %s", err)
		return nil, err
	}
	name := fmt.Sprintf("%s:%d", hostname, os.Getpid())
	worker := &WorkerApp{n: n, name: name, serverName: serverName, serverPort: serverPort, quitChan: quitChan}
	glog.V(1).Infof("created worker %s", name)
	return worker, nil
}

func (this *WorkerApp) shouldExit() bool {
	select {
	case <-*this.quitChan:
		//glog.Infof("worker%d saw exit channel set", this.n)
		return true
	default:
		//glog.Infof("worker%d NO exit flag yet...", this.n)
		return false
	}
}

func (this *WorkerApp) Connect() error {
	distributor := fmt.Sprintf("%s:%d", this.serverName, this.serverPort)
	conn, err := rpc.Dial("tcp", distributor)
	if err != nil {
		glog.Errorf("can't connect to distributor %s: %s", distributor, err)
		return err
	}
	glog.Infof("connected to %s", distributor)
	this.conn = conn
	return nil
}

func (this *WorkerApp) ConnectRetry() error {
	backoff := []time.Duration{1, 2, 5, 10, 20, 30}
	ctr := 0
	for {
		if this.shouldExit() {
			return nil
		}
		err := this.Connect()
		if err == nil {
			return nil
		}
		if shouldReconnect(err) {
			if ctr >= len(backoff) {
				ctr = len(backoff) - 1
			}
			// important: use this.Sleep not time.Sleep
			this.Sleep(backoff[ctr] * time.Second)
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

func (this *WorkerApp) GetNextTask() *midtown.WorkerTask {
	var task midtown.WorkerTask
	err := this.conn.Call("WorkerApi.GetWorkerTask", this.name, &task)
	if err != nil {
		if shouldReconnect(err) {
			this.ConnectRetry()
			// we will be called again my main loop on next iteration
			return nil
		}
		glog.Errorf("failed to get task: %s", err)
		return nil
	}
	if task.Job == "" {
		glog.V(2).Infof("no task")
		return nil
	} else {
		glog.Infof("got task %s:%d", task.Job, task.Seq)
		if glog.V(1) {
			glog.Info(spew.Sdump(task))
		}
		return &task
	}
}

func (this *WorkerApp) SetTaskDone(taskResult *midtown.TaskResult) error {
	var ok bool
RETRY:
	err := this.conn.Call("WorkerApi.SetTaskDone", taskResult, &ok)
	if err != nil {
		if shouldReconnect(err) {
			this.ConnectRetry()
			if this.shouldExit() {
				return nil
			} else {
				goto RETRY
			}
		}
		glog.Errorf("failed to set task done: %s", err)
		return nil
	}
	glog.V(1).Infof("set task %s:%d done", taskResult.Job, taskResult.Seq)
	return nil
}

func (this *WorkerApp) RunTask(task *midtown.WorkerTask) *midtown.TaskResult {
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

func (this *WorkerApp) Sleep(dur time.Duration) {
	const INTERVAL = 500 * time.Millisecond
	if dur < INTERVAL {
		time.Sleep(dur)
	} else {
		for ; dur > 0; dur -= INTERVAL {
			time.Sleep(INTERVAL)
			if this.shouldExit() {
				return
			}
		}
	}
}

func WorkerMainLoop(n int, quitChan *chan bool, wg *sync.WaitGroup, svr string, port int) {
	defer wg.Done()

	worker, err := NewWorkerApp(n, svr, port, quitChan)
	if err != nil {
		glog.Errorf("can't create worker app: %s", err)
		return
	}

	worker.ConnectRetry()

	for {
		if worker.shouldExit() {
			return
		}

		task := worker.GetNextTask()
		if task != nil {
			// TODO: distinguish between failure to run task and an error returned by
			// the task subproc?
			taskResult := worker.RunTask(task)
			worker.SetTaskDone(taskResult) // TODO: err handling

		} else {
			// TODO: orderly shutdown, signal handler etc.
			// TODO: exponential backoff
			worker.Sleep(1 * time.Second)
		}
	}
}

var sigChan = make(chan os.Signal, 1)
var quitChan = make(chan bool, 1)

func main() {
	var svr = flag.String("svr", "localhost", "distributor hostname")
	var basePort = flag.Int("port", 6877, "distributor base port")
	var n = flag.Int("n", 1, "number of worker goroutines")
	flag.Parse()

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		close(quitChan)
	}()

	for i := 0; i < *n; i++ {
		wg.Add(1)
		go WorkerMainLoop(i, &quitChan, wg, *svr, *basePort+2)
	}

	wg.Wait()
	glog.Info("midtownw stopped")
	glog.Flush()
}
