// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ datasource.DataSource = &ScriptProfileDataSource{}
var _ datasource.DataSourceWithConfigure = &ScriptProfileDataSource{}

func NewScriptProfileDataSource() datasource.DataSource {
	return &ScriptProfileDataSource{}
}

type ScriptProfileDataSource struct {
	client *landscape.ClientWithResponses
}

type ScriptProfileDataSourceModel struct {
	Id           types.Int64  `tfsdk:"id"`
	Title        types.String `tfsdk:"title"`
	ScriptId     types.Int64  `tfsdk:"script_id"`
	AccessGroup  types.String `tfsdk:"access_group"`
	Username     types.String `tfsdk:"username"`
	TimeLimit    types.Int64  `tfsdk:"time_limit"`
	AllComputers types.Bool   `tfsdk:"all_computers"`
	Tags         types.Set    `tfsdk:"tags"`
	Archived     types.Bool   `tfsdk:"archived"`
	CreatedAt    types.String `tfsdk:"created_at"`
	LastEditedAt types.String `tfsdk:"last_edited_at"`
	Trigger      types.Object `tfsdk:"trigger"`
}

func (d *ScriptProfileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script_profile"
}

func (d *ScriptProfileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Landscape script profile by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The unique identifier for the script profile.",
			},
			"title": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The title of the script profile.",
			},
			"script_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The ID of the script this profile executes.",
			},
			"access_group": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The access group associated with the script profile.",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Linux username under which the script runs.",
			},
			"time_limit": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Maximum execution time for the script in seconds.",
			},
			"all_computers": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script profile targets all computers in the account.",
			},
			"tags": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of tags used to target specific computers.",
			},
			"archived": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script profile has been archived.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the script profile was created (RFC3339).",
			},
			"last_edited_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the script profile was last modified (RFC3339).",
			},
			"trigger": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The trigger that controls when the script profile executes.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Trigger type: `event`, `recurring`, or `one_time`.",
					},
					"event_type": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "For `event` triggers: the event type (e.g. `post_enrollment`).",
					},
					"interval": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "For `recurring` triggers: cron expression.",
					},
					"start_after": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "For `recurring` triggers: datetime after which the schedule begins (RFC3339).",
					},
					"timestamp": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "For `one_time` triggers: datetime at which the profile executes (RFC3339).",
					},
				},
			},
		},
	}
}

func (d *ScriptProfileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*landscape.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *landscape.ClientWithResponses, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *ScriptProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ScriptProfileDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := d.client.GetScriptProfileWithResponse(ctx, int(config.Id.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script profile", err.Error())
		return
	}
	if res.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read script profile", res.Status())
		return
	}

	detail := res.JSON200
	sortedTags := make([]string, len(detail.Tags))
	copy(sortedTags, detail.Tags)
	sort.Strings(sortedTags)
	tags, diags := types.SetValueFrom(ctx, types.StringType, sortedTags)
	resp.Diagnostics.Append(diags...)

	triggerObj, diags := triggerResponseToObject(detail.Trigger)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := ScriptProfileDataSourceModel{
		Id:           types.Int64Value(int64(detail.Id)),
		Title:        types.StringValue(detail.Title),
		ScriptId:     types.Int64Value(int64(detail.ScriptId)),
		AccessGroup:  types.StringValue(detail.AccessGroup),
		Username:     types.StringValue(detail.Username),
		TimeLimit:    types.Int64Value(int64(detail.TimeLimit)),
		AllComputers: types.BoolValue(detail.AllComputers),
		Tags:         tags,
		Archived:     types.BoolValue(detail.Archived),
		CreatedAt:    types.StringValue(detail.CreatedAt),
		LastEditedAt: types.StringValue(detail.LastEditedAt),
		Trigger:      triggerObj,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
