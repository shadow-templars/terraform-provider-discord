package channel

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type textChannelOverwrite struct {
	ID    types.String `tfsdk:"id"`
	Type  types.String `tfsdk:"type"`
	Allow types.Int64  `tfsdk:"allow"`
	Deny  types.Int64  `tfsdk:"deny"`
}

var textChannelOverwriteType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id": types.StringType, "type": types.StringType, "allow": types.Int64Type, "deny": types.Int64Type,
}}

func textChannelOverwritesSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		Optional:    true,
		Description: "The complete set of channel permission overwrites. When set, at least one overwrite is required, sync_perms_with_category must be false, and separate discord_channel_permission resources must not manage this channel. Discord overwrites omitted from the set are removed. Omit this attribute to leave overwrites unmanaged by the text channel.",
		NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"id":    schema.StringAttribute{Required: true, Description: "Discord role or member ID. Use the guild ID for @everyone."},
			"type":  schema.StringAttribute{Required: true, Description: "Overwrite target type: role or member."},
			"allow": schema.Int64Attribute{Required: true, Description: "Allowed permission bits, or zero."},
			"deny":  schema.Int64Attribute{Required: true, Description: "Denied permission bits, or zero."},
		}},
	}
}

func (plan TextChannelResourceModel) channelOverwrites(ctx context.Context, planning bool) ([]*discordgo.PermissionOverwrite, diag.Diagnostics) {
	var diags diag.Diagnostics
	set := plan.PermissionOverwrites
	if set.IsNull() {
		return nil, diags
	}
	if plan.SyncPermsWithCategory.ValueBool() {
		diags.AddError("Conflicting Channel Permission Settings", "Set sync_perms_with_category to false when using permission_overwrites.")
		return nil, diags
	}
	if set.IsUnknown() {
		if !planning {
			diags.AddError("Unknown Channel Permission Overwrites", "The complete overwrite set must be known before writing the channel.")
		}
		return nil, diags
	}
	var entries []textChannelOverwrite
	diags.Append(set.ElementsAs(ctx, &entries, false)...)
	if diags.HasError() {
		return nil, diags
	}
	if len(entries) == 0 {
		diags.AddError("Empty Channel Permission Overwrites", "Provide at least one overwrite or omit permission_overwrites to leave the list unmanaged.")
		return nil, diags
	}
	ids := make(map[string]bool, len(entries))
	overwrites := make([]*discordgo.PermissionOverwrite, 0, len(entries))
	for _, entry := range entries {
		if entry.ID.IsUnknown() || entry.Type.IsUnknown() || entry.Allow.IsUnknown() || entry.Deny.IsUnknown() {
			if !planning {
				diags.AddError("Unknown Channel Permission Overwrite", "Each overwrite must be fully known before writing the channel.")
			}
			continue
		}
		id := entry.ID.ValueString()
		value, err := strconv.ParseUint(id, 10, 64)
		if err != nil || value == 0 || strconv.FormatUint(value, 10) != id || ids[id] {
			diags.AddError("Invalid Channel Permission Target", fmt.Sprintf("Overwrite ID %q must be a unique Discord ID.", id))
			continue
		}
		ids[id] = true
		targetType, typeDiags := parseOverwriteType(entry.Type.ValueString())
		diags.Append(typeDiags...)
		allow, deny := entry.Allow.ValueInt64(), entry.Deny.ValueInt64()
		if entry.Allow.IsNull() || entry.Deny.IsNull() || allow < 0 || deny < 0 || allow&deny != 0 {
			diags.AddError("Invalid Channel Permission Bits", fmt.Sprintf("Overwrite %q needs nonnegative allow and deny masks with no overlapping bits.", id))
		}
		overwrites = append(overwrites, &discordgo.PermissionOverwrite{ID: id, Type: targetType, Allow: allow, Deny: deny})
	}
	return overwrites, diags
}

func textChannelOverwriteState(ctx context.Context, overwrites []*discordgo.PermissionOverwrite) (types.Set, diag.Diagnostics) {
	entries := make([]textChannelOverwrite, 0, len(overwrites))
	for _, overwrite := range overwrites {
		kind := "role"
		if overwrite.Type == discordgo.PermissionOverwriteTypeMember {
			kind = "member"
		}
		entries = append(entries, textChannelOverwrite{
			ID: types.StringValue(overwrite.ID), Type: types.StringValue(kind),
			Allow: types.Int64Value(overwrite.Allow), Deny: types.Int64Value(overwrite.Deny),
		})
	}
	return types.SetValueFrom(ctx, textChannelOverwriteType, entries)
}
