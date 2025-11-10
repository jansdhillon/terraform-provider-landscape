// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &V1ScriptResource{}
var _ resource.ResourceWithImportState = &V1ScriptResource{}

func NewV1ScriptResource() resource.Resource {
	return &V1ScriptResource{}
}

// V1ScriptResource defines the resource implementation.
type V1ScriptResource struct {
	client *landscape.ClientWithResponses
}

// V1ScriptResourceModel describes the resource data model.
type V1ScriptResourceModel landscape.V1Script

func (r *V1ScriptResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script"
}

func (r *V1ScriptResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "V1 (Legacy) Script resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "V1Script identifier",
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title of the script.",
				Computed:            true,
				Optional:            true,
			},
			"access_group": schema.StringAttribute{
				MarkdownDescription: "The access group the script is in.",
				Computed:            true,
			},
			"code": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The script code content.",
				Optional:            true,
			},
			"interpreter": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The script interpreter.",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "When the script was created.",
			},
			"created_by": schema.SingleNestedAttribute{
				Computed: true,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"id":   schema.NumberAttribute{Computed: true, Optional: true},
					"name": schema.StringAttribute{Computed: true, Optional: true},
				},
				MarkdownDescription: "The creator of the script.",
			},
			"creator": schema.SingleNestedAttribute{
				Computed: true,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"id":    schema.NumberAttribute{Computed: true, Optional: true},
					"name":  schema.StringAttribute{Computed: true, Optional: true},
					"email": schema.StringAttribute{Computed: true, Optional: true},
				},
				MarkdownDescription: "The creator of the (legacy) script.",
			},
			"last_edited_at": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "When the script was last edited.",
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The status of the script.",
			},
			"version_number": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The version number of the script.",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The username associated with the script.",
			},
			"time_limit": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The time limit in milliseconds for a script to complete successfully.",
			},
			"is_editable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script is editable.",
			},
			"is_executable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script is executable.",
			},
			"is_redactable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the script is redactable.",
			},
			"last_edited_by": schema.SingleNestedAttribute{
				Computed: true,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"id":   schema.NumberAttribute{Computed: true, Optional: true},
					"name": schema.StringAttribute{Computed: true, Optional: true},
				},
				MarkdownDescription: "The user who last edited the script.",
			},
			"attachments": schema.DynamicAttribute{
				Computed:            true,
				MarkdownDescription: "V1Script attachments (list of strings or objects). Unforuntately, the API returns both 'V1' and 'V2' scripts from the same endpoint, and the return type of `attachments` is different. It is initially a dynamic type that is determined to be legacy (list of strings) or 'modern' (list of objects).",
			},
			"script_profiles": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":    schema.NumberAttribute{Computed: true},
						"title": schema.StringAttribute{Computed: true},
					},
				},
				MarkdownDescription: "List of script profiles.",
			},
		},
	}
}

func (r *V1ScriptResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*landscape.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected V1 Script Resource Configure Type",
			fmt.Sprintf("Expected *landscape.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *V1ScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data V1ScriptResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createParams := landscape.LegacyActionParams("CreateScript")
	urlVals := url.Values{}

	var username string
	if data.Username != nil {
		username = *data.Username

	}

	if data.TimeLimit != nil {
		urlVals.Add("time_limit", strconv.Itoa(*data.TimeLimit))
	}
	if data.AccessGroup != nil {
		urlVals.Add("access_group", *data.AccessGroup)
	}
	
	if data.

	queryArgsEditorFn := landscape.EncodeQueryRequestEditor(url.Values{
		"code":       []string{},
		"script_id":  []string{strconv.Itoa(data.Id)},
		"title":      []string{data.Title},
		"time_limit": []string{strconv.Itoa(timeLimit)},
		"username":   []string{username},
	})

	editedScriptRes, err := r.client.InvokeLegacyActionWithResponse(ctx, createParams, queryArgsEditorFn)

	if err != nil {
		log.Fatalf("failed to create script: %v", err)
	}

	if editedScriptRes == nil {
		log.Fatalf("edit script response returned nil response")
	}

	if editedScriptRes.StatusCode() != 200 {
		log.Fatalf("editing script failed: status=%d body=%s", editedScriptRes.StatusCode(), string(editedScriptRes.Body))
	}

	if editedScriptRes.JSON200 == nil {
		log.Fatalf("legacy action did not return a script object: %s", string(editedScriptRes.Body))
	}

	editedScript, err := editedScriptRes.JSON200.AsV1Script()
	if err != nil {
		log.Fatalf("failed to convert returned script into a V1Script: %s", err)
	}

	data = V1ScriptResourceModel(editedScript)

	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *V1ScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data V1ScriptResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	scriptRes, err := r.client.GetScriptWithResponse(ctx, data.Id)

	if err != nil {
		resp.Diagnostics.AddError("Failed to read script", err.Error())
		return
	}

	if scriptRes.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to get script", "An error occurred reading the script.")
		return
	}

	script := *scriptRes.JSON200

	v1Script, err := script.AsV1Script()
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert into (legacy) V1 script", "Couldn't convert returned script into a V1 script (is it a modern, V2 script?)")
	}

	data = V1ScriptResourceModel(v1Script)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *V1ScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data V1ScriptResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	editParams := landscape.LegacyActionParams("EditScript")
	var username string
	if data.Username != nil {
		username = *data.Username
	}

	var timeLimit int
	if data.TimeLimit != nil {
		timeLimit = *data.TimeLimit
	}
	queryArgsEditorFn := landscape.EncodeQueryRequestEditor(url.Values{
		"code":       []string{},
		"script_id":  []string{strconv.Itoa(data.Id)},
		"title":      []string{data.Title},
		"time_limit": []string{strconv.Itoa(timeLimit)},
		"username":   []string{username},
	})

	editedScriptRes, err := r.client.InvokeLegacyActionWithResponse(ctx, editParams, queryArgsEditorFn)

	if err != nil {
		log.Fatalf("failed to edit script: %v", err)
	}

	if editedScriptRes == nil {
		log.Fatalf("edit script response returned nil response")
	}

	if editedScriptRes.StatusCode() != 200 {
		log.Fatalf("editing script failed: status=%d body=%s", editedScriptRes.StatusCode(), string(editedScriptRes.Body))
	}

	if editedScriptRes.JSON200 == nil {
		log.Fatalf("editing script did not return a script object: %s", string(editedScriptRes.Body))
	}

	editedScript, err := editedScriptRes.JSON200.AsV1Script()
	if err != nil {
		log.Fatalf("failed to convert returned script into a V1Script: %s", err)
	}

	data = V1ScriptResourceModel(editedScript)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *V1ScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data V1ScriptResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	editParams := landscape.LegacyActionParams("RemoveScript")
	queryArgsEditorFn := landscape.EncodeQueryRequestEditor(url.Values{
		"script_id": []string{strconv.Itoa(data.Id)},
	})

	_, err := r.client.InvokeLegacyActionWithResponse(ctx, editParams, queryArgsEditorFn)

	if err != nil {
		log.Fatalf("failed to remove script: %v", err)
	}

}

func (r *V1ScriptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
