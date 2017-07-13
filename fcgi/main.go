package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"runtime"

	"github.com/Eun/domwatch/fcgi/api1"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var (
	local = flag.String("local", "", "serve as webserver, example: 0.0.0.0:8000")
	tcp   = flag.String("tcp", "", "serve as FCGI via TCP, example: 0.0.0.0:8000")
	unix  = flag.String("unix", "", "serve as FCGI via UNIX socket, example: /tmp/myprogram.sock")
)

var router *mux.Router

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {

	flag.Parse()

	var err error
	var db *gorm.DB
	var config *Config

	config, err = NewConfigFromFile("config.json")
	if err != nil {
		log.Println(err)
		config, err = NewConfigFromMap(nil)
		if err != nil {
			log.Fatalln(err)
		}
	}

	db, err = gorm.Open(*config.Database.Provider, *config.Database.Database)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	router = mux.NewRouter()

	var api *api1.API

	api, err = api1.NewApi(&config.Config, db, router.PathPrefix("/api1").Subrouter())
	if err != nil {
		log.Fatalln(err)
	}
	api.Run()
	defer api.Close()

	router.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("html/"))))

	if *local != "" { // Run as a local web server
		err = http.ListenAndServe(*local, router)
	} else if *tcp != "" { // Run as FCGI via TCP
		listener, err := net.Listen("tcp", *tcp)
		if err != nil {
			log.Fatal(err)
		}
		defer listener.Close()

		err = fcgi.Serve(listener, router)
	} else if *unix != "" { // Run as FCGI via UNIX socket
		listener, err := net.Listen("unix", *unix)
		if err != nil {
			log.Fatal(err)
		}
		defer listener.Close()

		err = fcgi.Serve(listener, router)
	} else { // Run as FCGI via standard I/O
		err = fcgi.Serve(nil, router)
	}
	if err != nil {
		log.Fatal(err)
	}
}
