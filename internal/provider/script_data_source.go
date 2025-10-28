// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &scriptDataSource{}
var _ datasource.DataSourceWithConfigure = &scriptDataSource{}

func NewScriptDataSource() datasource.DataSource {
	return &scriptDataSource{}
}

// scriptDataSource defines the data source implementation.
type scriptDataSource struct {
	client *landscape.ClientWithResponses
}

// ScriptDataSourceModel describes the data source data model.
type scriptDataSourceModel = landscape.Script

func (d *scriptDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script"
}

func (d *scriptDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Script data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Script identifier",
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title of the script.",
				Computed:            true,
				Optional:            true,
			},
			"access_group": schema.StringAttribute{
				MarkdownDescription: "The access group the script is in.",
				Computed:            true,
			},
			"code": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The script code content.",
				Optional:            true,
			},
			"interpreter": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The script interpreter.",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "When the script was created.",
			},
			"created_by": schema.SingleNestedAttribute{
				Computed: true,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"id":   schema.NumberAttribute{Computed: true, Optional: true},
					"name": schema.StringAttribute{Computed: true, Optional: true},
				},
				MarkdownDescription: "The creator of the script.",
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
			"last_edited_at": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "When the script was last edited.",
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The status of the script.",
			},
			"version_number": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The version number of the script.",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The username associated with the script.",
			},
			"time_limit": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The time limit in milliseconds for a script to complete successfully.",
			},
			"is_editable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script is editable.",
			},
			"is_executable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script is executable.",
			},
			"is_redactable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script is redactable.",
			},
			"last_edited_by": schema.SingleNestedAttribute{
				Computed: true,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"id":   schema.NumberAttribute{Computed: true, Optional: true},
					"name": schema.StringAttribute{Computed: true, Optional: true},
				},
				MarkdownDescription: "The user who last edited the script.",
			},
			"attachments": schema.DynamicAttribute{
				Computed:            true,
				MarkdownDescription: "Attachments (list of strings or objects) stored as opaque JSON.",
			},
			"script_profiles": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":    schema.NumberAttribute{Computed: true},
						"title": schema.StringAttribute{Computed: true},
					},
				},
				MarkdownDescription: "List of script profiles.",
			},
		},
	}
}

func (d *scriptDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *scriptDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var idValue types.Int64

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &idValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if idValue.IsUnknown() || idValue.IsNull() {
		resp.Diagnostics.AddError("Missing script identifier", "The `id` attribute must be provided for the landscape_script data source.")
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

	state := scriptDataSourceModel(*scriptRes.JSON200)

	originalAttachments := scriptRes.JSON200.Attachments
	state.Attachments = nil

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	if originalAttachments == nil || len(*originalAttachments) == 0 {
		return
	}

	// Unfortunately the legacy response is a list of strings, while the "modern"
	// attachment response is a keyed map (object). The tfsk doesn't let us convert
	// from objects to strings or vice-versa so we make it "dynamic" and determine it
	// at read time to preserve type safety.
	legacyAttachmentsStrings := make([]attr.Value, 0)
	for _, a := range *originalAttachments {
		legacyAttachment, err := a.AsLegacyScriptAttachment()
		if err != nil {
			legacyAttachmentsStrings = nil
			break
		}

		legacyAttachmentsStrings = append(legacyAttachmentsStrings, types.StringValue(string(legacyAttachment)))
	}

	if legacyAttachmentsStrings != nil {
		dynVal := types.DynamicValue(types.ListValueMust(types.StringType, legacyAttachmentsStrings))
		tflog.Debug(ctx, "dynval is list of strings (legacy)")
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("attachments"), dynVal)...)
		return
	}

	attachmentObjects := make([]attr.Value, 0)
	for _, a := range *originalAttachments {
		attachment, err := a.AsScriptAttachment()
		if err != nil {
			attachmentObjects = nil
			break
		}

		id := types.NumberValue(big.NewFloat(float64(attachment.Id)))
		filename := types.StringValue(attachment.Filename)
		attachmentObject, diags := types.ObjectValue(map[string]attr.Type{"id": id.Type(ctx), "filename": filename.Type(ctx)}, map[string]attr.Value{"id": id, "filename": filename})
		if diags.HasError() {
			tflog.Error(ctx, "couldn't convert script attachment into object")
		}

		attachmentObjects = append(attachmentObjects, attachmentObject)
	}

	if attachmentObjects != nil {
		objectType := types.ObjectType{AttrTypes: map[string]attr.Type{"id": types.NumberType, "filename": types.StringType}}
		dynVal := types.DynamicValue(types.ListValueMust(objectType, attachmentObjects))
		tflog.Debug(ctx, "dynval is list of objects")
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("attachments"), dynVal)...)
	}
}
