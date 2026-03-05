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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var _ resource.Resource = &RepositoryProfileResource{}
var _ resource.ResourceWithImportState = &RepositoryProfileResource{}

func NewRepositoryProfileResource() resource.Resource {
	return &RepositoryProfileResource{}
}

type RepositoryProfileResource struct {
	client *landscape.ClientWithResponses
}

type RepositoryProfileResourceModel struct {
	Name         types.String `tfsdk:"name"`
	Title        types.String `tfsdk:"title"`
	Description  types.String `tfsdk:"description"`
	AccessGroup  types.String `tfsdk:"access_group"`
	Pockets      types.List   `tfsdk:"pockets"`
	Series       types.String `tfsdk:"series"`
	Distribution types.String `tfsdk:"distribution"`
	AllComputers types.Bool   `tfsdk:"all_computers"`
	Tags         types.Set    `tfsdk:"tags"`
}

func (r *RepositoryProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_profile"
}

func (r *RepositoryProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "Manages a Landscape repository profile, which groups pockets and associates them with computers.",
		Attributes: map[string]resourceschema.Attribute{
			"name": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The slug name of the profile returned by the API.",
			},
			"title": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Title for the repository profile.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": resourceschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Description of the profile.",
			},
			"access_group": resourceschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Access group to create the profile in.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"pockets": resourceschema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Pocket names to add to this profile (e.g. `[\"release\", \"updates\"]`).",
			},
			"series": resourceschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Name of the series the pockets belong to. Required when `pockets` is set.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"distribution": resourceschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Name of the distribution the series belongs to. Required when `pockets` is set.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"all_computers": resourceschema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether to apply the profile to all computers.",
			},
			"tags": resourceschema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Tags used to target computers.",
			},
		},
	}
}

func (r *RepositoryProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RepositoryProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RepositoryProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createParams := &landscape.LegacyCreateRepositoryProfileParams{
		Title: plan.Title.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		createParams.Description = &v
	}
	if !plan.AccessGroup.IsNull() && !plan.AccessGroup.IsUnknown() {
		v := plan.AccessGroup.ValueString()
		createParams.AccessGroup = &v
	}

	rawResp, err := r.client.LegacyCreateRepositoryProfile(ctx, createParams)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create repository profile", err.Error())
		return
	}
	defer rawResp.Body.Close()
	body, _ := io.ReadAll(rawResp.Body)
	if rawResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("Failed to create repository profile", fmt.Sprintf("status %s: %s", rawResp.Status, body))
		return
	}

	// Extract the slug name from the response.
	profileData, err := landscape.ParseLegacyResponse[map[string]any](body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse repository profile response", err.Error())
		return
	}
	profileName, _ := profileData["name"].(string)
	if profileName == "" {
		resp.Diagnostics.AddError("Failed to parse repository profile response", "response missing 'name' field")
		return
	}
	plan.Name = types.StringValue(profileName)

	// Add pockets if specified.
	if !plan.Pockets.IsNull() && !plan.Pockets.IsUnknown() {
		var pockets []string
		resp.Diagnostics.Append(plan.Pockets.ElementsAs(ctx, &pockets, false)...)
		if !resp.Diagnostics.HasError() && len(pockets) > 0 {
			pRaw, err := r.client.LegacyAddPocketsToRepositoryProfile(ctx, &landscape.LegacyAddPocketsToRepositoryProfileParams{
				Name:         profileName,
				Pockets:      pockets,
				Series:       plan.Series.ValueString(),
				Distribution: plan.Distribution.ValueString(),
			})
			if err != nil {
				resp.Diagnostics.AddError("Failed to add pockets to repository profile", err.Error())
				return
			}
			defer pRaw.Body.Close()
			if pRaw.StatusCode != http.StatusOK {
				pBody, _ := io.ReadAll(pRaw.Body)
				resp.Diagnostics.AddError("Failed to add pockets to repository profile", fmt.Sprintf("status %s: %s", pRaw.Status, pBody))
				return
			}
		}
	}

	// Associate with tags / all_computers if specified.
	hasTags := !plan.Tags.IsNull() && !plan.Tags.IsUnknown()
	allComputersSet := !plan.AllComputers.IsNull() && !plan.AllComputers.IsUnknown() && plan.AllComputers.ValueBool()
	if hasTags || allComputersSet {
		assocParams := &landscape.LegacyAssociateRepositoryProfileParams{
			Name: profileName,
		}
		if hasTags {
			var tags []string
			resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
			assocParams.Tags = &tags
		}
		if allComputersSet {
			v := true
			assocParams.AllComputers = &v
		}
		if !resp.Diagnostics.HasError() {
			aRaw, err := r.client.LegacyAssociateRepositoryProfile(ctx, assocParams)
			if err != nil {
				resp.Diagnostics.AddError("Failed to associate repository profile", err.Error())
				return
			}
			defer aRaw.Body.Close()
			if aRaw.StatusCode != http.StatusOK {
				aBody, _ := io.ReadAll(aRaw.Body)
				resp.Diagnostics.AddError("Failed to associate repository profile", fmt.Sprintf("status %s: %s", aRaw.Status, aBody))
				return
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RepositoryProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RepositoryProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	names := []string{state.Name.ValueString()}
	rawResp, err := r.client.LegacyGetRepositoryProfiles(ctx, &landscape.LegacyGetRepositoryProfilesParams{
		Names: &names,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read repository profile", err.Error())
		return
	}
	defer rawResp.Body.Close()
	body, _ := io.ReadAll(rawResp.Body)

	if rawResp.StatusCode != http.StatusOK {
		resp.State.RemoveResource(ctx)
		return
	}

	profiles, err := landscape.ParseLegacyResponse[[]any](body)
	if err != nil || len(profiles) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RepositoryProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state RepositoryProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileName := state.Name.ValueString()
	plan.Name = state.Name

	// Update description if changed.
	if !plan.Description.Equal(state.Description) {
		desc := plan.Description.ValueString()
		eRaw, err := r.client.LegacyEditRepositoryProfile(ctx, &landscape.LegacyEditRepositoryProfileParams{
			Name:        profileName,
			Description: &desc,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update repository profile", err.Error())
			return
		}
		defer eRaw.Body.Close()
		if eRaw.StatusCode != http.StatusOK {
			eBody, _ := io.ReadAll(eRaw.Body)
			resp.Diagnostics.AddError("Failed to update repository profile", fmt.Sprintf("status %s: %s", eRaw.Status, eBody))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RepositoryProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RepositoryProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawResp, err := r.client.LegacyRemoveRepositoryProfile(ctx, &landscape.LegacyRemoveRepositoryProfileParams{
		Name: state.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to remove repository profile", err.Error())
		return
	}
	defer rawResp.Body.Close()
	if rawResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rawResp.Body)
		resp.Diagnostics.AddError("Failed to remove repository profile", fmt.Sprintf("status %s: %s", rawResp.Status, body))
	}
}

func (r *RepositoryProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
