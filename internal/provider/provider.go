// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &landscapeProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &landscapeProvider{
			version: version,
		}
	}
}

// landscapeProvider is the provider implementation.
type landscapeProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Metadata returns the provider type name.
func (p *landscapeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "landscape"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *landscapeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "Landscape base URL. Can also be set with the LANDSCAPE_BASE_URL environment variable.",
			},
			"account": schema.StringAttribute{
				Optional:    true,
				Description: "Landscape account name (optional when using email/password authentication). Can also be set with the LANDSCAPE_ACCOUNT environment variable.",
			},
			"access_key": schema.StringAttribute{
				Optional:    true,
				Description: "Landscape API access key (required with secret_key for access key authentication). Can also be set with the LANDSCAPE_ACCESS_KEY environment variable.",
			},
			"email": schema.StringAttribute{
				Optional:    true,
				Description: "Landscape account email (required with password for email authentication). Can also be set with the LANDSCAPE_EMAIL environment variable.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Landscape account password (required with email for email authentication). Can also be set with the LANDSCAPE_PASSWORD environment variable.",
			},
			"secret_key": schema.StringAttribute{
				Sensitive:   true,
				Optional:    true,
				Description: "Landscape API secret key (required with access_key for access key authentication). Can also be set with the LANDSCAPE_SECRET_KEY environment variable.",
			},
		},
	}
}

// landscapeProviderModel maps provider schema data to a Go type.
type landscapeProviderModel struct {
	BaseURL   types.String `tfsdk:"base_url"`
	Account   types.String `tfsdk:"account"`
	AccessKey types.String `tfsdk:"access_key"`
	Email     types.String `tfsdk:"email"`
	Password  types.String `tfsdk:"password"`
	SecretKey types.String `tfsdk:"secret_key"`
}

// Configure prepares shared API clients for data sources and resources.
func (p *landscapeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config landscapeProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.BaseURL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Unknown Base URL",
			"The provider cannot create the Landscape API client as there is an unknown configuration value for the base URL. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the LANDSCAPE_BASE_URL environment variable.",
		)
	}

	if config.AccessKey.IsUnknown() && !config.SecretKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("access_key"),
			"Unknown Access Key",
			"The provider cannot create the Landscape API client as there is an unknown configuration value for the access key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the LANDSCAPE_ACCESS_KEY environment variable.",
		)
	}

	if config.SecretKey.IsUnknown() && !config.AccessKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("secret_key"),
			"Unknown Secret Key",
			"The provider cannot create the Landscape API client as there is an unknown configuration value for the secret key, but an access key has been provided. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the LANDSCAPE_SECRET_KEY environment variable.",
		)
	}

	if config.Email.IsUnknown() && !config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("email"),
			"Unknown Email",
			"The provider cannot create the Landscape API client as there is an unknown configuration value for the email, but a password has been provided. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the LANDSCAPE_EMAIL environment variable.",
		)
	}

	if config.Password.IsUnknown() && !config.Email.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown Password",
			"The provider cannot create the Landscape API client as there is an unknown configuration value for the password, but an email has been provided. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the LANDSCAPE_PASSWORD environment variable.",
		)
	}

	if !config.Account.IsUnknown() && (config.Email.IsUnknown() || config.Password.IsUnknown()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("account"),
			"Account Name Requires Email/Password",
			"The provider cannot create the Landscape API client as there is an unknown configuration value for the email and password, but an account has been provided. "+
				"Either target apply the sources of the values first, set the values statically in the configuration, or use the LANDSCAPE_EMAIL and LANDSCAPE_PASSWORD environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := os.Getenv("LANDSCAPE_BASE_URL")
	accessKey := os.Getenv("LANDSCAPE_ACCESS_KEY")
	secretKey := os.Getenv("LANDSCAPE_SECRET_KEY")
	email := os.Getenv("LANDSCAPE_EMAIL")
	password := os.Getenv("LANDSCAPE_PASSWORD")
	account := os.Getenv("LANDSCAPE_ACCOUNT")

	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	if !config.Email.IsNull() {
		email = config.Email.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	if !config.AccessKey.IsNull() {
		accessKey = config.AccessKey.ValueString()
	}

	if !config.SecretKey.IsNull() {
		secretKey = config.SecretKey.ValueString()
	}

	if !config.Account.IsNull() {
		account = config.Account.ValueString()
	}

	if baseURL == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Missing Landscape Base URL",
			"The provider cannot create the Landscape API client as there is a missing or empty value for the base URL. "+
				"Set the base URL value in the configuration or use the LANDSCAPE_BASE_URL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	hasEmailAuth := email != "" && password != ""
	hasKeyAuth := accessKey != "" && secretKey != ""

	if !hasEmailAuth && !hasKeyAuth {
		resp.Diagnostics.AddError(
			"Missing Authentication Credentials",
			"The provider requires either email/password authentication or access_key/secret_key authentication. "+
				"Provide either an email and password (and, optionally, an account) or an access_key and secret_key."+
				"Set the values in the configuration or use the corresponding environment variables: "+
				"LANDSCAPE_EMAIL, LANDSCAPE_PASSWORD, LANDSCAPE_ACCOUNT or LANDSCAPE_ACCESS_KEY, LANDSCAPE_SECRET_KEY.",
		)
	}

	if (email != "" && password == "") || (email == "" && password != "") {
		resp.Diagnostics.AddError(
			"Incomplete Email Authentication",
			"Both email and password are required for email authentication. "+
				"Set both email and password values in the configuration or use the LANDSCAPE_EMAIL and LANDSCAPE_PASSWORD environment variables.",
		)
	}

	if (accessKey != "" && secretKey == "") || (accessKey == "" && secretKey != "") {
		resp.Diagnostics.AddError(
			"Incomplete Access Key Authentication",
			"Both access_key and secret_key are required for access key authentication. "+
				"Set both access_key and secret_key values in the configuration or use the LANDSCAPE_ACCESS_KEY and LANDSCAPE_SECRET_KEY environment variables.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var client *landscape.ClientWithResponses
	var err error

	if email != "" && password != "" {
		client, err = landscape.NewLandscapeAPIClient(baseURL, landscape.NewEmailPasswordProvider(email, password, &account))
	} else {
		client, err = landscape.NewLandscapeAPIClient(baseURL, landscape.NewAccessKeyProvider(accessKey, secretKey))
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Landscape API Client",
			"An unexpected error occurred when creating the Landscape API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Landscape API Client Error: "+err.Error(),
		)
		return
	}

	// Make the Landscape API client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

}

// DataSources defines the data sources implemented in the provider.
func (p *landscapeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewScriptDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *landscapeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewScriptResource,
	}
}
