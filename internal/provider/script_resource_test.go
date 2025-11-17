// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"regexp"
	"testing"

	pfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestScriptResourceMetadata(t *testing.T) {
	res := NewScriptResource()

	var resp pfresource.MetadataResponse
	res.Metadata(context.Background(), pfresource.MetadataRequest{ProviderTypeName: "landscape"}, &resp)

	if resp.TypeName != "landscape_script" {
		t.Fatalf("expected resource type name landscape_script, got %q", resp.TypeName)
	}
}

func TestAccScriptResourceInvalidType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccScriptResourceInvalidConfig,
				ExpectError: regexp.MustCompile(`Invalid script status`),
			},
		},
	})
}

const testAccScriptResourceInvalidConfig = `
provider "landscape" {}

resource "landscape_script" "test" {
  title       = "invalid script type"
  code        = "echo invalid"
  status      = "bogus"
}
`
