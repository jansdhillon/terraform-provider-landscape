// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ resource.Resource = &ScriptV1Resource{}
var _ resource.ResourceWithImportState = &ScriptV1Resource{}

func NewScriptV1Resource() resource.Resource {
	return &ScriptV1Resource{}
}

type ScriptV1Resource struct {
	client *landscape.ClientWithResponses
}

type ScriptV1ResourceModel struct {
	Id          types.Int64  `tfsdk:"id"`
	Title       types.String `tfsdk:"title"`
	AccessGroup types.String `tfsdk:"access_group"`
	Code        types.String `tfsdk:"code"`
	CreatedBy   types.Object `tfsdk:"created_by"`
	Status      types.String `tfsdk:"status"`
	Username    types.String `tfsdk:"username"`
	TimeLimit   types.Int64  `tfsdk:"time_limit"`
	Attachments types.List   `tfsdk:"attachments"`
}

func (r *ScriptV1Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script_v1"
}

func (r *ScriptV1Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "V1 (legacy) script resource",
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
				MarkdownDescription: "The raw script code content.",
			},
			"created_by": resourceschema.SingleNestedAttribute{
				MarkdownDescription: "The creator of the script.",
				Computed:            true,
				Attributes: map[string]resourceschema.Attribute{
					"id":    resourceschema.Int64Attribute{Computed: true},
					"name":  resourceschema.StringAttribute{Computed: true},
					"email": resourceschema.StringAttribute{Computed: true},
				},
			},
			"status": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the script (always 'V1' for legacy scripts).",
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
			"attachments": resourceschema.ListNestedAttribute{
				Computed: true,
				NestedObject: resourceschema.NestedAttributeObject{
					Attributes: map[string]resourceschema.Attribute{
						"filename": resourceschema.StringAttribute{Computed: true},
					},
				},
				MarkdownDescription: "Attachments associated with this script (filenames only for V1 scripts).",
			},
		},
	}
}

func (r *ScriptV1Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScriptV1Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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

	params := landscape.LegacyActionParams("CreateScript")
	vals := url.Values{
		"title":       []string{title.ValueString()},
		"code":        []string{base64.StdEncoding.EncodeToString([]byte(codeAttr.ValueString()))},
		"script_type": []string{"V1"},
	}

	if !timeLimit.IsNull() && !timeLimit.IsUnknown() {
		vals.Set("time_limit", fmt.Sprint(timeLimit.ValueInt64()))
	}
	if !username.IsNull() && !username.IsUnknown() {
		vals.Set("username", username.ValueString())
	}
	if !accessGroup.IsNull() && !accessGroup.IsUnknown() {
		vals.Set("access_group", accessGroup.ValueString())
	}

	editor := landscape.EncodeQueryRequestEditor(vals)
	res, err := r.client.InvokeLegacyActionWithResponse(ctx, params, editor)
	errTitle := "Failed to create V1 script"
	if err != nil {
		resp.Diagnostics.AddError(errTitle, err.Error())
		return
	}

	if res.JSON404 != nil {
		resp.Diagnostics.AddError(errTitle, *res.JSON404.Message)
		return
	}

	if res.JSON200 == nil {
		resp.Diagnostics.AddError(errTitle, *res.JSON400.Message)
		return
	}

	scriptRes, err := res.JSON200.AsScriptResult()
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse response as script", err.Error())
		return
	}

	v1Script, err := scriptRes.AsV1Script()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script as V1 script", err.Error())
		return
	}

	codeRes, err := r.client.InvokeLegacyActionWithResponse(
		ctx,
		landscape.LegacyActionParams("GetScriptCode"),
		landscape.EncodeQueryRequestEditor(url.Values{"script_id": []string{strconv.Itoa(v1Script.Id)}}),
	)
	errTitle = "Failed to get script code"
	if err != nil {
		resp.Diagnostics.AddError(errTitle, err.Error())
		return
	}

	if res.JSON200 == nil {
		if res.JSON404 != nil {
			resp.Diagnostics.AddError(errTitle, *res.JSON404.Message)
			return
		}

		resp.Diagnostics.AddError(errTitle, fmt.Sprintf("An error occurred getting the script code: %s", res.Status()))
		return
	}

	rawCode, err := codeRes.JSON200.AsLegacyScriptCode()
	if err != nil {
		resp.Diagnostics.AddError(errTitle, fmt.Sprintf("An error occurred getting the script code: %s", err))
		return
	}

	state, diags := v1ScriptToResourceState(ctx, v1Script, rawCode)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ScriptV1Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var current ScriptV1ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.GetScriptWithResponse(ctx, landscape.ScriptIdPathParam(current.Id.ValueInt64()))
	errTitle := "Failed to read script"
	if err != nil {
		resp.Diagnostics.AddError(errTitle, err.Error())
		return
	}

	if res.JSON404 != nil {
		resp.Diagnostics.AddError(errTitle, *res.JSON404.Message)
		return
	}

	if res.JSON200 == nil {
		resp.Diagnostics.AddError(errTitle, fmt.Sprintf("Error getting script: %s", res.Status()))
		return
	}

	v1Script, err := res.JSON200.AsV1Script()
	if err != nil {
		resp.Diagnostics.AddError(errTitle, fmt.Sprintf("Error getting script: %s", err))
		return
	}

	raw, diags := fetchV1Code(ctx, r.client, v1Script.Id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, stateDiags := v1ScriptToResourceState(ctx, v1Script, raw)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ScriptV1Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ScriptV1ResourceModel
	var state ScriptV1ResourceModel

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
	errTitle := "Update failed"
	if err != nil {
		resp.Diagnostics.AddError(errTitle, err.Error())
		return
	}

	if res.JSON404 != nil {
		resp.Diagnostics.AddError(errTitle, *res.JSON404.Message)
		return
	}

	if res.JSON400 != nil {
		resp.Diagnostics.AddError(errTitle, *res.JSON400.Message)
		return
	}

	if res.JSON200 == nil {
		resp.Diagnostics.AddError(errTitle, res.Status())
		return
	}

	scriptRes, err := res.JSON200.AsScriptResult()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script", fmt.Sprintf("Error getting script: %s", err))
		return
	}

	v1, err := scriptRes.AsV1Script()
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert script", fmt.Sprintf("Error getting script: %s", err))
		return
	}

	raw, diags := fetchV1Code(ctx, r.client, v1.Id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newState, stateDiags := v1ScriptToResourceState(ctx, v1, raw)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *ScriptV1Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ScriptV1ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Id.IsNull() || state.Id.IsUnknown() {
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

func (r *ScriptV1Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "No import ID was provided.")
		return
	}

	parsed, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Import ID '%s' is not a valid integer: %s", req.ID, err))
		return
	}

	v1AttachmentAttrType := map[string]attr.Type{
		"filename": types.StringType,
	}

	stateModel := ScriptV1ResourceModel{
		Id:          types.Int64Value(parsed),
		Title:       types.StringNull(),
		AccessGroup: types.StringNull(),
		Code:        types.StringNull(),
		CreatedBy:   types.ObjectNull(createdByAttrTypes),
		Status:      types.StringNull(),
		Username:    types.StringNull(),
		TimeLimit:   types.Int64Null(),
		Attachments: types.ListNull(types.ObjectType{AttrTypes: v1AttachmentAttrType}),
	}

	diags := resp.State.Set(ctx, stateModel)
	resp.Diagnostics.Append(diags...)
}

func v1ScriptToResourceState(_ context.Context, v1 landscape.V1Script, rawCode string) (ScriptV1ResourceModel, diag.Diagnostics) {
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

	var creatorName string
	if v1.Creator.Name != nil {
		creatorName = *v1.Creator.Name
	}

	var creatorEmail string
	if v1.Creator.Email != nil {
		creatorEmail = fmt.Sprint(*v1.Creator.Email)
	}

	creatorObj, cd := types.ObjectValue(createdByAttrTypes, map[string]attr.Value{
		"id":    types.Int64PointerValue(int64Ptr(int64(creatorId))),
		"name":  types.StringValue(creatorName),
		"email": types.StringValue(creatorEmail),
	})
	diags.Append(cd...)

	v1AttachmentAttrType := map[string]attr.Type{
		"filename": types.StringType,
	}

	attachments := types.ListNull(types.ObjectType{AttrTypes: v1AttachmentAttrType})
	if v1.Attachments != nil {
		elems := make([]attr.Value, 0, len(*v1.Attachments))
		for _, filename := range *v1.Attachments {
			elem, d := types.ObjectValue(v1AttachmentAttrType, map[string]attr.Value{
				"filename": types.StringValue(filename),
			})
			diags.Append(d...)
			elems = append(elems, elem)
		}
		if !diags.HasError() {
			list, d := types.ListValue(types.ObjectType{AttrTypes: v1AttachmentAttrType}, elems)
			diags.Append(d...)
			if !diags.HasError() {
				attachments = list
			}
		}
	}

	return ScriptV1ResourceModel{
		Id:          types.Int64Value(int64(v1.Id)),
		Title:       types.StringValue(v1.Title),
		AccessGroup: ag,
		Code:        types.StringValue(rawCode),
		CreatedBy:   creatorObj,
		Status:      types.StringValue(string(v1.Status)),
		Username:    u,
		TimeLimit:   tl,
		Attachments: attachments,
	}, diags
}
