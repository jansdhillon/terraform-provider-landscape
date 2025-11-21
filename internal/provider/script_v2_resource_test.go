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

func TestScriptV2ResourceMetadata(t *testing.T) {
	res := NewScriptV2Resource()

	var resp pfresource.MetadataResponse
	res.Metadata(context.Background(), pfresource.MetadataRequest{ProviderTypeName: "landscape"}, &resp)

	if resp.TypeName != "landscape_script_v2" {
		t.Fatalf("expected resource type name landscape_script_v2, got %q", resp.TypeName)
	}
}

func TestAccScriptV2ResourceMissingCode(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccScriptV2ResourceMissingCodeConfig,
				ExpectError: regexp.MustCompile(`(?i)code`),
			},
		},
	})
}

const testAccScriptV2ResourceMissingCodeConfig = `
provider "landscape" {}

resource "landscape_script_v2" "test" {
  title = "Test V2 Script"
}
`
