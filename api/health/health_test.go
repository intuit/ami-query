package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAppHealthCheck(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", AppHealthPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(AppHealthCheck)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Failed.: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `Health Check Ok`
	if rr.Body.String() != expected {
		t.Errorf("Unexpected Response: got %v want %v",
			rr.Body.String(), expected)
	}
}
