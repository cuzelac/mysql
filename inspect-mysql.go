//Copyright (c) 2014 Square, Inc
//Launches metrics collector for mysql databases

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/measure/metrics"
	"github.com/measure/mysql/dbstat"
	"github.com/measure/mysql/tablestat"
	//	"github.com/measure/mysql/tools"
)

func main() {
	var user, password, address, conf, group string
	var stepSec int
	var servermode, human bool

	m := metrics.NewMetricContext("system")

	flag.StringVar(&user, "u", "root", "user using database")
	flag.StringVar(&password, "p", "", "password for database")
	flag.BoolVar(&servermode, "server", false, "Runs continously and exposes metrics as JSON on HTTP")
	flag.StringVar(&address, "address", ":12345", "address to listen on for http if running in server mode")
	flag.IntVar(&stepSec, "step", 2, "metrics are collected every step seconds")
	flag.StringVar(&conf, "conf", "/root/.my.cnf", "configuration file")
	flag.BoolVar(&human, "h", false, "Makes output in MB for human readable sizes")
	flag.StringVar(&group, "group", "", "group of metrics to collect")
	flag.Parse()

	if servermode {
		go func() {
			http.HandleFunc("/api/v1/metrics.json/", m.HttpJsonHandler)
			log.Fatal(http.ListenAndServe(address, nil))
		}()
	}
	step := time.Millisecond * time.Duration(stepSec) * 1000

	if group != "" {
		sqlstat, err := dbstat.New(m, step, user, password, conf, false)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		//	sqlstatTables, err := tablestat.New(m, step, user, password, conf)
		//	if err != nil {
		//		fmt.Println(err)
		//		os.Exit(1)
		//	}
		err = sqlstat.CallByMethodName(group)
		fmt.Println(err)
		//  err = sqlstatTables.CallByMethodName(group)
		//  fmt.Println(err)
		b1 := sqlstat.GetNonemptyMetrics()
		for _, b := range b1 {
			fmt.Println(b)
		}
		//  b2 := sqlstatTables.GetNonemptyMetrics()
		//  for _, b := range b2 {
		//    fmt.Println(b)
		//  }
	} else {
		sqlstat, err := dbstat.New(m, step, user, password, conf, true)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		sqlstatTables, err := tablestat.New(m, step, user, password, conf)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		ticker := time.NewTicker(step * 2)
		for _ = range ticker.C {
			//Print stats here, more stats than printed are actually collected
			fmt.Println("--------------------------")
			fmt.Println("Version: " + strconv.FormatFloat(sqlstat.Metrics.Version.Get(), 'f', -1, 64))
			fmt.Println("Queries made: " + strconv.Itoa(int(sqlstat.Metrics.Queries.Get())))
			fmt.Println("Uptime: " + strconv.Itoa(int(sqlstat.Metrics.Uptime.Get())))
			fmt.Println("Database sizes: ")
			for dbname, db := range sqlstatTables.DBs {
				size := db.Metrics.SizeBytes.Get()
				units := " B"
				if human {
					size /= (1024 * 1024)
					units = " GB"
				}
				fmt.Println("    " + dbname + ": " + strconv.FormatFloat(size, 'f', 2, 64) + units)
			}
		}
	}

}
