package main

import (
	"flag"
	"fmt"
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

func dummy(d *Distributor) {
	d.wg.Add(1)
	defer d.wg.Done()
	for {
		select {
		case <-quitChan:
			midtown.StopWebApi()
			goto END
		default:
			glog.Info("dummy loop")
			time.Sleep(2.0 * time.Second)
		}
	}
END:
	fmt.Println("dummy goroutine exiting")
}

func main() {

	var basePort = flag.Int("port", 6877, "base port number")
	var dbName = flag.String("db", ".midtownd.db", "database filename")
	flag.Parse()

	webApiPort := *basePort
	//clientApiPort := webApiPort + 1
	//workerApiPort := clientApiPort + 1

	App.model = &midtown.Model{}
	App.model.Init(*dbName, 0600)
	App.wg = &sync.WaitGroup{}

	App.wg.Add(1)
	go midtown.StartWebApi(App.model, webApiPort, App.wg)

	//App.wg.Add(1)
	//go midtown.StartClientApi(App.model, clientApiPort, App.wg)

	//App.wg.Add(1)
	//go midtown.StartWorkerApi(App.model, workerApiPort, App.wg)

	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGTERM)

	go dummy(&App)
	App.wg.Wait()

	// defer db.Close() TODO: close db in shutdown handler
}
