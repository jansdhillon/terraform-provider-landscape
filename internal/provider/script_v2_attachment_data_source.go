// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ datasource.DataSource = &ScriptV2AttachmentDataSource{}

func NewScriptV2AttachmentDataSource() datasource.DataSource {
	return &ScriptV2AttachmentDataSource{}
}

type ScriptV2AttachmentDataSource struct {
	client *landscape.ClientWithResponses
}

type scriptV2AttachmentDataSourceModel struct {
	Id       types.Int64  `tfsdk:"id"`
	ScriptId types.Int64  `tfsdk:"script_id"`
	Filename types.String `tfsdk:"filename"`
	Content  types.String `tfsdk:"content"`
}

func (d *ScriptV2AttachmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script_v2_attachment"
}

func (d *ScriptV2AttachmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "V2 script attachment data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Attachment ID.",
			},
			"script_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "ID of the V2 script.",
			},
			"filename": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Filename of the attachment.",
			},
			"content": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Attachment content.",
			},
		},
	}
}

func (d *ScriptV2AttachmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScriptV2AttachmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config scriptV2AttachmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scriptRes, err := d.client.GetScriptWithResponse(ctx, landscape.ScriptIdPathParam(config.ScriptId.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script", err.Error())
		return
	}

	if scriptRes.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read script", fmt.Sprintf("Error getting script: %s", scriptRes.Status()))
		return
	}

	v2Script, err := scriptRes.JSON200.AsV2Script()
	if err != nil {
		resp.Diagnostics.AddError("Script is not a V2 script", "This data source is for V2 scripts only. Use landscape_script_v1_attachment for V1 scripts.")
		return
	}

	var filename string
	found := false
	if v2Script.Attachments != nil {
		for _, att := range *v2Script.Attachments {
			if int64(att.Id) == config.Id.ValueInt64() {
				filename = att.Filename
				found = true
				break
			}
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"Attachment not found",
			fmt.Sprintf("Attachment with ID %d not found in script %d", config.Id.ValueInt64(), config.ScriptId.ValueInt64()),
		)
		return
	}

	attachmentContent, err := d.client.GetScriptAttachmentWithResponse(ctx, int(config.ScriptId.ValueInt64()), int(config.Id.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to read attachment content", err.Error())
		return
	}

	if attachmentContent.StatusCode() == 404 {
		resp.Diagnostics.AddError("Attachment not found", *attachmentContent.JSON404.Message)
		return
	}

	// NOTE: attachments are returned as raw plain text instead of JSON
	bodyStr := string(attachmentContent.Body)
	if attachmentContent.StatusCode() == 200 && bodyStr != "" {
		state := scriptV2AttachmentDataSourceModel{
			Id:       config.Id,
			ScriptId: config.ScriptId,
			Filename: types.StringValue(filename),
			Content:  types.StringValue(bodyStr),
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	resp.Diagnostics.AddError("Error reading attachment", fmt.Sprintf("%s\n%s", attachmentContent.Status(), string(attachmentContent.Body)))
}
