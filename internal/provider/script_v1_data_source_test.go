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

func TestScriptV1DataSourceMetadata(t *testing.T) {
	dataSource := NewScriptV1DataSource()

	var resp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "landscape"}, &resp)

	if resp.TypeName != "landscape_script_v1" {
		t.Fatalf("expected data source type name landscape_script_v1, got %q", resp.TypeName)
	}
}

func TestAccScriptV1DataSourceRequiresID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccScriptV1DataSourceMissingIDConfig,
				ExpectError: regexp.MustCompile(`(?i)id`),
			},
		},
	})
}

const testAccScriptV1DataSourceMissingIDConfig = `
provider "landscape" {}

data "landscape_script_v1" "test" {}
`
