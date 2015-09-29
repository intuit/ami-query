// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package api

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"
)

// contextKey is a custom type that represents keys within a context.Context.
type contextKey int

// CacheManagerKey is the amicache.Manager key.
const CacheManagerKey contextKey = 1

// ContextHandler is an HTTP handler that adds context.Context to requests.
type ContextHandler interface {
	ServeHTTP(context.Context, http.ResponseWriter, *http.Request) (int, error)
}

// ContextHandlerFunc is a function adapter that allows the use of ordinary
// functions as HTTP Context handlers. If f is a function with the appropriate
// signature, ContextHandlerFunc(f) is a ContextHandler object that calls f.
type ContextHandlerFunc func(context.Context, http.ResponseWriter, *http.Request) (int, error)

// ServeHTTP returns the result of f(ctx, w, r).
func (f ContextHandlerFunc) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	return f(ctx, w, r)
}

// ContextAdapter joins a context.Context and ContextHandler and implements the
// http.Handler interface.
type ContextAdapter struct {
	Context context.Context
	Handler ContextHandler
}

// ServeHTTP passes context.Context to HTTP requests.
func (c *ContextAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if status, err := c.Handler.ServeHTTP(c.Context, w, r); err != nil {
		var id string
		switch status {
		case http.StatusBadRequest:
			id = "bad_request"
		case http.StatusInternalServerError:
			id = "internal_error"
		default:
			id = "unknown_error"
			status = http.StatusInternalServerError
		}
		http.Error(w, fmt.Sprintf(`{"id":"%s","message":"%s"}`, id, err), status)
	}
}
