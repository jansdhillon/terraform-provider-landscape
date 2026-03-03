// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"landscape": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside the landscape provider.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
// var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
// 	"landscape": providerserver.NewProtocol6WithError(New("test")()),
// 	"echo":      echoprovider.NewProviderServer(),
// }

func testAccPreCheck(t *testing.T) {
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
