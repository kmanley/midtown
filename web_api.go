package midtown

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/kmanley/midtown/templates"
	_ "io/ioutil"
	"net/http"
	"strconv"
	"sync"
)

type WebApi struct {
	model *Model
}

func NewWebApi(model *Model) *WebApi {
	return &WebApi{model}
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

	spew.Dump(summList) // TODO:

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

func StartWebApi(model *Model, port int, wg *sync.WaitGroup) {
	api := NewWebApi(model)
	handler := httprouter.New()
	handler.GET("/jobs/active", api.GetActiveJobs)

	glog.Infof("serving web API on %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), handler)

	wg.Done() // TODO: make sure we do orderly shutdown and call this in all cases
}
