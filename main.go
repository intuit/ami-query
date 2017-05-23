// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/VividCortex/godaemon"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/intuit/ami-query/amicache"
	"github.com/intuit/ami-query/api"
	"github.com/intuit/ami-query/api/v1"
)

const version = "1.1.0"
const usage = `usage: %s [--version] [--help]

  --help     Display this help message and exit
  --version  Print version and exit
`

var (
	// Command line arguments
	printVersion = flag.Bool("version", false, "")
	daemonize    = flag.Bool("daemonize", false, "")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		os.Exit(2)
	}

	flag.Parse()

	if *printVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	appLogger, err := setLogger(os.Getenv("AMIQUERY_APP_LOGFILE"))
	if err != nil {
		log.Fatalln("Unable to set application logging output:", err)
	}

	log.SetOutput(appLogger)

	httpLogger, err := setLogger(os.Getenv("AMIQUERY_HTTP_LOGFILE"))
	if err != nil {
		log.Fatalln("Unable to set HTTP logging output:", err)
	}

	cfg, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Loaded configuration")

	cacheMgr, err := amicache.NewManager(cfg.Manager,
		amicache.TTL(cfg.CacheTTL),
		amicache.Regions(cfg.Regions...),
		amicache.OwnerIDs(cfg.OwnerIDs...),
		amicache.AssumeRole(cfg.RoleARN),
		amicache.HTTPClient(&http.Client{
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				MaxIdleConnsPerHost:   1,
				DisableKeepAlives:     false,
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
			},
		}),
	)

	if err != nil {
		log.Fatal(err)
	}

	if *daemonize {
		log.Println("Daemonizing process...")
		godaemon.MakeDaemon(&godaemon.DaemonAttr{})
	}

	cacheMgr.Start()
	defer cacheMgr.Stop()

	// Create a context and add the cache manager
	ctx := context.Background()
	ctx = context.WithValue(ctx, api.CacheManagerKey, cacheMgr)

	// Version 1 of the REST API
	v1 := &api.ContextAdapter{
		Context: ctx,
		Handler: api.ContextHandlerFunc(v1.Handler),
	}

	router := mux.NewRouter()

	// Add the version 1 handler and set it as the default (Accept: */*)
	router.Handle("/amis", handlers.CombinedLoggingHandler(httpLogger, v1)).
		HeadersRegexp("Accept", `(application/vnd\.ami-query-v1\+json|\*/\*)`).
		Methods("GET")

	http.Handle("/", router)

	if cfg.SSLCert != "" && cfg.SSLKey != "" {
		err = http.ListenAndServeTLS(cfg.ListenAddr, cfg.SSLCert, cfg.SSLKey, nil)
	} else {
		err = http.ListenAndServe(cfg.ListenAddr, nil)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func setLogger(file string) (io.Writer, error) {
	logger := os.Stderr
	if file != "" {
		var err error
		if logger, err = os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
			return nil, err
		}
	}
	return logger, nil
}
