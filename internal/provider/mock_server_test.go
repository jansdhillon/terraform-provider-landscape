// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// startMockServer starts an httptest server that satisfies provider configuration
// (login endpoint) so that schema/plan-time validators fire without real credentials.
// It sets LANDSCAPE_BASE_URL, LANDSCAPE_ACCESS_KEY, and LANDSCAPE_SECRET_KEY env vars
// for the duration of the test.
func startMockServer(t *testing.T) {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/login/access-key", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "mock-token"})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	t.Setenv("LANDSCAPE_BASE_URL", srv.URL)
	t.Setenv("LANDSCAPE_ACCESS_KEY", "mock-access-key")
	t.Setenv("LANDSCAPE_SECRET_KEY", "mock-secret-key")
}
