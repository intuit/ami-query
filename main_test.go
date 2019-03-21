// Copyright 2017 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package main

import (
	"context"
	"os"
	"os/signal"
	"path"
	"syscall"
	"testing"
)

func TestSigTrapper(t *testing.T) {
	errCh := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())

	go func() { errCh <- sigTrapper(ctx, nil) }()
	cancel()

	if want, got := context.Canceled, <-errCh; want != got {
		t.Errorf("want: %v, got: %v", want, got)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)

	go func() { errCh <- sigTrapper(context.Background(), sigCh) }()

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}

	err := <-errCh

	if want, got := "received signal interrupt", err.Error(); want != got {
		t.Errorf("want: %s, got: %s", want, got)
	}
}

func TestSetLoggerStderr(t *testing.T) {
	logger, err := setLogger("")
	if err != nil {
		t.Fatal(err)
	}
	if logger != os.Stderr {
		t.Errorf("want: os.Stderr, got: %T", logger)
	}
}

func TestSetLoggerFile(t *testing.T) {
	file := path.Join(os.TempDir(), "ami-query-test.txt")
	if _, err := setLogger(file); err != nil {
		t.Fatal(err)
	}
	os.Remove(file)
}

func TestSetLoggerBadFile(t *testing.T) {
	_, err := setLogger("/F0o/b4r.txt")
	if want, got := "open /F0o/b4r.txt: no such file or directory", err.Error(); want != got {
		t.Errorf("want: %s, got: %s", want, got)
	}
}
