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
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ datasource.DataSource = &ScriptV1DataSource{}
var _ datasource.DataSourceWithConfigure = &ScriptV1DataSource{}

func NewScriptV1DataSource() datasource.DataSource {
	return &ScriptV1DataSource{}
}

type ScriptV1DataSource struct {
	client *landscape.ClientWithResponses
}

type ScriptV1DataSourceModel struct {
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

func (d *ScriptV1DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script_v1"
}

func (d *ScriptV1DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "V1 (legacy) script data source",
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
				MarkdownDescription: "The raw script code content.",
			},
			"created_by": schema.SingleNestedAttribute{
				MarkdownDescription: "The creator of the script.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"id":    schema.Int64Attribute{Computed: true},
					"name":  schema.StringAttribute{Computed: true},
					"email": schema.StringAttribute{Computed: true},
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the script (always 'V1' for legacy scripts).",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Linux user that will run the script on the Landscape Client instance.",
			},
			"time_limit": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The time limit in seconds for a script to complete successfully.",
			},
			"attachments": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"filename": schema.StringAttribute{Computed: true},
					},
				},
				MarkdownDescription: "Attachments associated with this script (filenames only for V1 scripts).",
			},
		},
	}
}

func (d *ScriptV1DataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScriptV1DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var idValue types.Int64

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &idValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if idValue.IsUnknown() || idValue.IsNull() {
		resp.Diagnostics.AddError("Missing script ID", "The `id` attribute must be provided for the landscape_script_v1 data source.")
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

	v1Script, err := scriptRes.JSON200.AsV1Script()
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert script", "The script is not a V1 script. Use landscape_script_v2 data source instead.")
		return
	}

	state, diags := v1ScriptToDataSourceState(ctx, d.client, v1Script)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func v1ScriptToDataSourceState(ctx context.Context, client *landscape.ClientWithResponses, v1 landscape.V1Script) (ScriptV1DataSourceModel, diag.Diagnostics) {
	raw, diags := fetchV1Code(ctx, client, v1.Id)
	if diags.HasError() {
		return ScriptV1DataSourceModel{}, diags
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

	return ScriptV1DataSourceModel{
		Id:          types.Int64Value(int64(v1.Id)),
		Title:       types.StringValue(v1.Title),
		AccessGroup: ag,
		Code:        types.StringValue(raw),
		CreatedBy:   creatorObj,
		Status:      types.StringValue(string(v1.Status)),
		Username:    u,
		TimeLimit:   tl,
		Attachments: attachments,
	}, diags
}
