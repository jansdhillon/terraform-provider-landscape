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

var _ datasource.DataSource = &ScriptAttachmentDataSource{}
var _ datasource.DataSourceWithConfigure = &ScriptAttachmentDataSource{}

func NewScriptAttachmentDataSource() datasource.DataSource {
	return &ScriptAttachmentDataSource{}
}

type ScriptAttachmentDataSource struct {
	client *landscape.ClientWithResponses
}

type scriptAttachmentDataModel struct {
	Id       types.Int64  `tfsdk:"id"`
	ScriptId types.Int64  `tfsdk:"script_id"`
	Filename types.String `tfsdk:"filename"`
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
			"filename": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Filename of the attachment.",
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

func (d *ScriptAttachmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg scriptAttachmentDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if cfg.ScriptId.IsNull() || cfg.ScriptId.IsUnknown() {
		resp.Diagnostics.AddError("Missing script_id", "`script_id` must be set.")
		return
	}
	if cfg.Filename.IsNull() || cfg.Filename.IsUnknown() {
		resp.Diagnostics.AddError("Missing filename", "`filename` must be set.")
		return
	}

	scriptRes, err := d.client.GetScriptWithResponse(ctx, landscape.ScriptIdPathParam(cfg.ScriptId.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script", err.Error())
		return
	}
	if scriptRes.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read script", fmt.Sprintf("Error getting script: %s", scriptRes.Status()))
		return
	}

	v2, err := scriptRes.JSON200.AsV2Script()
	if err != nil {
		resp.Diagnostics.AddError("Unsupported script type", "Script attachments are only available for V2 scripts.")
		return
	}

	var attachment *landscape.ScriptAttachment
	if v2.Attachments != nil {
		for _, a := range *v2.Attachments {
			if a.Filename == cfg.Filename.ValueString() {
				aCopy := a
				attachment = &aCopy
				break
			}
		}
	}
	if attachment == nil {
		resp.Diagnostics.AddError("Attachment not found", fmt.Sprintf("No attachment named %q exists on script %d", cfg.Filename.ValueString(), cfg.ScriptId.ValueInt64()))
		return
	}

	state := scriptAttachmentDataModel{
		Id:       types.Int64Value(int64(attachment.Id)),
		ScriptId: cfg.ScriptId,
		Filename: cfg.Filename,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
