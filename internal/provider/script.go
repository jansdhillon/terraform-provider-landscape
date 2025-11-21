// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

var v2LastEditedByAttrTypes = map[string]attr.Type{
	"id":   types.Int64Type,
	"name": types.StringType,
}

var createdByAttrTypes = map[string]attr.Type{
	"id":    types.Int64Type,
	"name":  types.StringType,
	"email": types.StringType,
}

var scriptProfileAttrType = map[string]attr.Type{
	"id":    types.Int64Type,
	"title": types.StringType,
}

var scriptAttachmentAttrType = map[string]attr.Type{
	"id":       types.Int64Type,
	"filename": types.StringType,
}

func int64Ptr(i int64) *int64 {
	return &i
}

func fetchV1Code(ctx context.Context, client *landscape.ClientWithResponses, id int) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	codeRes, err := client.InvokeLegacyActionWithResponse(
		ctx,
		landscape.LegacyActionParams("GetScriptCode"),
		landscape.EncodeQueryRequestEditor(url.Values{"script_id": []string{strconv.Itoa(id)}}),
	)
	if err != nil {
		diags.AddError("code fetch failed", err.Error())
		return "", diags
	}

	if codeRes.JSON200 == nil {
		errMsg := "unexpected response fetching V1 script code"
		if codeRes.JSON400 != nil && codeRes.JSON400.Message != nil {
			errMsg = fmt.Sprintf("%s: %s", errMsg, *codeRes.JSON400.Message)
		} else {
			errMsg = fmt.Sprintf("%s: %s", errMsg, codeRes.Status())
		}
		diags.AddError("Getting script code failed", errMsg)
		return "", diags
	}

	raw, err := codeRes.JSON200.AsLegacyScriptCode()
	if err != nil {
		diags.AddError("Decoding legacy script code failed", err.Error())
		return "", diags
	}

	return raw, diags
}
