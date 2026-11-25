// Command openapi-dump emits the exact generated Not Human Search OpenAPI
// document for release validation. It performs no network or database work.
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/unitedideas/nothumansearch/internal/handlers"
)

func main() {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	handlers.NewSEOHandler(nil, "https://nothumansearch.ai").OpenAPISpec(recorder, request)
	if recorder.Code != http.StatusOK {
		_, _ = fmt.Fprintf(os.Stderr, "OpenAPI generator returned HTTP %d\n", recorder.Code)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(recorder.Body.Bytes()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "write OpenAPI document: %v\n", err)
		os.Exit(1)
	}
}
