// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ resource.Resource = &ScriptV2AttachmentResource{}
var _ resource.ResourceWithImportState = &ScriptV2AttachmentResource{}

func NewScriptV2AttachmentResource() resource.Resource {
	return &ScriptV2AttachmentResource{}
}

type ScriptV2AttachmentResource struct {
	client *landscape.ClientWithResponses
}

type scriptV2AttachmentResourceModel struct {
	Id       types.Int64  `tfsdk:"id"`
	ScriptId types.Int64  `tfsdk:"script_id"`
	Filename types.String `tfsdk:"filename"`
	Content  types.String `tfsdk:"content"`
}

func (r *ScriptV2AttachmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script_v2_attachment"
}

func (r *ScriptV2AttachmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "V2 script attachment resource.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Attachment identifier.",
			},
			"script_id": resourceschema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "ID of the V2 script this attachment belongs to.",
			},
			"filename": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Filename for the attachment.",
			},
			"content": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Attachment content.",
			},
		},
	}
}

func (r *ScriptV2AttachmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScriptV2AttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scriptV2AttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
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

	scriptRes, err := r.client.GetScriptWithResponse(ctx, landscape.ScriptIdPathParam(plan.ScriptId.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script after creating attachment", err.Error())
		return
	}

	if scriptRes.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read script", fmt.Sprintf("Error getting script: %s", scriptRes.Status()))
		return
	}

	v2Script, err := scriptRes.JSON200.AsV2Script()
	if err != nil {
		resp.Diagnostics.AddError("Script is not a V2 script", "This attachment resource is for V2 scripts only. Use landscape_script_v1_attachment for V1 scripts.")
		return
	}

	var attachmentID int64
	var found bool

	if v2Script.Attachments != nil {
		for _, att := range *v2Script.Attachments {
			if att.Filename == plan.Filename.ValueString() {
				attachmentID = int64(att.Id)
				found = true
				break
			}
		}
	}

	if !found {
		resp.Diagnostics.AddError("Attachment not found after creation", "Could not find the created attachment in the script")
		return
	}

	state, diags := r.readAttachment(ctx, plan.ScriptId.ValueInt64(), attachmentID, plan.Filename.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ScriptV2AttachmentResource) readAttachment(ctx context.Context, scriptID int64, attachmentID int64, filename string) (*scriptV2AttachmentResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	attachmentContent, err := r.client.GetScriptAttachmentWithResponse(ctx, int(scriptID), int(attachmentID))
	if err != nil {
		diags.AddError("Failed to read script attachment content", err.Error())
		return nil, diags
	}

	if attachmentContent.JSON404 != nil {
		diags.AddError("Attachment not found", *attachmentContent.JSON404.Message)
		return nil, diags
	}

	bodyStr := string(attachmentContent.Body)
	if attachmentContent.StatusCode() == 200 && bodyStr != "" {
		state := scriptV2AttachmentResourceModel{
			Id:       types.Int64Value(attachmentID),
			ScriptId: types.Int64Value(scriptID),
			Filename: types.StringValue(filename),
			Content:  types.StringValue(bodyStr),
		}
		return &state, diags
	}

	diags.AddError("Error reading attachment", fmt.Sprintf("%s\n%s", attachmentContent.Status(), string(attachmentContent.Body)))
	return nil, diags

}

func (r *ScriptV2AttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scriptV2AttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newState, diags := r.readAttachment(ctx, state.ScriptId.ValueInt64(), state.Id.ValueInt64(), state.Filename.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *ScriptV2AttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported for script attachments",
		"Script attachments are immutable. To change the filename or content, delete and recreate the attachment.",
	)
}

func (r *ScriptV2AttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scriptV2AttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vals := url.Values{
		"script_id":     []string{fmt.Sprint(state.ScriptId.ValueInt64())},
		"attachment_id": []string{fmt.Sprint(state.Id.ValueInt64())},
	}

	if _, err := r.client.InvokeLegacyAction(ctx, landscape.LegacyActionParams("RemoveScriptAttachment"), landscape.EncodeQueryRequestEditor(vals)); err != nil {
		resp.Diagnostics.AddError("Failed to remove attachment", err.Error())
	}
}

func (r *ScriptV2AttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
