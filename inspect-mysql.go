//Copyright (c) 2014 Square, Inc

//Launches metrics collector for mysql databases
//

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/measure/metrics"
	"github.com/measure/mysql/dbstat"
	"github.com/measure/mysql/tablestat"
)

func main() {
	var user, password, address, conf, group, form string
	var stepSec int
	var servermode, human, loop bool

	m := metrics.NewMetricContext("system")

	flag.StringVar(&user, "u", "root", "user using database")
	flag.StringVar(&password, "p", "", "password for database")
	flag.BoolVar(&servermode, "server", false,
		"Runs continously and exposes metrics as JSON on HTTP")
	flag.StringVar(&address, "address", ":12345",
		"address to listen on for http if running in server mode")
	flag.IntVar(&stepSec, "step", 2, "metrics are collected every step seconds")
	flag.StringVar(&conf, "conf", "/root/.my.cnf", "configuration file")
	flag.StringVar(&form, "form", "graphite", "output format of metrics to stdout")
	flag.BoolVar(&human, "h", false,
		"Makes output in MB for human readable sizes")
	flag.StringVar(&group, "group", "", "group of metrics to collect")
	flag.BoolVar(&loop, "loop", false,
		"loop on collecting metrics when specifying group")
	flag.Parse()

	if servermode {
		go func() {
			http.HandleFunc("/api/v1/metrics.json/", m.HttpJsonHandler)
			log.Fatal(http.ListenAndServe(address, nil))
		}()
	}
	step := time.Millisecond * time.Duration(stepSec) * 1000

	//if a group is defined, run metrics collections for just that group
	if group != "" {
		//initialize metrics collectors to not loop and collect
		sqlstat, err := dbstat.New(m, user, password, conf)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		sqlstatTables, err := tablestat.New(m, user, password, conf)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		//call the specific method name for the wanted group of metrics
		sqlstat.CallByMethodName(group)
		sqlstatTables.CallByMethodName(group)
		outputMetrics(sqlstat, sqlstatTables, m, form)
		//if metrics collection for this group is wanted on a loop,
		if loop {
			ticker := time.NewTicker(step)
			for _ = range ticker.C {
				sqlstat.CallByMethodName(group)
				sqlstatTables.CallByMethodName(group)
				outputMetrics(sqlstat, sqlstatTables, m, form)
			}
		}
		//if no group is specified, just run all metrics collections on a loop
	} else {
		sqlstat, err := dbstat.New(m, user, password, conf)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		sqlstatTables, err := tablestat.New(m, user, password, conf)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		sqlstat.Collect()
		sqlstatTables.Collect()
		time.Sleep(time.Second)
		outputMetrics(sqlstat, sqlstatTables, m, form)
		if loop {
			ticker := time.NewTicker(step)
			for _ = range ticker.C {
				sqlstat.Collect()
				sqlstatTables.Collect()
				outputMetrics(sqlstat, sqlstatTables, m, form)
			}
		}
	}
}

//output metrics in specific output format
func outputMetrics(d *dbstat.MysqlStat, t *tablestat.MysqlStatTables, m *metrics.MetricContext, form string) {
	//print out json packages
	if form == "json" {
		m.EncodeJSON(os.Stdout)
	}
	//print out in graphite form:
	//<metric_name> <metric_value>
	if form == "graphite" {
		d.FormatGraphite(os.Stdout)
		t.FormatGraphite(os.Stdout)
	}
}
