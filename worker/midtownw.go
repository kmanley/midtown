package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/kmanley/midtown"
	_ "io/ioutil"
	"net/rpc"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Worker struct {
	name string
	conn *rpc.Client
}

func (this *Worker) GetNextTask() *midtown.WorkerTask {
	var task midtown.WorkerTask
	err := this.conn.Call("WorkerApi.GetWorkerTask", this.name, &task)
	if err != nil {
		glog.Errorf("failed to get task: %s", err)
		return nil
	}
	if task.Job == "" {
		glog.V(1).Infof("no task")
		return nil
	} else {
		if glog.V(1) {
			glog.Infof("got task %s:%d", task.Job, task.Seq)
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

	/*
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			glog.Errorf("failed to connect stdout pipe: %v", err)
			taskResult.Error = err
			return taskResult
		}
	*/

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	/*
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			glog.Errorf("failed to get stdout pipe: %v", err)
			taskResult.Error = err
			return taskResult
		}
	*/

	fmt.Println(cmd) // TODO:

	if err := cmd.Run(); err != nil {
		glog.Errorf("failed to run cmd: %v", err)
		taskResult.Error = err
		return taskResult
	}

	/*
		stdoutBytes, err := ioutil.ReadAll(stdout)
		if err != nil {
			glog.Errorf("failed to read stdout: %v", err)
			taskResult.Error = err
			return taskResult
		}

		if err := cmd.Wait(); err != nil {
			glog.Errorf("wait failed: %v", err)
			taskResult.Error = err
			return taskResult
		}
	*/

	fmt.Println("stdout: ", stdout.String())
	fmt.Println("stderr: ", stderr.String())

	/*
		xyz, err := cmd.Output()
		if err != nil {
			glog.Errorf("cmd.Output failed: %v", err)
		}
		fmt.Println("output:", xyz)
	*/

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
	flag.Parse()

	dest := "127.0.0.1" // TODO: cmdline
	port := 9998        // TODO: cmdline
	distributor := fmt.Sprintf("%s:%d", dest, port)
	conn, err := rpc.Dial("tcp", distributor)
	if err != nil {
		// TODO:
		glog.Errorf("can't connect to distributor: %s", err)
		os.Exit(1)
	}

	// TODO: allow overriding name on cmdline?
	hostname, err := os.Hostname()
	if err != nil {
		glog.Error("can't get hostname: %s", err)
		os.Exit(1)
	}
	name := fmt.Sprintf("%s:%d", hostname, os.Getpid())
	worker := &Worker{name, conn}
	glog.Infof("worker %s starting", name)

	//task := &midtown.WorkerTask{Cmd: "python", Args: []string{"-c", "print 12345"}}
	//task := &midtown.WorkerTask{Cmd: "python", Args: []string{"test.py"}}
	//task := &midtown.WorkerTask{Cmd: "echo", Args: []string{"wtf?"}}
	/*
		task := &midtown.WorkerTask{
			Job:  "12345",
			Seq:  0,
			Cmd:  "python",
			Args: []string{"-c", "import json,sys;job,seq,data,ctx=json.load(sys.stdin);json.dump(data*2,sys.stdout)"},
			Data: "[1,2,3]",
			Ctx:  &midtown.Context{"foo": "bar"},
		}
		x := worker.RunTask(task)
		fmt.Println("-------------------")
		fmt.Printf("%#v", x)
	*/

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
