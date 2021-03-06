package midtown

import (
	"errors"
	"fmt"
	_ "github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/kmanley/midtown/common"
	"github.com/kmanley/midtown/templates"
	"github.com/mailgun/manners"
	_ "io/ioutil"
	"net/http"
	"strconv"
	//"sync"
	"time"
)

type WebApi struct {
	model *Model
	//wg    *sync.WaitGroup
	svr *manners.GracefulServer
}

var webApi *WebApi

func (this *WebApi) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	http.Redirect(w, r, "/jobs/active", 302)
}

func (this *WebApi) GetActiveJobs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	offset := 0
	count := 0
	offset, _ = strconv.Atoi(r.FormValue("offset"))
	count, _ = strconv.Atoi(r.FormValue("count"))

	summList, err := this.model.GetActiveJobsSummary(offset, count)
	if err != nil {
		templates.Error(w, err)
	}

	//spew.Dump(summList) // TODO:

	templates.ActiveJobs(w, summList)

	//var jobs JobList
	//err := this.model.loadActiveJobs()
	//func (this *Model) loadActiveJobs(tx *bolt.Tx, jobs *JobList) error {
	/*
		if err != nil {
			rest.Error(w, err.Error(), 500) // TODO: error code?
			return
		}
		w.WriteJson(&res)
	*/
}

func (this *WebApi) GetCompletedJobs(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	dt := params.ByName("dt")
	offset := 0
	count := 0
	offset, _ = strconv.Atoi(r.FormValue("offset"))
	count, _ = strconv.Atoi(r.FormValue("count"))

	summList, err := this.model.GetCompletedJobsSummary(dt, offset, count)
	if err != nil {
		templates.Error(w, err)
	}

	//spew.Dump(summList) // TODO:
	templates.CompletedJobs(w, dt, summList)
}

func (this *WebApi) GetJob(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	jobid := params.ByName("jobid")

	job, err := this.model.GetJobDetails(common.JobID(jobid))
	if err != nil {
		templates.Error(w, err)
		return
	}

	if job == nil {
		templates.Error(w, errors.New("can't find job "+jobid))
		return
	}

	//spew.Dump(summList) // TODO:
	templates.Job(w, job)
}

func (this *WebApi) GetWorkers(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	offset := 0
	count := 0
	offset, _ = strconv.Atoi(r.FormValue("offset"))
	count, _ = strconv.Atoi(r.FormValue("count"))

	workers, err := this.model.GetWorkers(offset, count)
	if err != nil {
		templates.Error(w, err)
	}

	//spew.Dump(summList) // TODO:
	templates.Workers(w, workers)
}

func (this *WebApi) GetHandler() *httprouter.Router {
	handler := httprouter.New()
	handler.GET("/", this.Index)
	handler.GET("/jobs/active", this.GetActiveJobs)
	handler.GET("/jobs/completed/:dt", this.GetCompletedJobs)
	handler.GET("/job/:jobid", this.GetJob)
	handler.GET("/workers", this.GetWorkers)
	return handler
}

func StartWebApi(model *Model, port int) {
	webApi = &WebApi{model, nil}
	handler := webApi.GetHandler()

	webApi.svr = manners.NewWithServer(&http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        handler,
		ReadTimeout:    10 * time.Second, // TODO:?
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // TODO:?
	})

	glog.Infof("serving web API on %d", port)
	// this call blocks till someone calls StopWebApi
	webApi.svr.ListenAndServe()
	glog.Info("web server stopped")
}

func StopWebApi() {
	glog.Infof("stopping web server...")
	webApi.svr.Close()

}
