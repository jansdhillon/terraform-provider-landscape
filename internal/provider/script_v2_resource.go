// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ resource.Resource = &ScriptV2Resource{}
var _ resource.ResourceWithImportState = &ScriptV2Resource{}

func NewScriptV2Resource() resource.Resource {
	return &ScriptV2Resource{}
}

type ScriptV2Resource struct {
	client *landscape.ClientWithResponses
}

type ScriptV2ResourceModel struct {
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

func (r *ScriptV2Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script_v2"
}

func (r *ScriptV2Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "V2 script resource",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Script identifier for this account in Landscape.",
			},
			"title": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The title of the script.",
			},
			"access_group": resourceschema.StringAttribute{
				MarkdownDescription: "The access group the script is in. Defaults to 'global'.",
				Computed:            true,
				Optional:            true,
			},
			"code": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The raw script code content including the interpreter line.",
			},
			"created_at": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the script was created.",
			},
			"created_by": resourceschema.SingleNestedAttribute{
				MarkdownDescription: "The creator of the script.",
				Computed:            true,
				Attributes: map[string]resourceschema.Attribute{
					"id":   resourceschema.Int64Attribute{Computed: true},
					"name": resourceschema.StringAttribute{Computed: true},
				},
			},
			"last_edited_at": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the script was last edited.",
			},
			"status": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the script (ACTIVE, ARCHIVED, or REDACTED).",
			},
			"version_number": resourceschema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The version number of the script.",
			},
			"username": resourceschema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The Linux user that will run the script on the Landscape Client instance.",
			},
			"time_limit": resourceschema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The time limit in seconds for a script to complete successfully.",
			},
			"is_editable": resourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script is editable by the caller.",
			},
			"is_executable": resourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script is executable by the caller.",
			},
			"is_redactable": resourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script is redactable by the caller.",
			},
			"last_edited_by": resourceschema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]resourceschema.Attribute{
					"id":   resourceschema.Int64Attribute{Computed: true},
					"name": resourceschema.StringAttribute{Computed: true},
				},
				MarkdownDescription: "The Landscape user who last edited the script.",
			},
			"attachments": resourceschema.ListNestedAttribute{
				Computed: true,
				NestedObject: resourceschema.NestedAttributeObject{
					Attributes: map[string]resourceschema.Attribute{
						"id":       resourceschema.Int64Attribute{Computed: true},
						"filename": resourceschema.StringAttribute{Computed: true},
					},
				},
				MarkdownDescription: "Attachments associated with this script.",
			},
			"script_profiles": resourceschema.ListNestedAttribute{
				Computed: true,
				NestedObject: resourceschema.NestedAttributeObject{
					Attributes: map[string]resourceschema.Attribute{
						"id":    resourceschema.Int64Attribute{Computed: true},
						"title": resourceschema.StringAttribute{Computed: true},
					},
				},
				MarkdownDescription: "List of script profiles for this script.",
			},
		},
	}
}

func (r *ScriptV2Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*landscape.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *landscape.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ScriptV2Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var title types.String
	var codeAttr types.String
	var username types.String
	var timeLimit types.Int64
	var accessGroup types.String

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("title"), &title)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("code"), &codeAttr)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("username"), &username)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("time_limit"), &timeLimit)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("access_group"), &accessGroup)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if title.IsNull() || title.IsUnknown() {
		resp.Diagnostics.AddError("Missing title", "`title` must be set.")
		return
	}

	if codeAttr.IsNull() || codeAttr.IsUnknown() {
		resp.Diagnostics.AddError("Missing code", "`code` must be set.")
		return
	}

	vals := url.Values{
		"title":       []string{title.ValueString()},
		"code":        []string{base64.StdEncoding.EncodeToString([]byte(codeAttr.ValueString()))},
		"script_type": []string{"V2"},
	}

	if !timeLimit.IsNull() && !timeLimit.IsUnknown() {
		vals.Add("time_limit", fmt.Sprint(timeLimit.ValueInt64()))
	}
	if !username.IsNull() && !username.IsUnknown() {
		vals.Add("username", username.ValueString())
	}
	if !accessGroup.IsNull() && !accessGroup.IsUnknown() {
		vals.Add("access_group", accessGroup.ValueString())
	}

	createRes, err := r.client.InvokeLegacyActionWithResponse(ctx, landscape.LegacyActionParams("CreateScript"), landscape.EncodeQueryRequestEditor(vals))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create script", err.Error())
		return
	}

	if createRes.JSON200 == nil {
		errMsg := "Unexpected error creating script"
		if createRes.JSON400 != nil && createRes.JSON400.Message != nil {
			errMsg = fmt.Sprintf("%s: %s", errMsg, *createRes.JSON400.Message)
		} else if len(createRes.Body) > 0 {
			errMsg = fmt.Sprintf("%s: %s", errMsg, string(createRes.Body))
		}
		resp.Diagnostics.AddError("Failed to create script", errMsg)
		return
	}

	scriptRes, err := createRes.JSON200.AsScriptResult()
	if err != nil {
		resp.Diagnostics.AddError("Failed to decode response as script", err.Error())
		return
	}

	v2Script, err := scriptRes.AsV2Script()
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert script response into V2 script", err.Error())
		return
	}

	state, diags := v2ScriptToResourceState(ctx, v2Script)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ScriptV2Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var current ScriptV2ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if current.Id.IsNull() || current.Id.IsUnknown() {
		resp.Diagnostics.AddError("Missing script ID", "The `id` attribute must be set in state to read a script.")
		return
	}

	scriptRes, err := r.client.GetScriptWithResponse(ctx, landscape.ScriptIdPathParam(current.Id.ValueInt64()))
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
		resp.Diagnostics.AddError("Failed to convert script", "The script is not a V2 script.")
		return
	}

	state, diags := v2ScriptToResourceState(ctx, v2Script)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ScriptV2Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ScriptV2ResourceModel
	var state ScriptV2ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vals := url.Values{
		"script_id": []string{fmt.Sprint(state.Id.ValueInt64())},
	}

	if !plan.Title.IsUnknown() && !plan.Title.IsNull() {
		vals.Set("title", plan.Title.ValueString())
	}
	if !plan.TimeLimit.IsUnknown() && !plan.TimeLimit.IsNull() {
		vals.Set("time_limit", fmt.Sprint(plan.TimeLimit.ValueInt64()))
	}
	if !plan.Username.IsUnknown() && !plan.Username.IsNull() {
		vals.Set("username", plan.Username.ValueString())
	}
	if !plan.Code.IsUnknown() && !plan.Code.IsNull() && plan.Code != state.Code {
		b64 := base64.StdEncoding.EncodeToString([]byte(plan.Code.ValueString()))
		vals.Set("code", b64)
	}

	editor := landscape.EncodeQueryRequestEditor(vals)
	res, err := r.client.InvokeLegacyActionWithResponse(ctx, landscape.LegacyActionParams("EditScript"), editor)
	if err != nil {
		resp.Diagnostics.AddError("Update failed", err.Error())
		return
	}

	if res.JSON200 == nil {
		resp.Diagnostics.AddError("Update failed", res.Status())
		return
	}

	scriptRes, err := res.JSON200.AsScriptResult()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script", fmt.Sprintf("Error getting script: %s", err))
		return
	}

	v2, err := scriptRes.AsV2Script()
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert script", "The script is not a V2 script.")
		return
	}

	newState, stateDiags := v2ScriptToResourceState(ctx, v2)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// Delete archives a V2 script (they can't be deleted).
func (r *ScriptV2Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ScriptV2ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.ArchiveScript(ctx, landscape.ScriptIdPathParam(state.Id.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to archive script", err.Error())
	}
}

func (r *ScriptV2Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func v2ScriptToResourceState(ctx context.Context, v2Script landscape.V2Script) (ScriptV2ResourceModel, diag.Diagnostics) {
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

	return ScriptV2ResourceModel{
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
