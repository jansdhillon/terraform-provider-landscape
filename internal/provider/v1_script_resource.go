package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ resource.Resource = &V1ScriptResource{}
var _ resource.ResourceWithImportState = &V1ScriptResource{}

func NewV1ScriptResource() resource.Resource {
	return &V1ScriptResource{}
}

type V1ScriptResource struct {
	client *landscape.ClientWithResponses
}

type V1ScriptResourceModel v1ScriptResourceModel

type v1ScriptResourceModel struct {
	Id          types.Int64  `tfsdk:"id"`
	Title       types.String `tfsdk:"title"`
	AccessGroup types.String `tfsdk:"access_group"`
	Creator     types.Object `tfsdk:"creator"`
	Code        types.String `tfsdk:"code"`
	Status      types.String `tfsdk:"status"`
	Username    types.String `tfsdk:"username"`
	TimeLimit   types.Int64  `tfsdk:"time_limit"`
	Attachments types.List   `tfsdk:"attachments"`
}

var creatorType = map[string]attr.Type{
	"id":    types.NumberType,
	"name":  types.StringType,
	"email": types.StringType,
}

func (r *V1ScriptResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_v1_script"
}

func (r *V1ScriptResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":           schema.Int64Attribute{Computed: true},
			"title":        schema.StringAttribute{Computed: true, Optional: true},
			"access_group": schema.StringAttribute{Computed: true, Optional: true},
			"creator": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id":    schema.NumberAttribute{Computed: true},
					"name":  schema.StringAttribute{Computed: true},
					"email": schema.StringAttribute{Computed: true},
				},
			},
			"code":       schema.StringAttribute{Computed: true, Optional: true},
			"status":     schema.StringAttribute{Computed: true},
			"username":   schema.StringAttribute{Computed: true, Optional: true},
			"time_limit": schema.Int64Attribute{Computed: true, Optional: true},
			"attachments": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

func (r *V1ScriptResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*landscape.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError("unexpected provider data", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *V1ScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var title, codeAttr, username types.String
	var timeLimit types.Int64

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("title"), &title)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("code"), &codeAttr)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("username"), &username)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("time_limit"), &timeLimit)...)
	if resp.Diagnostics.HasError() {
		return
	}

	b64 := base64.StdEncoding.EncodeToString([]byte(codeAttr.ValueString()))
	params := landscape.LegacyActionParams("CreateScript")

	vals := url.Values{
		"title":       []string{title.ValueString()},
		"code":        []string{b64},
		"script_type": []string{"V1"},
	}

	if !timeLimit.IsUnknown() {
		vals.Set("time_limit", fmt.Sprint(timeLimit.ValueInt64()))
	}
	if !username.IsUnknown() {
		vals.Set("username", username.ValueString())
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

	idv := 0
	if v1.Creator.Id != nil {
		idv = *v1.Creator.Id
	}

	namev := ""
	if v1.Creator.Name != nil {
		namev = *v1.Creator.Name
	}

	emailv := ""
	if v1.Creator.Email != nil {
		emailv = fmt.Sprint(*v1.Creator.Email)
	}

	creatorObj, cd := types.ObjectValue(creatorType, map[string]attr.Value{
		"id":    types.NumberValue(big.NewFloat(float64(idv))),
		"name":  types.StringValue(namev),
		"email": types.StringValue(emailv),
	})
	resp.Diagnostics.Append(cd...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := V1ScriptResourceModel{
		Id:          types.Int64Value(int64(v1.Id)),
		Title:       types.StringValue(v1.Title),
		AccessGroup: ag,
		Creator:     creatorObj,
		Code:        types.StringValue(rawCode),
		Status:      types.StringValue(string(v1.Status)),
		Username:    u,
		TimeLimit:   tl,
		Attachments: types.ListNull(types.StringType),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *V1ScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var prev V1ScriptResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prev)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := prev.Id.ValueInt64()
	res, err := r.client.GetScriptWithResponse(ctx, landscape.ScriptIdPathParam(id))
	if err != nil {
		resp.Diagnostics.AddError("read failed", err.Error())
		return
	}
	if res.JSON200 == nil {
		resp.Diagnostics.AddError("missing script", "not found")
		return
	}

	v1, err := res.JSON200.AsV1Script()
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

	raw, err := codeRes.JSON200.AsLegacyScriptCode()
	if err != nil {
		resp.Diagnostics.AddError("decode failed", err.Error())
		return
	}

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

	idv := 0
	if v1.Creator.Id != nil {
		idv = *v1.Creator.Id
	}

	namev := ""
	if v1.Creator.Name != nil {
		namev = *v1.Creator.Name
	}

	emailv := ""
	if v1.Creator.Email != nil {
		emailv = fmt.Sprint(*v1.Creator.Email)
	}

	creatorObj, cd := types.ObjectValue(creatorType, map[string]attr.Value{
		"id":    types.NumberValue(big.NewFloat(float64(idv))),
		"name":  types.StringValue(namev),
		"email": types.StringValue(emailv),
	})
	resp.Diagnostics.Append(cd...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := v1ScriptResourceModel{
		Id:          types.Int64Value(int64(v1.Id)),
		Title:       types.StringValue(v1.Title),
		AccessGroup: ag,
		Creator:     creatorObj,
		Code:        types.StringValue(raw),
		Status:      types.StringValue(string(v1.Status)),
		Username:    u,
		TimeLimit:   tl,
		Attachments: types.ListNull(types.StringType),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *V1ScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan V1ScriptResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state V1ScriptResourceModel
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

	sr, err := r.client.GetScriptWithResponse(ctx, landscape.ScriptIdPathParam(state.Id.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("read failed", err.Error())
		return
	}
	if sr.JSON200 == nil {
		resp.Diagnostics.AddError("missing", "")
		return
	}

	v1, err := sr.JSON200.AsV1Script()
	if err != nil {
		resp.Diagnostics.AddError("bad v1", err.Error())
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

	raw, err := codeRes.JSON200.AsLegacyScriptCode()
	if err != nil {
		resp.Diagnostics.AddError("decode failed", err.Error())
		return
	}

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

	idv := 0
	if v1.Creator.Id != nil {
		idv = *v1.Creator.Id
	}

	namev := ""
	if v1.Creator.Name != nil {
		namev = *v1.Creator.Name
	}

	emailv := ""
	if v1.Creator.Email != nil {
		emailv = fmt.Sprint(*v1.Creator.Email)
	}

	creatorObj, cd := types.ObjectValue(creatorType, map[string]attr.Value{
		"id":    types.NumberValue(big.NewFloat(float64(idv))),
		"name":  types.StringValue(namev),
		"email": types.StringValue(emailv),
	})
	resp.Diagnostics.Append(cd...)
	if resp.Diagnostics.HasError() {
		return
	}

	newState := V1ScriptResourceModel{
		Id:          types.Int64Value(int64(v1.Id)),
		Title:       types.StringValue(v1.Title),
		AccessGroup: ag,
		Creator:     creatorObj,
		Code:        types.StringValue(raw),
		Status:      types.StringValue(string(v1.Status)),
		Username:    u,
		TimeLimit:   tl,
		Attachments: types.ListNull(types.StringType),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *V1ScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var x V1ScriptResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &x)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.InvokeLegacyAction(
		ctx,
		landscape.LegacyActionParams("RemoveScript"),
		landscape.EncodeQueryRequestEditor(url.Values{"script_id": []string{fmt.Sprint(x.Id.ValueInt64())}}),
	)
	if err != nil {
		resp.Diagnostics.AddError("delete failed", err.Error())
	}
}

func (r *V1ScriptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
