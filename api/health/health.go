package health

import "net/http"

// health Route
const AppHealthPath = "/health"


func AppHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`Health Check Ok`))
}