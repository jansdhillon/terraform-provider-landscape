// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ datasource.DataSource = &v1ScriptDataSource{}
var _ datasource.DataSourceWithConfigure = &v1ScriptDataSource{}

func NewV1ScriptDataSource() datasource.DataSource {
	return &v1ScriptDataSource{}
}

// v1ScriptDataSource defines the data source implementation.
type v1ScriptDataSource struct {
	client *landscape.ClientWithResponses
}

type v1ScriptDataSourceModel = landscape.V1Script

func (d *v1ScriptDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_v1_script"
}

func (d *v1ScriptDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "V1 Script data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The script identifier",
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title of the script.",
				Computed:            true,
				Optional:            true,
			},
			"access_group": schema.StringAttribute{
				MarkdownDescription: "The access group the script is in.",
				Computed:            true,
				Optional:            true,
			},
			"creator": schema.SingleNestedAttribute{
				Computed: true,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"id":    schema.NumberAttribute{Computed: true, Optional: true},
					"name":  schema.StringAttribute{Computed: true, Optional: true},
					"email": schema.StringAttribute{Computed: true, Optional: true},
				},
				MarkdownDescription: "The creator of the (legacy) script.",
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The status of the script.",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The username associated with the script.",
			},
			"time_limit": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The time limit in milliseconds for a script to complete successfully.",
			},
			"attachments": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Legacy attachments as list of strings.",
			},
		},
	}
}

func (d *v1ScriptDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*landscape.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *landscape.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *v1ScriptDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var idValue types.Int64

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &idValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if idValue.IsUnknown() || idValue.IsNull() {
		resp.Diagnostics.AddError("Missing script identifier", "The `id` attribute must be provided for the landscape_v1_script data source.")
		return
	}

	scriptRes, err := d.client.GetScriptWithResponse(ctx, landscape.ScriptIdPathParam(int(idValue.ValueInt64())))
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script", err.Error())
		return
	}

	if scriptRes.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to get script", "An error occurred getting the script.")
		return
	}

	script := *scriptRes.JSON200

	v1Script, err := script.AsV1Script()
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert into (legacy) V1 script", "Couldn't convert returned script into a V1 script (is it a modern, V2 script?)")
	}

	state := v1ScriptDataSourceModel(v1Script)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
