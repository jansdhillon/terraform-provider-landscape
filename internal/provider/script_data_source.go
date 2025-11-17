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

var _ datasource.DataSource = &ScriptDataSource{}
var _ datasource.DataSourceWithConfigure = &ScriptDataSource{}

func NewScriptDataSource() datasource.DataSource {
	return &ScriptDataSource{}
}

// scriptDataSource defines the data source implementation.
type ScriptDataSource struct {
	client *landscape.ClientWithResponses
}

func (d *ScriptDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script"
}

func (d *ScriptDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Script data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Script identifier for this account in Landscape.",
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title of the script.",
				Computed:            true,
			},
			"access_group": schema.StringAttribute{
				MarkdownDescription: "The access group the script is in. Defaults to 'global'.",
				Computed:            true,
				Optional:            true,
			},
			"code": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The raw script code content. Note that this does NOT split on the interpreter/executable portion.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "When the script was created. Not applicable for V1 scripts.",
			},
			"created_by": schema.SingleNestedAttribute{
				MarkdownDescription: "The creator of the script. Note that only V1 scripts have an email.",
				Computed:            true,
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"id":    schema.NumberAttribute{Computed: true},
					"name":  schema.StringAttribute{Computed: true},
					"email": schema.StringAttribute{Computed: true, Optional: true},
				},
			},
			"last_edited_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the script was last edited. Not applicable for V1 scripts.",
				Optional:            true,
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the script (active, archived, or redacted), or V1 for legacy scripts.",
			},
			"version_number": schema.Int64Attribute{
				Computed: true,

				MarkdownDescription: "The version number of the script. Not applicable for V1 scripts.",
			},
			"username": schema.StringAttribute{
				Computed: true,
				Optional: true,

				MarkdownDescription: "The Linux user that will run the script on the Landscape Client instance.",
			},
			"time_limit": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The time limit in second for a script to complete successfully.",
			},
			"is_editable": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "Whether the script is editable by the caller. Not applicable for V1 scripts.",
			},
			"is_executable": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "Whether the script is executable by the caller. Not applicable for V1 scripts.",
			},
			"is_redactable": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "Whether the script is redactable by the caller. Not applicable for V1 scripts.",
			},
			"last_edited_by": schema.SingleNestedAttribute{
				Computed: true,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"id":   schema.NumberAttribute{Computed: true, Optional: true},
					"name": schema.StringAttribute{Computed: true, Optional: true},
				},
				MarkdownDescription: "The Landscape user who last edited the script. Not applicable for V1 scripts.",
			},
			"attachments": schema.ListNestedAttribute{
				Computed: true,
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.NumberAttribute{Computed: true, Optional: true},
						"filename": schema.StringAttribute{Computed: true},
					},
				},
				MarkdownDescription: "Attachments associated with this script. IDs of the attachments are not returned for V1 scripts.",
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
				MarkdownDescription: "List of script profiles associated with the script. Not applicable for V1 scripts.",
			},
			"script_type": schema.StringAttribute{
				Computed: true,
				Optional: true,
				MarkdownDescription: `The script version to create. Either V1 for legacy, V2 for 'modern' scripts with versioning and status.
				Note that V1 scripts are only visible in the legacy Landscape UI and V2+ scripts are only visible in the modern Landscape UI.`,
			},
		},
	}

}

func (d *ScriptDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScriptDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var idValue types.Int64

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &idValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if idValue.IsUnknown() || idValue.IsNull() {
		resp.Diagnostics.AddError("Missing script ID", "The `id` attribute must be provided for the landscape_script data source.")
		return
	}

	scriptRes, err := d.client.GetScriptWithResponse(ctx, landscape.ScriptIdPathParam(idValue.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script", err.Error())
		return
	}

	if scriptRes.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to get script", "An error occurred getting the script.")
		return
	}

	script := *scriptRes.JSON200
	if v2Script, err := script.AsV2Script(); err == nil {
		state, diags := v2ScriptToState(ctx, v2Script, "V2")
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
		return
	}

	v1Script, err := script.AsV1Script()
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert script", "Could not convert returned script into V1 or V2 form")
		return
	}

	state, diags := v1ToState(ctx, d.client, v1Script, "V1")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
