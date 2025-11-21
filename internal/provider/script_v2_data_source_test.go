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

func TestScriptV2DataSourceMetadata(t *testing.T) {
	dataSource := NewScriptV2DataSource()

	var resp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "landscape"}, &resp)

	if resp.TypeName != "landscape_script_v2" {
		t.Fatalf("expected data source type name landscape_script_v2, got %q", resp.TypeName)
	}
}

func TestAccScriptV2DataSourceRequiresID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccScriptV2DataSourceMissingIDConfig,
				ExpectError: regexp.MustCompile(`(?i)id`),
			},
		},
	})
}

const testAccScriptV2DataSourceMissingIDConfig = `
provider "landscape" {}

data "landscape_script_v2" "test" {}
`
