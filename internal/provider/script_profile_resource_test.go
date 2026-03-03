// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"regexp"
	"testing"

	pfdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	pfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestScriptProfileResourceMetadata(t *testing.T) {
	res := NewScriptProfileResource()

	var resp pfresource.MetadataResponse
	res.Metadata(context.Background(), pfresource.MetadataRequest{ProviderTypeName: "landscape"}, &resp)

	if resp.TypeName != "landscape_script_profile" {
		t.Fatalf("expected resource type name landscape_script_profile, got %q", resp.TypeName)
	}
}

func TestScriptProfileDataSourceMetadata(t *testing.T) {
	ds := NewScriptProfileDataSource()

	var resp pfdatasource.MetadataResponse
	ds.Metadata(context.Background(), pfdatasource.MetadataRequest{ProviderTypeName: "landscape"}, &resp)

	if resp.TypeName != "landscape_script_profile" {
		t.Fatalf("expected data source type name landscape_script_profile, got %q", resp.TypeName)
	}
}

func TestAccScriptProfileResourceMissingTrigger(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccScriptProfileResourceMissingTriggerConfig,
				ExpectError: regexp.MustCompile(`(?i)trigger`),
			},
		},
	})
}

func TestAccScriptProfileResourceInvalidTriggerType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccScriptProfileResourceInvalidTriggerTypeConfig,
				ExpectError: regexp.MustCompile(`(?i)trigger type`),
			},
		},
	})
}

const testAccScriptProfileResourceMissingTriggerConfig = `
provider "landscape" {}

resource "landscape_script_profile" "test" {
  title      = "Test Script Profile"
  script_id  = 1
  username   = "root"
  time_limit = 600
}
`

const testAccScriptProfileResourceInvalidTriggerTypeConfig = `
provider "landscape" {}

resource "landscape_script_profile" "test" {
  title      = "Test Script Profile"
  script_id  = 1
  username   = "root"
  time_limit = 600
  trigger = {
    type = "invalid_type"
  }
}
`
