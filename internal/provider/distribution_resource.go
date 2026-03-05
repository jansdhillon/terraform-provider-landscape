// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
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

var _ resource.Resource = &DistributionResource{}
var _ resource.ResourceWithImportState = &DistributionResource{}

func NewDistributionResource() resource.Resource {
	return &DistributionResource{}
}

type DistributionResource struct {
	client *landscape.ClientWithResponses
}

type DistributionResourceModel struct {
	Name        types.String `tfsdk:"name"`
	AccessGroup types.String `tfsdk:"access_group"`
}

func (r *DistributionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_distribution"
}

func (r *DistributionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "Manages a Landscape repository distribution (the top-level container for series and pockets).",
		Attributes: map[string]resourceschema.Attribute{
			"name": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique name for the distribution. Must start with an alphanumeric character and contain only lowercase letters, numbers, `-`, or `+`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_group": resourceschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Access group to create the distribution in. Defaults to `global`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *DistributionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DistributionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DistributionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &landscape.LegacyCreateDistributionParams{
		Name: plan.Name.ValueString(),
	}
	if !plan.AccessGroup.IsNull() && !plan.AccessGroup.IsUnknown() {
		ag := plan.AccessGroup.ValueString()
		params.AccessGroup = &ag
	}

	rawResp, err := r.client.LegacyCreateDistribution(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create distribution", err.Error())
		return
	}
	defer rawResp.Body.Close()
	if rawResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rawResp.Body)
		// Treat duplicate as already-exists: adopt the existing distribution into state.
		if rawResp.StatusCode == http.StatusBadRequest {
			var apiErr struct {
				Error string `json:"error"`
			}
			if jsonErr := json.Unmarshal(body, &apiErr); jsonErr == nil && apiErr.Error == "DuplicateDistribution" {
				resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
				return
			}
		}
		resp.Diagnostics.AddError("Failed to create distribution", fmt.Sprintf("status %s: %s", rawResp.Status, body))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DistributionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DistributionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	names := []string{state.Name.ValueString()}
	rawResp, err := r.client.LegacyGetDistributions(ctx, &landscape.LegacyGetDistributionsParams{
		Names: &names,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read distribution", err.Error())
		return
	}
	defer rawResp.Body.Close()
	body, _ := io.ReadAll(rawResp.Body)

	if rawResp.StatusCode != http.StatusOK {
		resp.State.RemoveResource(ctx)
		return
	}

	dists, err := landscape.ParseLegacyResponse[[]any](body)
	if err != nil || len(dists) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DistributionResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All fields require replacement; Update is never called.
}

func (r *DistributionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DistributionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawResp, err := r.client.LegacyRemoveDistribution(ctx, &landscape.LegacyRemoveDistributionParams{
		Name: state.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to remove distribution", err.Error())
		return
	}
	defer rawResp.Body.Close()
	if rawResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rawResp.Body)
		resp.Diagnostics.AddError("Failed to remove distribution", fmt.Sprintf("status %s: %s", rawResp.Status, body))
	}
}

func (r *DistributionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
