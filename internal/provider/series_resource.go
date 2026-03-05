// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ resource.Resource = &SeriesResource{}
var _ resource.ResourceWithImportState = &SeriesResource{}

func NewSeriesResource() resource.Resource {
	return &SeriesResource{}
}

type SeriesResource struct {
	client *landscape.ClientWithResponses
}

type SeriesResourceModel struct {
	Name          types.String `tfsdk:"name"`
	Distribution  types.String `tfsdk:"distribution"`
	Pockets       types.List   `tfsdk:"pockets"`
	Components    types.List   `tfsdk:"components"`
	Architectures types.List   `tfsdk:"architectures"`
	GpgKey        types.String `tfsdk:"gpg_key"`
	MirrorUri     types.String `tfsdk:"mirror_uri"`
	MirrorSeries  types.String `tfsdk:"mirror_series"`
	MirrorGpgKey  types.String `tfsdk:"mirror_gpg_key"`
	IncludeUdeb   types.Bool   `tfsdk:"include_udeb"`
}

func (r *SeriesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_series"
}

func (r *SeriesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "Manages a Landscape series (e.g. `noble`, `jammy`) within a distribution, optionally creating mirror pockets.",
		Attributes: map[string]resourceschema.Attribute{
			"name": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the series (e.g. `noble`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"distribution": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the parent distribution.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"pockets": resourceschema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Pocket names to create (e.g. `[\"release\",\"updates\",\"security\"]`). Created in mirror mode by default.",
			},
			"components": resourceschema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Component names for the created pockets (e.g. `[\"main\",\"universe\"]`).",
			},
			"architectures": resourceschema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Architecture names for the created pockets (e.g. `[\"amd64\"]`).",
			},
			"gpg_key": resourceschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Name of the GPG key to sign pocket package lists.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"mirror_uri": resourceschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "URI to mirror (e.g. `http://archive.ubuntu.com/ubuntu`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"mirror_series": resourceschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Remote series name to mirror. Defaults to the local series name.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"mirror_gpg_key": resourceschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "GPG key to verify the mirrored archive signature.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"include_udeb": resourceschema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether to mirror .udeb packages (debian-installer).",
			},
		},
	}
}

func (r *SeriesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SeriesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SeriesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &landscape.LegacyCreateSeriesParams{
		Name:         plan.Name.ValueString(),
		Distribution: plan.Distribution.ValueString(),
	}

	if !plan.Pockets.IsNull() && !plan.Pockets.IsUnknown() {
		var pockets []string
		resp.Diagnostics.Append(plan.Pockets.ElementsAs(ctx, &pockets, false)...)
		params.Pockets = &pockets
	}
	if !plan.Components.IsNull() && !plan.Components.IsUnknown() {
		var components []string
		resp.Diagnostics.Append(plan.Components.ElementsAs(ctx, &components, false)...)
		params.Components = &components
	}
	if !plan.Architectures.IsNull() && !plan.Architectures.IsUnknown() {
		var architectures []string
		resp.Diagnostics.Append(plan.Architectures.ElementsAs(ctx, &architectures, false)...)
		params.Architectures = &architectures
	}
	if !plan.GpgKey.IsNull() && !plan.GpgKey.IsUnknown() {
		v := plan.GpgKey.ValueString()
		params.GpgKey = &v
	}
	if !plan.MirrorUri.IsNull() && !plan.MirrorUri.IsUnknown() {
		v := plan.MirrorUri.ValueString()
		params.MirrorUri = &v
	}
	if !plan.MirrorSeries.IsNull() && !plan.MirrorSeries.IsUnknown() {
		v := plan.MirrorSeries.ValueString()
		params.MirrorSeries = &v
	}
	if !plan.MirrorGpgKey.IsNull() && !plan.MirrorGpgKey.IsUnknown() {
		v := plan.MirrorGpgKey.ValueString()
		params.MirrorGpgKey = &v
	}
	if !plan.IncludeUdeb.IsNull() && !plan.IncludeUdeb.IsUnknown() {
		v := plan.IncludeUdeb.ValueBool()
		params.IncludeUdeb = &v
	}

	if resp.Diagnostics.HasError() {
		return
	}

	rawResp, err := r.client.LegacyCreateSeries(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create series", err.Error())
		return
	}
	defer rawResp.Body.Close()
	if rawResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rawResp.Body)
		resp.Diagnostics.AddError("Failed to create series", fmt.Sprintf("status %s: %s", rawResp.Status, body))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SeriesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SeriesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	distName := state.Distribution.ValueString()
	names := []string{distName}
	rawResp, err := r.client.LegacyGetDistributions(ctx, &landscape.LegacyGetDistributionsParams{
		Names: &names,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read series", err.Error())
		return
	}
	defer rawResp.Body.Close()
	body, _ := io.ReadAll(rawResp.Body)

	if rawResp.StatusCode != http.StatusOK {
		resp.State.RemoveResource(ctx)
		return
	}

	dists, err := landscape.ParseLegacyResponse[[]map[string]any](body)
	if err != nil || len(dists) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	seriesName := state.Name.ValueString()
	found := false
	for _, dist := range dists {
		if series, ok := dist["series"].([]any); ok {
			for _, s := range series {
				if sm, ok := s.(map[string]any); ok {
					if sm["name"] == seriesName {
						found = true
						break
					}
				}
			}
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SeriesResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All identity fields require replacement; Update is never called.
}

func (r *SeriesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SeriesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawResp, err := r.client.LegacyRemoveSeries(ctx, &landscape.LegacyRemoveSeriesParams{
		Name:         state.Name.ValueString(),
		Distribution: state.Distribution.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to remove series", err.Error())
		return
	}
	defer rawResp.Body.Close()
	if rawResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rawResp.Body)
		resp.Diagnostics.AddError("Failed to remove series", fmt.Sprintf("status %s: %s", rawResp.Status, body))
	}
}

// ImportState accepts "<distribution>/<name>".
func (r *SeriesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: <distribution>/<name>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("distribution"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}
