// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ datasource.DataSource = &ScriptV2DataSource{}
var _ datasource.DataSourceWithConfigure = &ScriptV2DataSource{}

func NewScriptV2DataSource() datasource.DataSource {
	return &ScriptV2DataSource{}
}

type ScriptV2DataSource struct {
	client *landscape.ClientWithResponses
}

type ScriptV2DataSourceModel struct {
	Id             types.Int64  `tfsdk:"id"`
	Title          types.String `tfsdk:"title"`
	AccessGroup    types.String `tfsdk:"access_group"`
	Code           types.String `tfsdk:"code"`
	CreatedAt      types.String `tfsdk:"created_at"`
	CreatedBy      types.Object `tfsdk:"created_by"`
	LastEditedAt   types.String `tfsdk:"last_edited_at"`
	Status         types.String `tfsdk:"status"`
	VersionNumber  types.Int64  `tfsdk:"version_number"`
	Username       types.String `tfsdk:"username"`
	TimeLimit      types.Int64  `tfsdk:"time_limit"`
	IsEditable     types.Bool   `tfsdk:"is_editable"`
	IsExecutable   types.Bool   `tfsdk:"is_executable"`
	IsRedactable   types.Bool   `tfsdk:"is_redactable"`
	LastEditedBy   types.Object `tfsdk:"last_edited_by"`
	Attachments    types.List   `tfsdk:"attachments"`
	ScriptProfiles types.List   `tfsdk:"script_profiles"`
}

func (d *ScriptV2DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script_v2"
}

func (d *ScriptV2DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "V2 script data source",
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
			},
			"code": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The raw script code content including the interpreter line.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the script was created.",
			},
			"created_by": schema.SingleNestedAttribute{
				MarkdownDescription: "The creator of the script.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"id":   schema.Int64Attribute{Computed: true},
					"name": schema.StringAttribute{Computed: true},
				},
			},
			"last_edited_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the script was last edited.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the script (ACTIVE, ARCHIVED, or REDACTED).",
			},
			"version_number": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The version number of the script.",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Linux user that will run the script on the Landscape Client instance.",
			},
			"time_limit": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The time limit in seconds for a script to complete successfully.",
			},
			"is_editable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script is editable by the caller.",
			},
			"is_executable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script is executable by the caller.",
			},
			"is_redactable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script is redactable by the caller.",
			},
			"last_edited_by": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id":   schema.Int64Attribute{Computed: true},
					"name": schema.StringAttribute{Computed: true},
				},
				MarkdownDescription: "The Landscape user who last edited the script.",
			},
			"attachments": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.Int64Attribute{Computed: true},
						"filename": schema.StringAttribute{Computed: true},
					},
				},
				MarkdownDescription: "Attachments associated with this script.",
			},
			"script_profiles": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":    schema.Int64Attribute{Computed: true},
						"title": schema.StringAttribute{Computed: true},
					},
				},
				MarkdownDescription: "List of script profiles associated with the script.",
			},
		},
	}
}

func (d *ScriptV2DataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScriptV2DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var idValue types.Int64

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &idValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if idValue.IsUnknown() || idValue.IsNull() {
		resp.Diagnostics.AddError("Missing script ID", "The `id` attribute must be provided for the landscape_script_v2 data source.")
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

	v2Script, err := scriptRes.JSON200.AsV2Script()
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert script", "The script is not a V2 script. Use landscape_script_v1 data source instead.")
		return
	}

	state, diags := v2ScriptToDataSourceState(ctx, v2Script)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func v2ScriptToDataSourceState(ctx context.Context, v2Script landscape.V2Script) (ScriptV2DataSourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	v2CreatedByAttrTypes := map[string]attr.Type{
		"id":   types.Int64Type,
		"name": types.StringType,
	}

	createdBy := types.ObjectNull(v2CreatedByAttrTypes)
	if v2Script.CreatedBy != nil {
		obj, d := types.ObjectValue(v2CreatedByAttrTypes, map[string]attr.Value{
			"id":   types.Int64Value(int64(*v2Script.CreatedBy.Id)),
			"name": types.StringPointerValue(v2Script.CreatedBy.Name),
		})
		diags.Append(d...)
		if !diags.HasError() {
			createdBy = obj
		} else {
			tflog.Debug(ctx, "Couldn't convert script created_by field into an object")
		}
	}

	lastEditedBy := types.ObjectNull(v2LastEditedByAttrTypes)
	if v2Script.LastEditedBy != nil {
		lastEditedByID := types.Int64Null()
		if v2Script.LastEditedBy.Id != nil {
			lastEditedByID = types.Int64Value(int64(*v2Script.LastEditedBy.Id))
		}
		obj, d := types.ObjectValue(v2LastEditedByAttrTypes, map[string]attr.Value{
			"id":   lastEditedByID,
			"name": types.StringPointerValue(v2Script.LastEditedBy.Name),
		})
		diags.Append(d...)
		if !diags.HasError() {
			lastEditedBy = obj
		}
	}

	attachments := types.ListNull(types.ObjectType{AttrTypes: scriptAttachmentAttrType})
	if v2Script.Attachments != nil {
		elems := make([]attr.Value, 0, len(*v2Script.Attachments))
		for _, a := range *v2Script.Attachments {
			elem, d := types.ObjectValue(scriptAttachmentAttrType, map[string]attr.Value{
				"id":       types.Int64PointerValue(int64Ptr(int64(a.Id))),
				"filename": types.StringValue(a.Filename),
			})
			diags.Append(d...)
			elems = append(elems, elem)
		}
		if !diags.HasError() {
			list, d := types.ListValue(types.ObjectType{AttrTypes: scriptAttachmentAttrType}, elems)
			diags.Append(d...)
			if !diags.HasError() {
				attachments = list
			}
		}
	}

	scriptProfiles := types.ListNull(types.ObjectType{AttrTypes: scriptProfileAttrType})
	if v2Script.ScriptProfiles != nil {
		elems := make([]attr.Value, 0, len(*v2Script.ScriptProfiles))
		for _, sp := range *v2Script.ScriptProfiles {
			elem, d := types.ObjectValue(scriptProfileAttrType, map[string]attr.Value{
				"id":    types.Int64PointerValue(int64Ptr(int64(sp.Id))),
				"title": types.StringValue(sp.Title),
			})
			diags.Append(d...)
			elems = append(elems, elem)
		}
		if !diags.HasError() {
			list, d := types.ListValue(types.ObjectType{AttrTypes: scriptProfileAttrType}, elems)
			diags.Append(d...)
			if !diags.HasError() {
				scriptProfiles = list
			}
		}
	}

	var mergedCode types.String
	if v2Script.Interpreter != nil && v2Script.Code != nil {
		mergedCode = types.StringValue(fmt.Sprintf("#!%s\n%s", *v2Script.Interpreter, *v2Script.Code))
	}

	versionNumber := types.Int64Null()
	if v2Script.VersionNumber != nil {
		versionNumber = types.Int64Value(int64(*v2Script.VersionNumber))
	}

	timeLimit := types.Int64Null()
	if v2Script.TimeLimit != nil {
		tl := int64(*v2Script.TimeLimit)
		timeLimit = types.Int64Value(tl)
	}

	return ScriptV2DataSourceModel{
		Id:             types.Int64Value(int64(v2Script.Id)),
		Title:          types.StringValue(v2Script.Title),
		AccessGroup:    types.StringPointerValue(v2Script.AccessGroup),
		Code:           mergedCode,
		CreatedAt:      types.StringPointerValue(v2Script.CreatedAt),
		CreatedBy:      createdBy,
		LastEditedAt:   types.StringPointerValue(v2Script.LastEditedAt),
		Status:         types.StringValue(string(v2Script.Status)),
		VersionNumber:  versionNumber,
		Username:       types.StringPointerValue(v2Script.Username),
		TimeLimit:      timeLimit,
		IsEditable:     types.BoolPointerValue(v2Script.IsEditable),
		IsExecutable:   types.BoolPointerValue(v2Script.IsExecutable),
		IsRedactable:   types.BoolPointerValue(v2Script.IsRedactable),
		LastEditedBy:   lastEditedBy,
		Attachments:    attachments,
		ScriptProfiles: scriptProfiles,
	}, diags
}
