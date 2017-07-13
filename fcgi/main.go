package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
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

	var logger *log.Logger
	if config.LogFile == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	} else {
		var logFile *os.File
		logFile, err = os.OpenFile(*config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			log.Fatalln(err)
		}
		defer logFile.Close()
		logger = log.New(logFile, "", log.LstdFlags)

	}

	if *config.Database.Provider == "mssql" {
		db, err = gorm.Open("mssql", fmt.Sprintf("sqlserver://%s:%s@%s:1433?database=%s", *config.Database.User, *config.Database.Password, *config.Database.Host, *config.Database.Database))
	} else if *config.Database.Provider == "mysql" {
		db, err = gorm.Open("mysql", fmt.Sprintf("%s:%s@/%s?charset=utf8&parseTime=True&loc=%s", *config.Database.User, *config.Database.Password, *config.Database.Database, *config.Database.Host))
	} else if *config.Database.Provider == "postgres" {
		db, err = gorm.Open("postgres", fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", *config.Database.Host, *config.Database.User, *config.Database.Database, *config.Database.Password))
	} else if *config.Database.Provider == "sqlite3" {
		db, err = gorm.Open(*config.Database.Provider, *config.Database.Database)
	} else {
		logger.Fatalf("Unknown provider '%s'\n", *config.Database.Provider)
	}

	if err != nil {
		logger.Fatalln(err)
	}
	defer db.Close()

	router = mux.NewRouter()

	var api *api1.API

	api, err = api1.NewApi(&config.Config, db, router.PathPrefix("/api1").Subrouter(), logger)
	if err != nil {
		logger.Fatalln(err)
	}
	api.Run()
	defer api.Close()

	router.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("html/"))))

	if *local != "" { // Run as a local web server
		err = http.ListenAndServe(*local, router)
	} else if *tcp != "" { // Run as FCGI via TCP
		listener, err := net.Listen("tcp", *tcp)
		if err != nil {
			logger.Fatal(err)
		}
		defer listener.Close()

		err = fcgi.Serve(listener, router)
	} else if *unix != "" { // Run as FCGI via UNIX socket
		listener, err := net.Listen("unix", *unix)
		if err != nil {
			logger.Fatal(err)
		}
		defer listener.Close()

		err = fcgi.Serve(listener, router)
	} else { // Run as FCGI via standard I/O
		err = fcgi.Serve(nil, router)
	}
	if err != nil {
		logger.Fatal(err)
	}
}
