package midtown

import (
	"fmt"
	_ "github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/kmanley/midtown/templates"
	"github.com/stretchr/graceful"
	_ "io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type WebApi struct {
	model *Model
	wg    *sync.WaitGroup
	svr   *graceful.Server
}

var api *WebApi

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

func StartWebApi(model *Model, port int, wg *sync.WaitGroup) {
	api = &WebApi{model, wg, nil}
	handler := httprouter.New()
	handler.GET("/jobs/active", api.GetActiveJobs)
	handler.GET("/jobs/completed/:dt", api.GetCompletedJobs)

	glog.Infof("serving web API on %d", port)
	//http.ListenAndServe(fmt.Sprintf(":%d", port), handler)

	api.svr = &graceful.Server{
		Timeout: 5 * time.Second,
		Server:  &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: handler},
	}
	api.svr.ListenAndServe()
}

func StopWebApi() {
	glog.Infof("stopping web API...")
	api.svr.Stop(5 * time.Second)
	<-api.svr.StopChan()
	api.wg.Done()
}
