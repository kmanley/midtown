package midtown

import (
	"errors"
	"fmt"
	"os"
	//	"github.com/kmanley/golang-grid"
	"github.com/davecgh/go-spew/spew"
	"testing"
)

const filename = "/tmp/midtown_test.db"

func init() {
	os.Remove(filename)
}

func getModel() *Model {
	model := &Model{}
	model.Init(filename, 0600)
	return model
}

func TestNewJobId(t *testing.T) {
	model := getModel()
	defer model.Close()
	job1 := model.NewJobID()
	job2 := model.NewJobID()
	//fmt.Println(job1)
	//fmt.Println(job2)
	if job1 == job2 {
		t.Errorf("got identical job ids %s and %s", job1, job2)
	}
}

func _TestJobFail(t *testing.T) {
	model := getModel()
	defer model.Close()

	data := []interface{}{1, 3, 5, 7, 9}
	ctx := &Context{"foo": "bar"}
	ctrl := &JobControl{MaxConcurrency: 20, Priority: 8}
	jobdef := &JobDefinition{Cmd: "python.exe doit.py", Data: data,
		Description: "my first job", Ctx: ctx, Ctrl: ctrl}
	jobid, err := model.CreateJob(jobdef)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("created job", jobid)

	model.Close()
	model = getModel()

	job, err := model.GetJob(jobid)
	if err != nil {
		t.Error(err)
	}

	spew.Dump(job)

	fmt.Println("------------------------------------------------")

	workerTask1, err := model.GetWorkerTask("worker1")
	if err != nil {
		t.Error(err)
	}
	spew.Dump(workerTask1)

	workerTask2, err := model.GetWorkerTask("worker2")
	if err != nil {
		t.Error(err)
	}
	spew.Dump(workerTask2)

	fmt.Println("*************************************************")

	model.Close()
	model = getModel()
	job, err = model.GetJob(jobid)
	if err != nil {
		t.Error(err)
	}
	spew.Dump(job)

	fmt.Println("**************************************************")
	workers, _ := model.GetWorkers()
	spew.Dump(workers)

	model.SetTaskDone("worker2", workerTask2.Job, workerTask2.Seq, 200, "stdout2", "stderr2", nil)
	model.SetTaskDone("worker1", workerTask1.Job, workerTask1.Seq, 100, "stdout1", "stderr1", errors.New("oh dear"))

	fmt.Println("*** after settaskdone ***********************************************")
	model.Close()
	model = getModel()
	job, err = model.GetJob(jobid)
	if err == nil {
		t.Error("expected to not find job in active list")
	}

	fmt.Println("*** active job wtf")
	spew.Dump(job)

	fmt.Println("*** completed job")
	job, err = model.GetJob(jobid)
	if err != nil {
		t.Error(err)
	}
	spew.Dump(job)

}

func TestJobOK(t *testing.T) {
	model := getModel()
	defer model.Close()

	data := []interface{}{1, 3, 5, 7, 9}
	ctx := &Context{"foo": "bar"}
	ctrl := &JobControl{MaxConcurrency: 20, Priority: 8}
	jobdef := &JobDefinition{Cmd: "python.exe doit.py", Data: data,
		Description: "my first job", Ctx: ctx, Ctrl: ctrl}
	jobid, err := model.CreateJob(jobdef)
	fmt.Println("jobid", jobid)
	assertnil(t, err)

	model.Close()
	model = getModel()

	workerTask1, err := model.GetWorkerTask("worker1")
	workerTask2, err := model.GetWorkerTask("worker2")
	workerTask3, err := model.GetWorkerTask("worker3")
	workerTask4, err := model.GetWorkerTask("worker4")
	workerTask5, err := model.GetWorkerTask("worker5")

	model.Close()
	model = getModel()

	model.SetTaskDone("worker2", workerTask2.Job, workerTask2.Seq, 200, "stdout2", "stderr2", nil)
	res, err := model.GetJobResult(jobid)
	asserterr(t, err)

	model.SetTaskDone("worker1", workerTask1.Job, workerTask1.Seq, 100, "stdout1", "stderr1", nil)
	model.SetTaskDone("worker5", workerTask5.Job, workerTask5.Seq, 500, "stdout5", "stderr5", nil)
	model.SetTaskDone("worker3", workerTask3.Job, workerTask3.Seq, 300, "stdout3", "stderr3", nil)
	res, err = model.GetJobResult(jobid)
	asserterr(t, err)

	model.SetTaskDone("worker4", workerTask4.Job, workerTask4.Seq, 400, "stdout4", "stderr4", nil)

	res, err = model.GetJobResult(jobid)
	assertnil(t, err)
	spew.Dump(res)

}

/*
func TestCreateJobs(t *testing.T) {
	resetModel()
	data := []interface{}{1, 3, 5, 7, 9}
	ctx := &Context{"foo": "bar"}
	ctrl := &JobControl{MaxConcurrency: 20}
	const COUNT = 5
	for i := 0; i < COUNT; i++ {
		CreateJob(&JobDefinition{Cmd: "python.exe doit.py", Data: data,
			Description: fmt.Sprintf("job %d", i), Ctx: ctx,
			Ctrl: ctrl})
	}
	PrintStats()
	//sanityCheck()
	for i := 0; i < COUNT; i++ {

	}
}
*/

/*
func TestSimple(t *testing.T) {
	resetModel()
	jobID, _ := CreateJob(&JobDefinition{Cmd: "python.exe doit.py", Data: []interface{}{1, 3, 5, 7, 9},
		Description: "my first job", Ctx: &Context{"foo": "bar"},
		Ctrl: &JobControl{MaxConcurrency: 20}})
	PrintStats()
	t1 := GetWorkerTask("worker1")
	t2 := GetWorkerTask("worker2")
	PrintStats()
	//fmt.Println(t1, t2)
	t3 := GetWorkerTask("worker3")
	t4 := GetWorkerTask("worker4")
	PrintStats()
	t5 := GetWorkerTask("worker5")
	PrintStats()

	//fmt.Println(t1, t2, t3, t4, t5)

	SetTaskDone("worker1", t1.Job, t1.Seq, 10, "", "", nil)
	SetTaskDone("worker2", t2.Job, t2.Seq, 30, "", "", nil)
	SetTaskDone("worker3", t3.Job, t3.Seq, 50, "", "", nil)
	SetTaskDone("worker4", t4.Job, t4.Seq, 70, "", "", nil)
	SetTaskDone("worker5", t5.Job, t5.Seq, 90, "", "", nil)

	PrintStats()

	res, _ := GetJobResult(jobID)
	fmt.Println("RESULT: ", res)
}
*/
