// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ScriptAttachmentResource{}
var _ resource.ResourceWithImportState = &ScriptAttachmentResource{}

func NewScriptAttachmentResource() resource.Resource {
	return &ScriptAttachmentResource{}
}

// ScriptAttachmentResource manages a single attachment on a script.
type ScriptAttachmentResource struct {
	client *landscape.ClientWithResponses
}

type ScriptAttachmentResourceModel struct {
	Id       types.Int64  `tfsdk:"id"`        // Attachment ID (V2 only)
	ScriptId types.Int64  `tfsdk:"script_id"` // Owning script ID
	Filename types.String `tfsdk:"filename"`  // Attachment filename
	Content  types.String `tfsdk:"content"`   // Raw content for upload only
}

var scriptAttachmentResourceSchema = resourceschema.Schema{
	MarkdownDescription: "Script attachment resource (V2 scripts only).",
	Attributes: map[string]resourceschema.Attribute{
		"id": resourceschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Attachment identifier (V2 only).",
		},
		"script_id": resourceschema.Int64Attribute{
			Required:            true,
			MarkdownDescription: "ID of the script this attachment belongs to.",
		},
		"filename": resourceschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Filename for the attachment.",
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"content": resourceschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Attachment content (base64 encoded during upload). Not returned by the API; kept in state.",
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
	},
}

func (r *ScriptAttachmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script_attachment"
}

func (r *ScriptAttachmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = scriptAttachmentResourceSchema
}

func (r *ScriptAttachmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScriptAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ScriptAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ScriptId.IsNull() || plan.ScriptId.IsUnknown() {
		resp.Diagnostics.AddError("Missing script_id", "`script_id` must be set.")
		return
	}
	if plan.Filename.IsNull() || plan.Filename.IsUnknown() {
		resp.Diagnostics.AddError("Missing filename", "`filename` must be set.")
		return
	}
	if plan.Content.IsNull() || plan.Content.IsUnknown() {
		resp.Diagnostics.AddError("Missing content", "`content` must be set.")
		return
	}

	scriptIDStr := fmt.Sprint(plan.ScriptId.ValueInt64())
	fileParam := fmt.Sprintf("%s$$%s", plan.Filename.ValueString(), base64.StdEncoding.EncodeToString([]byte(plan.Content.ValueString())))

	vals := url.Values{
		"script_id": []string{scriptIDStr},
		"file":      []string{fileParam},
	}

	createRes, err := r.client.InvokeLegacyActionWithResponse(ctx, landscape.LegacyActionParams("CreateScriptAttachment"), landscape.EncodeQueryRequestEditor(vals))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create attachment", err.Error())
		return
	}

	if createRes.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to create attachment", fmt.Sprintf("Unexpected response (%s)\n%s", createRes.Status(), createRes.Body))
		return
	}

	state, diags := r.readAttachment(ctx, plan.ScriptId.ValueInt64(), plan.Filename.ValueString(), plan.Content)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ScriptAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ScriptAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Filename.IsNull() || state.Filename.IsUnknown() {
		resp.Diagnostics.AddError("Missing filename", "`filename` must be known to read an attachment.")
		return
	}

	newState, diags := r.readAttachment(ctx, state.ScriptId.ValueInt64(), state.Filename.ValueString(), state.Content)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *ScriptAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All fields are force-new; Update should never be called.
	resp.Diagnostics.AddError("Attachment Update not supported", "Attachments must be replaced.")
}

func (r *ScriptAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ScriptAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vals := url.Values{
		"script_id": []string{fmt.Sprint(state.ScriptId.ValueInt64())},
	}
	if !state.Id.IsNull() && !state.Id.IsUnknown() {
		vals.Set("attachment_id", fmt.Sprint(state.Id.ValueInt64()))
	}
	if !state.Filename.IsNull() && !state.Filename.IsUnknown() {
		vals.Set("filename", state.Filename.ValueString())
	}

	if _, err := r.client.InvokeLegacyAction(ctx, landscape.LegacyActionParams("RemoveScriptAttachment"), landscape.EncodeQueryRequestEditor(vals)); err != nil {
		resp.Diagnostics.AddError("Failed to remove attachment", err.Error())
	}
}

func (r *ScriptAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expect import ID in the form "<script_id>/<filename>"
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected import identifier in the form <script_id>/<filename>.")
		return
	}

	scriptID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid script_id", fmt.Sprintf("Could not parse script_id from import ID: %v", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("script_id"), scriptID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("filename"), parts[1])...)
}

func (r *ScriptAttachmentResource) readAttachment(ctx context.Context, scriptID int64, filename string, content types.String) (*ScriptAttachmentResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	scriptRes, err := r.client.GetScriptWithResponse(ctx, landscape.ScriptIdPathParam(scriptID))
	if err != nil {
		diags.AddError("Failed to read script", err.Error())
		return nil, diags
	}

	if scriptRes.JSON200 == nil {
		diags.AddError("Failed to read script", fmt.Sprintf("Error getting script: %s", scriptRes.Status()))
		return nil, diags
	}

	v2, err := scriptRes.JSON200.AsV2Script()
	if err != nil {
		diags.AddError("Unsupported script type", "Script attachments are only available for V2 scripts.")
		return nil, diags
	}

	var attachment *landscape.ScriptAttachment
	if v2.Attachments != nil {
		for _, a := range *v2.Attachments {
			if a.Filename == filename {
				aCopy := a
				attachment = &aCopy
				break
			}
		}
	}

	if attachment == nil {
		diags.AddError("Attachment not found", fmt.Sprintf("No attachment named %q exists on script %d", filename, scriptID))
		return nil, diags
	}

	state := ScriptAttachmentResourceModel{
		Id:       types.Int64Value(int64(attachment.Id)),
		ScriptId: types.Int64Value(scriptID),
		Filename: types.StringValue(filename),
		Content:  content, // not returned by API; preserve prior value
	}

	return &state, diags
}
