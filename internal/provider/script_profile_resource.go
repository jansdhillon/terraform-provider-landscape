// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ resource.Resource = &ScriptProfileResource{}
var _ resource.ResourceWithImportState = &ScriptProfileResource{}

func NewScriptProfileResource() resource.Resource {
	return &ScriptProfileResource{}
}

type ScriptProfileResource struct {
	client *landscape.ClientWithResponses
}

// triggerAttrTypes is the Terraform attribute type map for the trigger block.
var triggerAttrTypes = map[string]attr.Type{
	"type":        types.StringType,
	"event_type":  types.StringType,
	"interval":    types.StringType,
	"start_after": types.StringType,
	"timestamp":   types.StringType,
}

type ScriptProfileResourceModel struct {
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

func (r *ScriptProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script_profile"
}

func (r *ScriptProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "Manages a Landscape script profile. A script profile defines when and how a V2 script is executed across targeted computers.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier for the script profile.",
			},
			"title": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The title of the script profile.",
			},
			"script_id": resourceschema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The ID of the V2 script this profile executes.",
			},
			"username": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Linux username under which the script will run.",
			},
			"time_limit": resourceschema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Maximum execution time for the script in seconds.",
			},
			"all_computers": resourceschema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the script profile targets all computers in the account.",
			},
			"tags": resourceschema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of tags used to target specific computers.",
			},
			"access_group": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The access group associated with the script profile.",
			},
			"archived": resourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script profile has been archived.",
			},
			"created_at": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the script profile was created.",
			},
			"last_edited_at": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the script profile was last modified.",
			},
			"trigger": resourceschema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "The trigger that controls when the script profile executes.",
				Attributes: map[string]resourceschema.Attribute{
					"type": resourceschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Trigger type: `event`, `recurring`, or `one_time`.",
					},
					"event_type": resourceschema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "For `event` triggers: the event type (e.g. `post_enrollment`).",
					},
					"interval": resourceschema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "For `recurring` triggers: cron expression (e.g. `0 * * * *`).",
					},
					"start_after": resourceschema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "For `recurring` triggers: RFC3339 datetime after which the schedule begins.",
					},
					"timestamp": resourceschema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "For `one_time` triggers: RFC3339 datetime at which the profile executes.",
					},
				},
			},
		},
	}
}

func (r *ScriptProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*landscape.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *landscape.ClientWithResponses, got: %T.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *ScriptProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ScriptProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := planToCreateBody(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.CreateScriptProfileWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create script profile", err.Error())
		return
	}
	if res.JSON201 == nil {
		resp.Diagnostics.AddError("Failed to create script profile", errorFromCreateResp(res))
		return
	}

	state, diags := scriptProfileDetailToState(ctx, res.JSON201)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ScriptProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ScriptProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.GetScriptProfileWithResponse(ctx, int(state.Id.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script profile", err.Error())
		return
	}
	if res.JSON200 == nil {
		if res.StatusCode() == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read script profile", res.Status())
		return
	}

	newState, diags := scriptProfileDetailToState(ctx, res.JSON200)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *ScriptProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ScriptProfileResourceModel
	var state ScriptProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	patch, diags := planToPatchBody(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.UpdateScriptProfileWithResponse(ctx, int(state.Id.ValueInt64()), patch)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update script profile", err.Error())
		return
	}
	if res.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to update script profile", res.Status())
		return
	}

	newState, diags := scriptProfileDetailToState(ctx, res.JSON200)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// Delete archives the script profile (Landscape has no hard delete for profiles).
func (r *ScriptProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ScriptProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.ArchiveScriptProfileWithResponse(ctx, int(state.Id.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to archive script profile", err.Error())
		return
	}
	if res.StatusCode() != 204 && res.StatusCode() != 404 {
		resp.Diagnostics.AddError("Failed to archive script profile", res.Status())
	}
}

func (r *ScriptProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected a numeric script profile ID, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func planToCreateBody(ctx context.Context, plan ScriptProfileResourceModel) (landscape.ScriptProfileCreateBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	trigger, d := planTriggerToCreateRequest(ctx, plan.Trigger)
	diags.Append(d...)

	var tags []string
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		diags.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
	} else {
		tags = []string{}
	}

	allComputers := plan.AllComputers.ValueBool()

	return landscape.ScriptProfileCreateBody{
		Title:        plan.Title.ValueString(),
		ScriptId:     int(plan.ScriptId.ValueInt64()),
		Username:     plan.Username.ValueString(),
		TimeLimit:    int(plan.TimeLimit.ValueInt64()),
		AllComputers: &allComputers,
		Tags:         &tags,
		Trigger:      trigger,
	}, diags
}

func planToPatchBody(ctx context.Context, plan ScriptProfileResourceModel) (landscape.ScriptProfilePatchBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	title := plan.Title.ValueString()
	username := plan.Username.ValueString()
	timeLimit := int(plan.TimeLimit.ValueInt64())
	allComputers := plan.AllComputers.ValueBool()

	var tags []string
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		diags.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
	}

	patchTrigger, d := planTriggerToPatchRequest(ctx, plan.Trigger)
	diags.Append(d...)

	return landscape.ScriptProfilePatchBody{
		Title:        &title,
		Username:     &username,
		TimeLimit:    &timeLimit,
		AllComputers: &allComputers,
		Tags:         &tags,
		Trigger:      patchTrigger,
	}, diags
}

func planTriggerToCreateRequest(_ context.Context, triggerObj types.Object) (landscape.ScriptProfileTriggerCreateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	var t landscape.ScriptProfileTriggerCreateRequest

	attrs := triggerObj.Attributes()
	triggerType := attrs["type"].(types.String).ValueString() //nolint:forcetypeassert

	switch triggerType {
	case "event":
		eventType := attrs["event_type"].(types.String).ValueString() //nolint:forcetypeassert
		err := t.FromScriptProfileEventTrigger(landscape.ScriptProfileEventTrigger{
			TriggerType: landscape.Event,
			EventType:   landscape.ScriptProfileEventType(eventType),
		})
		if err != nil {
			diags.AddError("Failed to build event trigger", err.Error())
		}
	case "recurring":
		interval := attrs["interval"].(types.String).ValueString()         //nolint:forcetypeassert
		startAfterStr := attrs["start_after"].(types.String).ValueString() //nolint:forcetypeassert
		startAfter, err := time.Parse(time.RFC3339, startAfterStr)
		if err != nil {
			diags.AddError("Invalid start_after", fmt.Sprintf("Must be RFC3339: %s", err))
			return t, diags
		}
		err = t.FromScriptProfileScheduleDraftTrigger(landscape.ScriptProfileScheduleDraftTrigger{
			TriggerType: landscape.ScriptProfileScheduleDraftTriggerTriggerTypeRecurring,
			Interval:    interval,
			StartAfter:  startAfter,
		})
		if err != nil {
			diags.AddError("Failed to build recurring trigger", err.Error())
		}
	case "one_time":
		tsStr := attrs["timestamp"].(types.String).ValueString() //nolint:forcetypeassert
		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			diags.AddError("Invalid timestamp", fmt.Sprintf("Must be RFC3339: %s", err))
			return t, diags
		}
		err = t.FromScriptProfileOneTimeDraftTrigger(landscape.ScriptProfileOneTimeDraftTrigger{
			TriggerType: landscape.ScriptProfileOneTimeDraftTriggerTriggerTypeOneTime,
			Timestamp:   ts,
		})
		if err != nil {
			diags.AddError("Failed to build one_time trigger", err.Error())
		}
	default:
		diags.AddError("Invalid trigger type", fmt.Sprintf("Must be one of: event, recurring, one_time. Got: %s", triggerType))
	}
	return t, diags
}

func planTriggerToPatchRequest(_ context.Context, triggerObj types.Object) (*landscape.ScriptProfileTriggerPatchRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	var t landscape.ScriptProfileTriggerPatchRequest

	attrs := triggerObj.Attributes()
	triggerType := attrs["type"].(types.String).ValueString() //nolint:forcetypeassert

	switch triggerType {
	case "event":
		eventType := attrs["event_type"].(types.String).ValueString() //nolint:forcetypeassert
		err := t.FromScriptProfileEventTrigger(landscape.ScriptProfileEventTrigger{
			TriggerType: landscape.Event,
			EventType:   landscape.ScriptProfileEventType(eventType),
		})
		if err != nil {
			diags.AddError("Failed to build event trigger", err.Error())
		}
	case "recurring":
		interval := attrs["interval"].(types.String).ValueString()         //nolint:forcetypeassert
		startAfterStr := attrs["start_after"].(types.String).ValueString() //nolint:forcetypeassert
		var startAfterPtr *time.Time
		if startAfterStr != "" {
			sa, err := time.Parse(time.RFC3339, startAfterStr)
			if err != nil {
				diags.AddError("Invalid start_after", fmt.Sprintf("Must be RFC3339: %s", err))
				return nil, diags
			}
			startAfterPtr = &sa
		}
		err := t.FromScriptProfileScheduleDraftEditTrigger(landscape.ScriptProfileScheduleDraftEditTrigger{
			TriggerType: landscape.ScriptProfileScheduleDraftEditTriggerTriggerTypeRecurring,
			Interval:    &interval,
			StartAfter:  startAfterPtr,
		})
		if err != nil {
			diags.AddError("Failed to build recurring trigger", err.Error())
		}
	case "one_time":
		tsStr := attrs["timestamp"].(types.String).ValueString() //nolint:forcetypeassert
		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			diags.AddError("Invalid timestamp", fmt.Sprintf("Must be RFC3339: %s", err))
			return nil, diags
		}
		err = t.FromScriptProfileOneTimeDraftTrigger(landscape.ScriptProfileOneTimeDraftTrigger{
			TriggerType: landscape.ScriptProfileOneTimeDraftTriggerTriggerTypeOneTime,
			Timestamp:   ts,
		})
		if err != nil {
			diags.AddError("Failed to build one_time trigger", err.Error())
		}
	default:
		diags.AddError("Invalid trigger type", fmt.Sprintf("Must be one of: event, recurring, one_time. Got: %s", triggerType))
	}
	return &t, diags
}

func scriptProfileDetailToState(ctx context.Context, detail *landscape.ScriptProfileDetail) (ScriptProfileResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	sortedTags := make([]string, len(detail.Tags))
	copy(sortedTags, detail.Tags)
	sort.Strings(sortedTags)
	tags, d := types.SetValueFrom(ctx, types.StringType, sortedTags)
	diags.Append(d...)

	triggerObj, d := triggerResponseToObject(detail.Trigger)
	diags.Append(d...)

	return ScriptProfileResourceModel{
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
	}, diags
}

func triggerResponseToObject(trigger landscape.ScriptProfileTriggerResponse) (types.Object, diag.Diagnostics) {
	null := func() types.String { return types.StringNull() }

	// Unmarshal to detect the discriminator field first
	if ev, err := trigger.AsScriptProfileEventTrigger(); err == nil && string(ev.TriggerType) == "event" {
		return types.ObjectValue(triggerAttrTypes, map[string]attr.Value{
			"type":        types.StringValue(string(ev.TriggerType)),
			"event_type":  types.StringValue(string(ev.EventType)),
			"interval":    null(),
			"start_after": null(),
			"timestamp":   null(),
		})
	}

	if sched, err := trigger.AsScriptProfileScheduleTrigger(); err == nil && string(sched.TriggerType) == "recurring" {
		startAfter := null()
		if !sched.StartAfter.IsZero() {
			startAfter = types.StringValue(sched.StartAfter.Format(time.RFC3339))
		}
		return types.ObjectValue(triggerAttrTypes, map[string]attr.Value{
			"type":        types.StringValue(string(sched.TriggerType)),
			"event_type":  null(),
			"interval":    types.StringValue(sched.Interval),
			"start_after": startAfter,
			"timestamp":   null(),
		})
	}

	if ot, err := trigger.AsScriptProfileOneTimeTrigger(); err == nil && string(ot.TriggerType) == "one_time" {
		return types.ObjectValue(triggerAttrTypes, map[string]attr.Value{
			"type":        types.StringValue(string(ot.TriggerType)),
			"event_type":  null(),
			"interval":    null(),
			"start_after": null(),
			"timestamp":   types.StringValue(ot.Timestamp.Format(time.RFC3339)),
		})
	}

	var diags diag.Diagnostics
	diags.AddError("Unknown trigger type", "Could not deserialise trigger from API response")
	return types.ObjectNull(triggerAttrTypes), diags
}

func errorFromCreateResp(res *landscape.CreateScriptProfileResponse) string {
	if res.JSON400 != nil && res.JSON400.Message != nil {
		return *res.JSON400.Message
	}
	if res.JSON401 != nil && res.JSON401.Message != nil {
		return *res.JSON401.Message
	}
	if res.JSON403 != nil && res.JSON403.Message != nil {
		return *res.JSON403.Message
	}
	if res.JSON404 != nil && res.JSON404.Message != nil {
		return *res.JSON404.Message
	}
	if res.JSON409 != nil && res.JSON409.Message != nil {
		return *res.JSON409.Message
	}
	return res.Status()
}
