// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &V2ScriptResource{}
var _ resource.ResourceWithImportState = &V2ScriptResource{}

func NewV2ScriptResource() resource.Resource {
	return &V2ScriptResource{}
}

// V2ScriptResource defines the resource implementation.
type V2ScriptResource struct {
	client *landscape.ClientWithResponses
}

// V2ScriptResourceModel describes the resource data model.
type V2ScriptResourceModel landscape.V2Script

func (r *V2ScriptResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_v2_script"
}

func (r *V2ScriptResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "V2 Script resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Script identifier",
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
				MarkdownDescription: "V2Script attachments (list of strings or objects). Unforuntately, the API returns both 'V1' and 'V2' scripts from the same endpoint, and the return type of `attachments` is different. It is initially a dynamic type that is determined to be legacy (list of strings) or 'modern' (list of objects).",
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

func (r *V2ScriptResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *V2ScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data V2ScriptResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
	//     return
	// }

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	// data.Id = types.StringValue("example-id")

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *V2ScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data V2ScriptResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *V2ScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data V2ScriptResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *V2ScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data V2ScriptResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}

func (r *V2ScriptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
