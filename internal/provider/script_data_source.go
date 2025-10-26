// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	landscape "github.com/jansdhillon/landscape-go-client/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &scriptDataSource{}
var _ datasource.DataSourceWithConfigure = &scriptDataSource{}

func NewScriptDataSource() datasource.DataSource {
	return &scriptDataSource{}
}

// scriptDataSource defines the data source implementation.
type scriptDataSource struct {
	client *landscape.Client
}

// ScriptDataSourceModel describes the data source data model.
type ScriptDataSourceModel landscape.Script

func (d *scriptDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script"
}

func (d *scriptDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Script data source",

		Attributes: map[string]schema.Attribute{
			"access_group": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The access group the script is in.",
			},
			"id": schema.NumberAttribute{
				MarkdownDescription: "Script identifier",
				Computed:            true,
			},
			// Script defines model for Script.
		},
	}
}

func (d *scriptDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*landscape.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *landscape.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *scriptDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ScriptDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
