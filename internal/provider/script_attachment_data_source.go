// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ datasource.DataSource = &ScriptAttachmentDataSource{}
var _ datasource.DataSourceWithConfigure = &ScriptAttachmentDataSource{}

func NewScriptAttachmentDataSource() datasource.DataSource {
	return &ScriptAttachmentDataSource{}
}

type ScriptAttachmentDataSource struct {
	client *landscape.ClientWithResponses
}

type scriptAttachmentDataSourceModel struct {
	Id       types.Int64  `tfsdk:"id"`
	ScriptId types.Int64  `tfsdk:"script_id"`
	Filename types.String `tfsdk:"filename"`
	Content  types.String `tfsdk:"content"`
}

func (d *ScriptAttachmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script_attachment"
}

func (d *ScriptAttachmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Script attachment data source (V2 scripts only).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Attachment identifier (V2 only).",
			},
			"script_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "ID of the script this attachment belongs to.",
			},
		},
	}
}

func (d *ScriptAttachmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *ScriptAttachmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state scriptAttachmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newState, diags := r.readAttachment(ctx, state.ScriptId.ValueInt64(), state.Id.ValueInt64())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *ScriptAttachmentDataSource) readAttachment(ctx context.Context, scriptID int64, attachmentID int64) (*scriptAttachmentDataSourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	scriptRes, err := r.client.GetScriptWithResponse(ctx, landscape.ScriptIdPathParam(scriptID))
	if err != nil {
		diags.AddError("Failed to read script", err.Error())
		return nil, diags
	}

	if scriptRes.JSON200 == nil {
		diags.AddError("Failed to read script", fmt.Sprintf("Error getting script: %s", scriptRes.Status()))
		return nil, diags
	}

	attachmentContent, err := r.client.GetScriptAttachmentWithResponse(ctx, int(scriptID), int(attachmentID))
	if err != nil {
		diags.AddError("Failed to read script attachment content", err.Error())
		return nil, diags
	}

	if attachmentContent.JSON200 == nil {
		if attachmentContent.JSON404 != nil {
			diags.AddError("Script not found", *attachmentContent.JSON404.Message)
			return nil, diags
		}

		diags.AddError("Error reading script attachment", attachmentContent.Status())
		return nil, diags
	}

	state := scriptAttachmentDataSourceModel{
		Id:       types.Int64Value(attachmentID),
		ScriptId: types.Int64Value(scriptID),
		Content:  types.StringValue(*attachmentContent.JSON200),
	}

	return &state, diags
}
