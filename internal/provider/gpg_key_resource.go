// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ resource.Resource = &GPGKeyResource{}
var _ resource.ResourceWithImportState = &GPGKeyResource{}

func NewGPGKeyResource() resource.Resource {
	return &GPGKeyResource{}
}

type GPGKeyResource struct {
	client *landscape.ClientWithResponses
}

type GPGKeyResourceModel struct {
	Name     types.String `tfsdk:"name"`
	Material types.String `tfsdk:"material"`
}

func (r *GPGKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gpg_key"
}

func (r *GPGKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "Imports a GPG key into Landscape for signing repository mirror pocket package lists.",
		Attributes: map[string]resourceschema.Attribute{
			"name": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique name for the GPG key. Must start with an alphanumeric character and contain only lowercase letters, numbers, `-`, or `+`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"material": resourceschema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "ASCII-armored GPG private key material.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *GPGKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*landscape.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *landscape.ClientWithResponses, got: %T.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *GPGKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GPGKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawResp, err := r.client.LegacyImportGPGKey(ctx, &landscape.LegacyImportGPGKeyParams{
		Name:     plan.Name.ValueString(),
		Material: plan.Material.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to import GPG key", err.Error())
		return
	}
	defer rawResp.Body.Close()
	if rawResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rawResp.Body)
		resp.Diagnostics.AddError("Failed to import GPG key", fmt.Sprintf("status %s: %s", rawResp.Status, body))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GPGKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GPGKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	names := []string{state.Name.ValueString()}
	rawResp, err := r.client.LegacyGetGPGKeys(ctx, &landscape.LegacyGetGPGKeysParams{
		Names: &names,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read GPG key", err.Error())
		return
	}
	defer rawResp.Body.Close()
	body, _ := io.ReadAll(rawResp.Body)

	if rawResp.StatusCode != http.StatusOK {
		resp.State.RemoveResource(ctx)
		return
	}

	keys, err := landscape.ParseLegacyResponse[[]any](body)
	if err != nil || len(keys) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GPGKeyResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All fields require replacement; Update is never called.
}

func (r *GPGKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GPGKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawResp, err := r.client.LegacyRemoveGPGKey(ctx, &landscape.LegacyRemoveGPGKeyParams{
		Name: state.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to remove GPG key", err.Error())
		return
	}
	defer rawResp.Body.Close()
	if rawResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rawResp.Body)
		resp.Diagnostics.AddError("Failed to remove GPG key", fmt.Sprintf("status %s: %s", rawResp.Status, body))
	}
}

func (r *GPGKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
