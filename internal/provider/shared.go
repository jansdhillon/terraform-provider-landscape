package provider

import (
	"context"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	landscape "github.com/jansdhillon/landscape-go-api-client/client"
)

// ScriptAttachmentItemsAsListOfStringValues takes an array of script attachment objects (JSON) and attempts
// to convert them into a list of string values to conform with the legacy script attachment schema.
// Returns the resulting list and ok when conversion succeeds; otherwise returns a zero value and false.
func ScriptAttachmentItemsAsListOfStringValues(ctx context.Context, sai []landscape.Script_Attachments_Item) (basetypes.ListValue, bool) {
	if len(sai) == 0 {
		return basetypes.ListValue{}, false
	}

	legacyAttachmentsStrings := make([]attr.Value, 0, len(sai))
	for _, a := range sai {
		legacyAttachment, err := a.AsLegacyScriptAttachment()
		if err != nil {
			// not all attachments are legacy-compatible; signal caller to try the modern shape
			tflog.Debug(ctx, "attachment not convertible to legacy string representation")
			return basetypes.ListValue{}, false
		}

		legacyAttachmentsStrings = append(legacyAttachmentsStrings, types.StringValue(string(legacyAttachment)))
	}

	listValue, diags := types.ListValue(types.StringType, legacyAttachmentsStrings)
	if diags.HasError() {
		tflog.Error(ctx, "failed to build legacy attachments list", map[string]any{"diagnostics": diags})
		return basetypes.ListValue{}, false
	}

	return listValue, true
}

// ScriptAttachmentItemsAsListOfObjectValues takes an array of script attachment objects (JSON) and attempts
// to convert them into a list of objects to conform with the modern script attachment schema.
func ScriptAttachmentItemsAsListOfObjectValues(ctx context.Context, sai []landscape.Script_Attachments_Item) (basetypes.ListValue, bool) {
	if len(sai) == 0 {
		return basetypes.ListValue{}, false
	}

	objType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":       types.NumberType,
		"filename": types.StringType,
	}}

	attachmentObjects := make([]attr.Value, 0, len(sai))
	for _, a := range sai {
		attachment, err := a.AsScriptAttachment()
		if err != nil {
			tflog.Debug(ctx, "attachment not convertible to object representation")
			return basetypes.ListValue{}, false
		}

		id := types.NumberValue(big.NewFloat(float64(attachment.Id)))
		filename := types.StringValue(attachment.Filename)
		attachmentObject, diags := types.ObjectValue(objType.AttrTypes, map[string]attr.Value{"id": id, "filename": filename})
		if diags.HasError() {
			tflog.Error(ctx, "couldn't convert script attachment into object", map[string]any{"diagnostics": diags})
			return basetypes.ListValue{}, false
		}

		attachmentObjects = append(attachmentObjects, attachmentObject)
	}

	listValue, diags := types.ListValue(objType, attachmentObjects)
	if diags.HasError() {
		tflog.Error(ctx, "failed to build attachments object list", map[string]any{"diagnostics": diags})
		return basetypes.ListValue{}, false
	}

	return listValue, true
}
