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

//func (this *ClientApi) CreateJob(jobDef *JobDefinition) (JobID, error) {

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

func StartClientApi(model *Model, port int, wg *sync.WaitGroup) {
	api := NewClientApi(model)

	handler := rest.ResourceHandler{}
	handler.SetRoutes(
		&rest.Route{"POST", "/jobs", api.CreateJob},
	)
	glog.Infof("serving client API on %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), &handler)

	wg.Done() // TODO: make sure we do orderly shutdown and call this in all cases
}
