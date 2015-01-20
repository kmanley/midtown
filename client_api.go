package midtown

import (
	"fmt"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/kmanley/midtown/common"
	"github.com/mailgun/manners"
	_ "io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type ClientApi struct {
	model *Model
	svr   *manners.GracefulServer
}

var clientApi *ClientApi

// TODO: don't allow posting job with no data! currently this can be done via python if
// only Cmd is sent
func (this *ClientApi) CreateJob(w rest.ResponseWriter, req *rest.Request) {
	jobDef := &common.JobDefinition{}

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
	res, err := this.model.GetJobResult(common.JobID(jobid))
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
	res, err := this.model.GetJobDetails(common.JobID(jobid))
	if err != nil {
		rest.Error(w, err.Error(), 500) // TODO: error code?
		return
	}
	w.WriteJson(&res)
}

// GET /workers
func (this *ClientApi) GetWorkers(w rest.ResponseWriter, r *rest.Request) {
	offset := 0
	count := 0
	offset, _ = strconv.Atoi(r.FormValue("offset"))
	count, _ = strconv.Atoi(r.FormValue("count"))
	res, err := this.model.GetWorkers(offset, count)
	if err != nil {
		rest.Error(w, err.Error(), 500) // TODO: error code?
		return
	}
	w.WriteJson(&res)
}

func (this *ClientApi) GetHandler() *rest.ResourceHandler {
	handler := rest.ResourceHandler{}
	handler.SetRoutes(
		&rest.Route{"POST", "/jobs", this.CreateJob},
		&rest.Route{"GET", "/result/:jobid", this.GetJobResult},
		&rest.Route{"GET", "/job/:jobid", this.GetJob},
		&rest.Route{"GET", "/workers", this.GetWorkers},
	)
	return &handler
}

func StartClientApi(model *Model, port int) {
	clientApi = &ClientApi{model, nil}

	handler := clientApi.GetHandler()

	clientApi.svr = manners.NewWithServer(&http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        handler,
		ReadTimeout:    10 * time.Second, // TODO:?
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // TODO:?
	})

	glog.Infof("serving client API on %d", port)
	// this line blocks till someone calls StopClientApi
	clientApi.svr.ListenAndServe()
	glog.Info("client API server stopped")

}

func StopClientApi() {
	glog.Infof("stopping client API server...")
	clientApi.svr.Close()
}
