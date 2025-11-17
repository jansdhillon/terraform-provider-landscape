// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ScriptResource{}
var _ resource.ResourceWithImportState = &ScriptResource{}

func NewScriptResource() resource.Resource {
	return &ScriptResource{}
}

// ScriptResource defines the resource implementation.
type ScriptResource struct {
	client *landscape.ClientWithResponses
}

// ScriptResourceModel represents either a V1 or V2 script in state.
type ScriptResourceModel struct {
	Id             types.Int64  `tfsdk:"id"`
	Title          types.String `tfsdk:"title"`
	AccessGroup    types.String `tfsdk:"access_group"`
	ScriptType     types.String `tfsdk:"script_type"`
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

var v2LastEditedByAttrTypes = map[string]attr.Type{
	"id":   types.Int64Type,
	"name": types.StringType,
}

var createdByAttrTypes = map[string]attr.Type{
	"id":    types.Int64Type,
	"name":  types.StringType,
	"email": types.StringType,
}

var scriptProfileAttrType = map[string]attr.Type{
	"id":    types.Int64Type,
	"title": types.StringType,
}

var scriptAttachmentAttrType = map[string]attr.Type{
	"id":       types.Int64Type,
	"filename": types.StringType,
}

// scriptCreateOpts defines script creation options.
type scriptCreateOpts struct {
	Title           string
	CodeB64         string
	Username        *string
	TimeLimit       *int64
	ScriptType      string
	AccessGroup     *string
	StateScriptType string
}

func newScriptCreateOpts(title, codeAttr, username types.String, timeLimit types.Int64, scriptType, accessGroup types.String) scriptCreateOpts {
	return scriptCreateOpts{
		Title:           title.ValueString(),
		CodeB64:         base64.StdEncoding.EncodeToString([]byte(codeAttr.ValueString())),
		Username:        username.ValueStringPointer(),
		TimeLimit:       timeLimit.ValueInt64Pointer(),
		ScriptType:      scriptType.ValueString(),
		AccessGroup:     accessGroup.ValueStringPointer(),
		StateScriptType: scriptType.ValueString(),
	}
}

// scriptResourceSchema defines the schema for the V2 script resource.
var scriptResourceSchema = resourceschema.Schema{
	MarkdownDescription: "Script resource",
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
			MarkdownDescription: "The raw script code content. Note that this does NOT split on the interpreter/executable portion.",
		},
		"created_at": resourceschema.StringAttribute{
			Computed:            true,
			Optional:            true,
			MarkdownDescription: "When the script was created. Not applicable for V1 scripts.",
		},
		"created_by": resourceschema.SingleNestedAttribute{
			MarkdownDescription: "The creator of the script. Note that only V1 scripts have an email.",
			Computed:            true,
			Optional:            true,
			Attributes: map[string]resourceschema.Attribute{
				"id":    resourceschema.NumberAttribute{Computed: true},
				"name":  resourceschema.StringAttribute{Computed: true},
				"email": resourceschema.StringAttribute{Computed: true, Optional: true},
			},
		},
		"last_edited_at": resourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "When the script was last edited. Not applicable for V1 scripts.",
			Optional:            true,
		},
		"status": resourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The status of the script (active, archived, or redacted), or V1 for legacy scripts.",
		},
		"version_number": resourceschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "The version number of the script. Note that V1 scripts are unversioned.",
		},
		"username": resourceschema.StringAttribute{
			Computed:            true,
			Optional:            true,
			MarkdownDescription: "The Linux user that will run the script on the Landscape Client instance.",
		},
		"time_limit": resourceschema.Int64Attribute{
			Computed:            true,
			Optional:            true,
			MarkdownDescription: "The time limit in second for a script to complete successfully.",
		},
		"is_editable": resourceschema.BoolAttribute{
			Computed:            true,
			Optional:            true,
			MarkdownDescription: "Whether the script is editable by the caller. Not applicable for V1 scripts.",
		},
		"is_executable": resourceschema.BoolAttribute{
			Computed:            true,
			Optional:            true,
			MarkdownDescription: "Whether the script is executable by the caller. Not applicable for V1 scripts.",
		},
		"is_redactable": resourceschema.BoolAttribute{
			Computed:            true,
			Optional:            true,
			MarkdownDescription: "Whether the script is redactable by the caller. Not applicable for V1 scripts.",
		},
		"last_edited_by": resourceschema.SingleNestedAttribute{
			Computed: true,
			Optional: true,
			Attributes: map[string]resourceschema.Attribute{
				"id":   resourceschema.NumberAttribute{Computed: true, Optional: true},
				"name": resourceschema.StringAttribute{Computed: true, Optional: true},
			},
			MarkdownDescription: "The (Landscape) user who last edited the script.",
		},
		"attachments": resourceschema.ListNestedAttribute{
			Computed: true,
			Optional: true,
			NestedObject: resourceschema.NestedAttributeObject{
				Attributes: map[string]resourceschema.Attribute{
					"id":       resourceschema.NumberAttribute{Computed: true, Optional: true},
					"filename": resourceschema.StringAttribute{Computed: true},
				},
			},
			MarkdownDescription: "Attachments associated with this script. IDs of the attachments are only returned for V2+ scripts.",
		},
		"script_profiles": resourceschema.ListNestedAttribute{
			Computed: true,
			Optional: true,
			NestedObject: resourceschema.NestedAttributeObject{
				Attributes: map[string]resourceschema.Attribute{
					"id":    resourceschema.NumberAttribute{Computed: true},
					"title": resourceschema.StringAttribute{Computed: true},
				},
			},
			MarkdownDescription: "List of script profiles for V2+ scripts.",
		},
		"script_type": resourceschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("V1"),
			MarkdownDescription: "The script vesrsion indicator (V1 for legacy, V2 for modern scripts).",
		},
	},
}

func (r *ScriptResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script"
}

func (r *ScriptResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = scriptResourceSchema
}

func (r *ScriptResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*landscape.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var title types.String
	var codeAttr types.String
	var username types.String
	var timeLimit types.Int64
	var gotScriptType types.String
	var rawScriptType string
	var stateScriptType string
	var accessGroup types.String

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("title"), &title)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("code"), &codeAttr)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("username"), &username)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("time_limit"), &timeLimit)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("script_type"), &gotScriptType)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("access_group"), &accessGroup)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("script_type: %s", gotScriptType))

	if title.IsNull() || title.IsUnknown() {
		resp.Diagnostics.AddError("Missing title", "`title` must be set.")
		return
	}

	if codeAttr.IsNull() || codeAttr.IsUnknown() {
		resp.Diagnostics.AddError("Missing code", "`code` must be set.")
		return
	}

	if gotScriptType.IsNull() || gotScriptType.IsUnknown() {
		resp.Diagnostics.AddError("Missing script type", "`script_type` must be set.")
		return
	}

	stateScriptType = gotScriptType.ValueString()
	rawScriptType = strings.ToUpper(stateScriptType)
	opts := newScriptCreateOpts(title, codeAttr, username, timeLimit, gotScriptType, accessGroup)

	switch rawScriptType {
	case "V2":
		r.createV2(ctx, resp, opts)
	case "V1":
		r.createV1(ctx, resp, opts)
	default:
		resp.Diagnostics.AddError("Invalid script type", "`script_type` must be either V1 or V2.")
	}
}

func (r *ScriptResource) createV2(ctx context.Context, resp *resource.CreateResponse, opts scriptCreateOpts) {
	vals := url.Values{
		"title":       []string{opts.Title},
		"code":        []string{opts.CodeB64},
		"script_type": []string{opts.ScriptType},
	}

	if opts.TimeLimit != nil {
		vals.Add("time_limit", fmt.Sprint(*opts.TimeLimit))
	}
	if opts.Username != nil {
		vals.Add("username", *opts.Username)
	}
	if opts.AccessGroup != nil {
		vals.Add("access_group", *opts.AccessGroup)
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

	v2, err := createRes.JSON200.AsV2Script()
	if err != nil {
		resp.Diagnostics.AddError("Failed to decode V2 script", err.Error())
		return
	}

	state, diags := v2ScriptToState(ctx, v2, opts.StateScriptType)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ScriptResource) createV1(ctx context.Context, resp *resource.CreateResponse, opts scriptCreateOpts) {
	params := landscape.LegacyActionParams("CreateScript")

	vals := url.Values{
		"title":       []string{opts.Title},
		"code":        []string{opts.CodeB64},
		"script_type": []string{opts.ScriptType},
	}

	if opts.TimeLimit != nil {
		vals.Set("time_limit", fmt.Sprint(*opts.TimeLimit))
	}
	if opts.Username != nil {
		vals.Set("username", *opts.Username)
	}
	if opts.AccessGroup != nil {
		vals.Set("access_group", *opts.AccessGroup)
	}

	editor := landscape.EncodeQueryRequestEditor(vals)
	cre, err := r.client.InvokeLegacyActionWithResponse(ctx, params, editor)
	if err != nil {
		resp.Diagnostics.AddError("create failed", err.Error())
		return
	}
	if cre.JSON200 == nil {
		resp.Diagnostics.AddError("create failed", *cre.JSON400.Message)
		return
	}

	v1, err := cre.JSON200.AsV1Script()
	if err != nil {
		resp.Diagnostics.AddError("invalid V1", err.Error())
		return
	}

	codeRes, err := r.client.InvokeLegacyActionWithResponse(
		ctx,
		landscape.LegacyActionParams("GetScriptCode"),
		landscape.EncodeQueryRequestEditor(url.Values{"script_id": []string{strconv.Itoa(v1.Id)}}),
	)
	if err != nil {
		resp.Diagnostics.AddError("code fetch failed", err.Error())
		return
	}

	rawCode, err := codeRes.JSON200.AsLegacyScriptCode()
	if err != nil {
		resp.Diagnostics.AddError("decode failed", err.Error())
		return
	}

	state, diags := v1ScriptToState(ctx, v1, rawCode, opts.StateScriptType)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var current ScriptResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if current.Id.IsNull() || current.Id.IsUnknown() {
		resp.Diagnostics.AddError("Missing script ID", "The `id` attribute must be set in state to read a script.")
		return
	}

	state, diags := r.readScript(ctx, current.Id.ValueInt64(), current.ScriptType.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ScriptResourceModel
	var state ScriptResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if state.Id.IsNull() || state.Id.IsUnknown() {
		resp.Diagnostics.AddError("Missing script ID", "The `id` attribute must be set in state to update a script.")
		return
	}

	if !state.Status.IsUnknown() && !state.Status.IsNull() && strings.EqualFold(state.Status.ValueString(), "V1") {
		r.updateV1(ctx, plan, state, resp)
		return
	}

	vals := url.Values{
		"script_id": []string{strconv.FormatInt(state.Id.ValueInt64(), 10)},
	}

	if !plan.Title.IsUnknown() && !plan.Title.IsNull() {
		vals.Add("title", plan.Title.ValueString())
	}

	if !plan.Code.IsUnknown() && !plan.Code.IsNull() {
		vals.Add("code", base64.StdEncoding.EncodeToString([]byte(plan.Code.ValueString())))
	}

	if !plan.Username.IsUnknown() && !plan.Username.IsNull() {
		vals.Add("username", plan.Username.ValueString())
	}

	if !plan.TimeLimit.IsUnknown() && !plan.TimeLimit.IsNull() {
		vals.Add("time_limit", fmt.Sprint(plan.TimeLimit.ValueInt64()))
	}
	if !plan.AccessGroup.IsUnknown() && !plan.AccessGroup.IsNull() {
		vals.Add("access_group", plan.AccessGroup.ValueString())
	}

	editRes, err := r.client.InvokeLegacyActionWithResponse(ctx, landscape.LegacyActionParams("EditScript"), landscape.EncodeQueryRequestEditor(vals))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update script", err.Error())
		return
	}

	if editRes.JSON200 == nil {
		errMsg := "Unexpected error updating script"
		if editRes.JSON400 != nil && editRes.JSON400.Message != nil {
			errMsg = fmt.Sprintf("%s: %s", errMsg, *editRes.JSON400.Message)
		} else if len(editRes.Body) > 0 {
			errMsg = fmt.Sprintf("%s: %s", errMsg, string(editRes.Body))
		}
		resp.Diagnostics.AddError("Failed to update script", errMsg)
		return
	}

	v2, err := editRes.JSON200.AsV2Script()
	if err != nil {
		resp.Diagnostics.AddError("Failed to decode V2 script", err.Error())
		return
	}

	newState, diags := v2ScriptToState(ctx, v2, state.ScriptType.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *ScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ScriptResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Id.IsNull() || state.Id.IsUnknown() {
		return
	}

	if state.Status != types.StringValue("V1") {
		return
	}

	vals := url.Values{
		"script_id": []string{fmt.Sprint(state.Id.ValueInt64())},
	}

	_, err := r.client.InvokeLegacyAction(ctx, landscape.LegacyActionParams("RemoveScript"), landscape.EncodeQueryRequestEditor(vals))
	if err != nil {
		resp.Diagnostics.AddError("Failed to remove script", err.Error())
	}
}

func (r *ScriptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ScriptResource) updateV1(ctx context.Context, plan, state ScriptResourceModel, resp *resource.UpdateResponse) {
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
	if !plan.Code.IsUnknown() && !plan.Code.IsNull() {
		b64 := base64.StdEncoding.EncodeToString([]byte(plan.Code.ValueString()))
		vals.Set("code", b64)
	}

	editor := landscape.EncodeQueryRequestEditor(vals)
	_, err := r.client.InvokeLegacyAction(ctx, landscape.LegacyActionParams("EditScript"), editor)
	if err != nil {
		resp.Diagnostics.AddError("update failed", err.Error())
		return
	}

	newState, diags := r.readScript(ctx, state.Id.ValueInt64(), state.ScriptType.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *ScriptResource) readScript(ctx context.Context, id int64, stateScriptType string) (*ScriptResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	scriptRes, err := r.client.GetScriptWithResponse(ctx, landscape.ScriptIdPathParam(id))
	if err != nil {
		diags.AddError("Failed to read script", err.Error())
		return nil, diags
	}

	if scriptRes.JSON200 == nil {
		diags.AddError("Failed to read script", "Script not found")
		return nil, diags
	}

	// Determine which conversion to try first based on the type stored in state.
	// This avoids trying to coerce a V1 script into the V2 shape (which can yield
	// a nil code/interpreter and an unexpected null value in state).
	if strings.EqualFold(stateScriptType, "V1") {
		if v1, err := scriptRes.JSON200.AsV1Script(); err == nil {
			state, stateDiags := v1ToState(ctx, r.client, v1, stateScriptType)
			diags.Append(stateDiags...)
			return &state, diags
		}

		if v2, err := scriptRes.JSON200.AsV2Script(); err == nil {
			state, stateDiags := v2ScriptToState(ctx, v2, stateScriptType)
			diags.Append(stateDiags...)
			return &state, diags
		}
	} else {
		if v2, err := scriptRes.JSON200.AsV2Script(); err == nil {
			state, stateDiags := v2ScriptToState(ctx, v2, stateScriptType)
			diags.Append(stateDiags...)
			return &state, diags
		}

		if v1, err := scriptRes.JSON200.AsV1Script(); err == nil {
			state, stateDiags := v1ToState(ctx, r.client, v1, stateScriptType)
			diags.Append(stateDiags...)
			return &state, diags
		}
	}

	diags.AddError("Failed to convert script", "Could not convert returned script into V1 or V2 form")
	return nil, diags
}

func fetchV1Code(ctx context.Context, client *landscape.ClientWithResponses, id int) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	codeRes, err := client.InvokeLegacyActionWithResponse(
		ctx,
		landscape.LegacyActionParams("GetScriptCode"),
		landscape.EncodeQueryRequestEditor(url.Values{"script_id": []string{strconv.Itoa(id)}}),
	)
	if err != nil {
		diags.AddError("code fetch failed", err.Error())
		return "", diags
	}

	raw, err := codeRes.JSON200.AsLegacyScriptCode()
	if err != nil {
		diags.AddError("decode failed", err.Error())
		return "", diags
	}

	return raw, diags
}

func int64Ptr(i int64) *int64 {
	return &i
}

func v1ToState(ctx context.Context, client *landscape.ClientWithResponses, v1 landscape.V1Script, stateScriptType string) (ScriptResourceModel, diag.Diagnostics) {
	raw, diags := fetchV1Code(ctx, client, v1.Id)
	if diags.HasError() {
		return ScriptResourceModel{}, diags
	}
	return v1ScriptToState(ctx, v1, raw, stateScriptType)
}

func v1ScriptToState(_ context.Context, v1 landscape.V1Script, rawCode string, stateScriptType string) (ScriptResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	ag := types.StringNull()
	if v1.AccessGroup != nil {
		ag = types.StringValue(*v1.AccessGroup)
	}

	u := types.StringNull()
	if v1.Username != nil {
		u = types.StringValue(*v1.Username)
	}

	tl := types.Int64Null()
	if v1.TimeLimit != nil {
		tl = types.Int64Value(int64(*v1.TimeLimit))
	}

	var creatorId int
	if v1.Creator.Id != nil {
		creatorId = *v1.Creator.Id
	}

	namev := ""
	if v1.Creator.Name != nil {
		namev = *v1.Creator.Name
	}

	emailv := ""
	if v1.Creator.Email != nil {
		emailv = fmt.Sprint(*v1.Creator.Email)
	}

	creatorObj, cd := types.ObjectValue(createdByAttrTypes, map[string]attr.Value{
		"id":    types.Int64PointerValue(int64Ptr(int64(creatorId))),
		"name":  types.StringValue(namev),
		"email": types.StringValue(emailv),
	})
	diags.Append(cd...)

	attachments := types.ListNull(types.ObjectType{AttrTypes: scriptAttachmentAttrType})
	if v1.Attachments != nil {
		elems := make([]attr.Value, 0, len(*v1.Attachments))
		for _, filename := range *v1.Attachments {
			elem, d := types.ObjectValue(scriptAttachmentAttrType, map[string]attr.Value{
				"id":       types.NumberNull(),
				"filename": types.StringValue(filename),
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

	return ScriptResourceModel{
		Id:             types.Int64Value(int64(v1.Id)),
		Title:          types.StringValue(v1.Title),
		AccessGroup:    ag,
		ScriptType:     types.StringValue(stateScriptType),
		Code:           types.StringValue(rawCode),
		CreatedAt:      types.StringNull(),
		CreatedBy:      creatorObj,
		LastEditedAt:   types.StringNull(),
		Status:         types.StringValue(string(v1.Status)),
		VersionNumber:  types.Int64Null(),
		Username:       u,
		TimeLimit:      tl,
		IsEditable:     types.BoolNull(),
		IsExecutable:   types.BoolNull(),
		IsRedactable:   types.BoolNull(),
		LastEditedBy:   types.ObjectNull(v2LastEditedByAttrTypes),
		Attachments:    attachments,
		ScriptProfiles: types.ListNull(types.ObjectType{AttrTypes: scriptProfileAttrType}),
	}, diags
}

func v2ScriptToState(ctx context.Context, v2Script landscape.V2Script, stateScriptType string) (ScriptResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	createdBy := types.ObjectNull(createdByAttrTypes)
	if v2Script.CreatedBy != nil {
		obj, d := types.ObjectValue(createdByAttrTypes, map[string]attr.Value{
			"id":    types.Int64Value(int64(*v2Script.CreatedBy.Id)),
			"name":  types.StringPointerValue(v2Script.CreatedBy.Name),
			"email": types.StringNull(),
		})
		diags.Append(d...)
		if !diags.HasError() {
			createdBy = obj
		} else {
			tflog.Debug(ctx, "Couldn't convert script create_by field into an object")
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

	// NOTE: Attachments for V1 and V2+ scripts are different. V1 script attachments
	// are just a list of filenames, while modern attachments are objects with an ID and
	// the filenames. Due to limitations around creating dynamic nested objects with the TFSDK,
	//	we just create them both as objects and omit the ID for legacy scripts.
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
				"id":    types.NumberValue(big.NewFloat(float64(sp.Id))),
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

	return ScriptResourceModel{
		Id:             types.Int64Value(int64(v2Script.Id)),
		Title:          types.StringValue(v2Script.Title),
		AccessGroup:    types.StringPointerValue(v2Script.AccessGroup),
		ScriptType:     types.StringValue(stateScriptType),
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
