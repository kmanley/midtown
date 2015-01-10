package midtown

import (
	"fmt"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	_ "io/ioutil"
	"net/http"
	"sync"
)

type ClientApi struct {
	model *Model
}

func NewClientApi(model *Model) *ClientApi {
	return &ClientApi{model}
}

// TODO: don't allow posting job with no data! currently this can be done via python if
// only Cmd is sent
func (this *ClientApi) CreateJob(w rest.ResponseWriter, req *rest.Request) {
	jobDef := &JobDefinition{}

	/*
		content, err := ioutil.ReadAll(req.Body)
		fmt.Println("*******************************")
		fmt.Println(string(content))
		fmt.Println("*******************************")
	*/

	err := req.DecodeJsonPayload(jobDef)
	if err != nil {
		rest.Error(w, fmt.Sprintf("failed to decode job definition: %s", err.Error()), 503) // TODO: error code?
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
		glog.Infof("err=%#v", err)
		_, ok := err.(*ErrJobNotFinished)
		glog.Infof("ok=%#v", ok)
		if ok {
			rest.Error(w, "still working", 102) // TODO: 102 Processing (used by WebDAV)
		} else {
			rest.Error(w, err.Error(), 500) // TODO: error code?
		}
		return
	}
	w.WriteJson(&res)
}

// GET /job/:jobid
func (this *ClientApi) GetJob(w rest.ResponseWriter, req *rest.Request) {
	jobid := req.PathParam("jobid")
	res, err := this.model.GetJob(JobID(jobid))
	if err != nil {
		rest.Error(w, err.Error(), 500) // TODO: error code?
		return
	}
	w.WriteJson(&res)
}

// GET /workers
func (this *ClientApi) GetWorkers(w rest.ResponseWriter, req *rest.Request) {
	res, err := this.model.GetWorkers()
	if err != nil {
		rest.Error(w, err.Error(), 500) // TODO: error code?
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
