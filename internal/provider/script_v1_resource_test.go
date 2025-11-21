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

func TestScriptV1ResourceMetadata(t *testing.T) {
	res := NewScriptV1Resource()

	var resp pfresource.MetadataResponse
	res.Metadata(context.Background(), pfresource.MetadataRequest{ProviderTypeName: "landscape"}, &resp)

	if resp.TypeName != "landscape_script_v1" {
		t.Fatalf("expected resource type name landscape_script_v1, got %q", resp.TypeName)
	}
}

func TestAccScriptV1ResourceMissingTitle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccScriptV1ResourceMissingTitleConfig,
				ExpectError: regexp.MustCompile(`(?i)title`),
			},
		},
	})
}

const testAccScriptV1ResourceMissingTitleConfig = `
provider "landscape" {}

resource "landscape_script_v1" "test" {
  code = "#!/bin/bash\necho test"
}
`
