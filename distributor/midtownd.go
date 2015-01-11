package main

import (
	"flag"
	_ "github.com/golang/glog"
	"github.com/kmanley/midtown"
	"sync"
)

type Distributor struct {
	model *midtown.Model
	wg    *sync.WaitGroup
}

var App Distributor

func main() {

	flag.Parse()
	//dbname := "midtownd.db" // TODO: cmdline
	dbname := "/tmp/midtown_test.db" // TODO:
	clientApiPort := 9997
	workerApiPort := 9998 // TODO: cmdline
	webApiPort := 9999
	App.model = &midtown.Model{}
	App.model.Init(dbname, 0600)
	App.wg = &sync.WaitGroup{}

	App.wg.Add(1)
	go midtown.StartWorkerApi(App.model, workerApiPort, App.wg)

	App.wg.Add(1)
	go midtown.StartClientApi(App.model, clientApiPort, App.wg)

	App.wg.Add(1)
	go midtown.StartWebApi(App.model, webApiPort, App.wg)

	App.wg.Wait()

	//startWebServer()
	//startClientApi()
	//startWorkerApi()

	// defer db.Close() TODO: close db in shutdown handler

}
