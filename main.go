// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/intuit/ami-query/amicache"
	"github.com/intuit/ami-query/api/query"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/oklog/oklog/pkg/group"
)

// HTTP client used for AWS API calls.
var httpClient = &http.Client{
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
}

func main() {
	// Command line arguments
	var (
		debug        = flag.Bool("debug", false, "Enable debug logging")
		printVersion = flag.Bool("version", false, "Prints the version and exits")
	)

	stdlog.SetFlags(0)
	flag.Parse()

	if *printVersion {
		stdlog.Print(version)
		os.Exit(0)
	}

	sess, err := session.NewSession(aws.NewConfig().WithHTTPClient(httpClient))
	if err != nil {
		stdlog.Fatalf("failed to create AWS session: %v", err)
	}

	cfg, err := NewConfig()
	if err != nil {
		stdlog.Fatalf("failed to parse configuration: %v", err)
	}

	appLogger, err := setLogger(cfg.AppLog)
	if err != nil {
		stdlog.Fatalf("failed to set application logging output: %v", err)
	}

	httpLogger, err := setLogger(cfg.HTTPLog)
	if err != nil {
		stdlog.Fatalf("failed to set HTTP logging output: %v", err)
	}

	// Setup go-kit logger.
	logger := log.NewLogfmtLogger(log.NewSyncWriter(appLogger))
	logger = log.With(logger, "ts", log.TimestampFormat(time.Now, "2006-01-02T15:04:05.000"))
	if *debug {
		logger = level.NewFilter(logger, level.AllowAll())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}

	// Redirect anything using stdlib log to go-kit log.
	stdlog.SetOutput(log.NewStdlibAdapter(logger))

	router := mux.NewRouter()
	server := http.Server{
		Addr:    cfg.ListenAddr,
		Handler: router,
		// TODO: make these configurable?
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	cache := amicache.New(
		sts.New(sess),
		cfg.RoleName,
		cfg.OwnerIDs,
		amicache.TagFilter(cfg.TagFilter),
		amicache.StateTag(cfg.StateTag),
		amicache.Regions(cfg.Regions...),
		amicache.TTL(cfg.CacheTTL),
		amicache.MaxConcurrentRequests(cfg.CacheMaxConcurrentRequests),
		amicache.MaxRequestRetries(cfg.CacheMaxRequestRetries),
		amicache.CollectLaunchPermissions(cfg.CollectLaunchPermissions),
		amicache.HTTPClient(httpClient),
		amicache.Logger(logger),
	)

	// Create the query endpoint and use Apache Combined log format.
	api := handlers.CombinedLoggingHandler(httpLogger, query.NewAPI(cache))

	// Optionally add CORS support for allowed Origins.
	if len(cfg.CorsAllowedOrigins) > 0 {
		api = handlers.CORS(
			handlers.AllowedMethods([]string{"GET"}),
			handlers.AllowedOrigins(cfg.CorsAllowedOrigins),
		)(api)
	}

	// Register the route.
	router.Handle(query.APIPathQuery, api).
		HeadersRegexp("Accept", `(application/vnd\.ami-query-v1\+json|\*/\*)`).
		Methods("GET")

	// Create a group and context for running the services.
	g := group.Group{}
	ctx, cancel := context.WithCancel(context.Background())

	// Used to block on waiting for the cache to warm.
	warmed := make(chan struct{})

	// Add the cache.
	g.Add(func() error {
		return cache.Run(ctx, warmed)
	}, func(error) {
		level.Info(logger).Log("msg", "stopping cache")
		cache.Stop()
		level.Info(logger).Log("msg", "cache stopped")
		cancel()
	})

	// Add the http server.
	g.Add(func() error {
		<-warmed // Wait for the cache
		if cfg.SSLCert != "" && cfg.SSLKey != "" {
			return server.ListenAndServeTLS(cfg.SSLCert, cfg.SSLKey)
		}
		return server.ListenAndServe()
	}, func(error) {
		// Wait for up to 1 minute for active connections to finish.
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		level.Info(logger).Log("msg", "gracefully shutting down http server")
		if err := server.Shutdown(ctx); err != nil {
			level.Error(logger).Log("msg", "http graceful shutdown failed", "error", err)
		}
		level.Info(logger).Log("msg", "http server shutdown")
	})

	// Add the signal trapper.
	g.Add(func() error {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(ch)
		return sigTrapper(ctx, ch)
	}, func(error) {
		cancel()
	})

	// Start the service.
	if err = g.Run(); err != nil {
		level.Info(logger).Log("service", err)
		os.Exit(1)
	}
}

// Signal trapper. It closes setup once it registers the signals.
func sigTrapper(ctx context.Context, ch <-chan os.Signal) error {
	select {
	case sig := <-ch:
		return fmt.Errorf("received signal %s", sig)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Creates a log file or returns os.Stderr if none is provided.
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
