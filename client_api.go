package midtown

import (
	"fmt"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"net/http"
	"sync"
)

type ClientApi struct {
	model *Model
}

func NewClientApi(model *Model) *ClientApi {
	return &ClientApi{model}
}

// POST /jobs
func (this *ClientApi) CreateJob(w rest.ResponseWriter, req *rest.Request) {
	var jobDef *JobDefinition
	err := req.DecodeJsonPayload(jobDef)
	if err != nil {
		rest.Error(w, "failed to decode job definition", 503) // TODO: error code?
		return
	}
	spew.Dump(jobDef) // TODO:remove
	jobId, err := this.model.CreateJob(jobDef)
	if err != nil {
		rest.Error(w, err.Error(), 503) // TODO: error code?
		return
	}
	w.WriteJson(&jobId)
}

// GET /result/:jobid
func (this *ClientApi) GetJobResult(w rest.ResponseWriter, req *rest.Request) {
	jobid := req.PathParam("jobid")
	res, err := this.model.GetJobResult(JobID(jobid))
	if err != nil {
		rest.Error(w, err.Error(), 503) // TODO: error code?
		return
	}
	w.WriteJson(&res)
}

// GET /job/:jobid
func (this *ClientApi) GetJob(w rest.ResponseWriter, req *rest.Request) {
	jobid := req.PathParam("jobid")
	res, err := this.model.GetJob(JobID(jobid))
	if err != nil {
		rest.Error(w, err.Error(), 503) // TODO: error code?
		return
	}
	w.WriteJson(&res)
}

// GET /workers
func (this *ClientApi) GetWorkers(w rest.ResponseWriter, req *rest.Request) {
	res, err := this.model.GetWorkers()
	if err != nil {
		rest.Error(w, err.Error(), 503) // TODO: error code?
		return
	}
	w.WriteJson(&res)
}

func StartClientApi(model *Model, port int, wg *sync.WaitGroup) {
	api := NewClientApi(model)

	handler := rest.ResourceHandler{}
	handler.SetRoutes(
		&rest.Route{"POST", "/jobs", api.CreateJob},
		&rest.Route{"GET", "/result/:jobid", api.GetJobResult},
		&rest.Route{"GET", "/job/:jobid", api.GetJob},
		&rest.Route{"GET", "/workers", api.GetWorkers},
	)
	glog.Infof("serving client API on %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), &handler)

	wg.Done() // TODO: make sure we do orderly shutdown and call this in all cases
}
