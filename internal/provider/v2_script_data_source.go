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

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &v1ScriptDataSource{}
var _ datasource.DataSourceWithConfigure = &v1ScriptDataSource{}

func NewV2ScriptDataSource() datasource.DataSource {
	return &v1ScriptDataSource{}
}

// v2ScriptDataSource defines the data source implementation.
type v2ScriptDataSource struct {
	client *landscape.ClientWithResponses
}

// v2ScriptDataSourceState wraps the generated API model so we can add extra
// Terraform-only attributes without duplicating the struct.
type v2ScriptDataSourceState landscape.V1Script

func (d *v2ScriptDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script"
}

func (d *v2ScriptDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "V1Script data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "V1Script identifier",
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
				Optional:            true,
				MarkdownDescription: "The version number of the script.",
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
			"is_editable": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "Whether the script is editable.",
			},
			"is_executable": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "Whether the script is executable.",
			},
			"is_redactable": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
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
			"attachments": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.NumberAttribute{Computed: true},
						"filename": schema.StringAttribute{Computed: true},
					},
				},
				MarkdownDescription: "V1Script attachments as nested objects.",
			},
			"attachments_legacy": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Legacy attachments as list of strings.",
			},
			"script_profiles": schema.ListNestedAttribute{
				Computed: true,
				Optional: true,
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

func (d *v2ScriptDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *v2ScriptDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var idValue types.Int64

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &idValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if idValue.IsUnknown() || idValue.IsNull() {
		resp.Diagnostics.AddError("Missing script identifier", "The `id` attribute must be provided for the landscape_script data source.")
		return
	}

	scriptRes, err := d.client.GetV1ScriptWithResponse(ctx, landscape.V1ScriptIdPathParam(int(idValue.ValueInt64())))
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script", err.Error())
		return
	}

	if scriptRes.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to get script", "An error occurred getting the script.")
		return
	}

	state := v1ScriptDataSourceState{
		V1Script:          *scriptRes.JSON200,
		AttachmentsLegacy: types.ListNull(types.StringType),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
