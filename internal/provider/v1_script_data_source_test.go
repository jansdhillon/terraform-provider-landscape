// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// Acceptance tests for the V1 script data source.
func TestAccV1ScriptDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccScriptDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.landscape_v1_script.test",
						tfjsonpath.New("id"),
						knownvalue.NumberExact(big.NewFloat(1)),
					),
				},
			},
		},
	})
}

const testAccScriptDataSourceConfig = `
provider "landscape" {}

data "landscape_v1_script" "test" {
  id = 21434
}
`
