// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestScriptDataSourceMetadata(t *testing.T) {
	dataSource := NewScriptDataSource()

	var resp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "landscape"}, &resp)

	if resp.TypeName != "landscape_script" {
		t.Fatalf("expected data source type name landscape_script, got %q", resp.TypeName)
	}
}

func TestAccScriptDataSourceRequiresID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccScriptDataSourceMissingIDConfig,
				ExpectError: regexp.MustCompile(`(?i)id`),
			},
		},
	})
}

const testAccScriptDataSourceMissingIDConfig = `
provider "landscape" {
  base_url   = "https://example.invalid"
  access_key = "test"
  secret_key = "test"
}

data "landscape_script" "test" {}
`
