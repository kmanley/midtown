package main

import (
	"flag"
	_ "fmt"
	"github.com/golang/glog"
	"github.com/kmanley/midtown"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Distributor struct {
	model *midtown.Model
	wg    *sync.WaitGroup
}

var quitChan = make(chan os.Signal, 1)
var App Distributor

func backgroundLoop(d *Distributor) {
	defer d.wg.Done()
	for {
		select {
		case <-quitChan:
			go midtown.StopWebApi()
			go midtown.StopClientApi()
			go midtown.StopWorkerApi()
			goto END
		default:
			glog.Info("dummy loop")
			time.Sleep(2.0 * time.Second)
		}
	}
END:
	glog.Info("background loop exiting")
}

func main() {

	var basePort = flag.Int("port", 6877, "base port number")
	var dbName = flag.String("db", ".midtownd.db", "database filename")
	flag.Parse()

	webApiPort := *basePort
	clientApiPort := webApiPort + 1
	workerApiPort := clientApiPort + 1

	App.model = &midtown.Model{}
	App.model.Init(*dbName, 0600)
	App.wg = &sync.WaitGroup{}

	App.wg.Add(1)
	go func() {
		defer App.wg.Done()
		midtown.StartWebApi(App.model, webApiPort)
	}()

	App.wg.Add(1)
	go func() {
		defer App.wg.Done()
		midtown.StartClientApi(App.model, clientApiPort)
	}()

	App.wg.Add(1)
	go func() {
		defer App.wg.Done()
		midtown.StartWorkerApi(App.model, workerApiPort)
	}()

	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGTERM)

	App.wg.Add(1)
	go backgroundLoop(&App)
	App.wg.Wait()

	glog.Info("midtownd stopped")

	// defer db.Close() TODO: close db in shutdown handler
}
