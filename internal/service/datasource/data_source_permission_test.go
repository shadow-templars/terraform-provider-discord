package datasource

import (
	"context"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// discordPermissionBits lists every permission attribute and its bit, taken
// from https://discord.com/developers/docs/topics/permissions#permissions-bitwise-permission-flags
// rather than from the constants under test.
var discordPermissionBits = map[string]uint{
	"create_instant_invite":               0,
	"kick_members":                        1,
	"ban_members":                         2,
	"administrator":                       3,
	"manage_channels":                     4,
	"manage_guild":                        5,
	"add_reactions":                       6,
	"view_audit_log":                      7,
	"priority_speaker":                    8,
	"stream":                              9,
	"view_channel":                        10,
	"send_messages":                       11,
	"send_tts_messages":                   12,
	"manage_messages":                     13,
	"embed_links":                         14,
	"attach_files":                        15,
	"read_message_history":                16,
	"mention_everyone":                    17,
	"use_external_emojis":                 18,
	"view_guild_insights":                 19,
	"connect":                             20,
	"speak":                               21,
	"mute_members":                        22,
	"deafen_members":                      23,
	"move_members":                        24,
	"use_vad":                             25,
	"change_nickname":                     26,
	"manage_nicknames":                    27,
	"manage_roles":                        28,
	"manage_webhooks":                     29,
	"manage_expressions":                  30,
	"use_app_commands":                    31,
	"request_to_speak":                    32,
	"manage_events":                       33,
	"manage_threads":                      34,
	"create_public_threads":               35,
	"create_private_threads":              36,
	"use_external_stickers":               37,
	"send_messages_in_threads":            38,
	"use_embedded_activities":             39,
	"moderate_members":                    40,
	"view_creator_monetization_analytics": 41,
	"use_soundboard":                      42,
	"create_expressions":                  43,
	"create_events":                       44,
	"use_external_sounds":                 45,
	"send_voice_messages":                 46,
	"set_voice_channel_status":            48,
	"send_polls":                          49,
	"use_external_apps":                   50,
	"pin_messages":                        51,
	"bypass_slowmode":                     52,

	// Aliases for the names above.
	"use_application_commands":  31,
	"send_thread_messages":      38,
	"start_embedded_activities": 39,
}

// readPermissionBits runs the data source with the given attributes set and
// returns its allow_bits and deny_bits.
func readPermissionBits(t *testing.T, settings map[string]string) (int64, int64) {
	t.Helper()
	ctx := context.Background()
	ds := NewPermissionDataSource()

	var schemaResp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	attrTypes := schemaResp.Schema.Type().(types.ObjectType).AttrTypes

	values := make(map[string]attr.Value, len(attrTypes))
	for name, attrType := range attrTypes {
		if attrType.Equal(types.Int64Type) {
			values[name] = types.Int64Null()
			continue
		}
		if setting, ok := settings[name]; ok {
			values[name] = types.StringValue(setting)
		} else {
			values[name] = types.StringNull()
		}
	}
	for name := range settings {
		if _, ok := attrTypes[name]; !ok {
			t.Fatalf("attribute %q is not in the schema", name)
		}
	}

	config, diags := types.ObjectValue(attrTypes, values)
	if diags.HasError() {
		t.Fatalf("building config: %v", diags)
	}
	rawConfig, err := config.ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("converting config: %v", err)
	}
	rawState, err := types.ObjectNull(attrTypes).ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("converting state: %v", err)
	}

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: rawConfig}}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
	ds.Read(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var allow, deny types.Int64
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("allow_bits"), &allow)...)
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("deny_bits"), &deny)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("reading state: %v", resp.Diagnostics)
	}
	return allow.ValueInt64(), deny.ValueInt64()
}

func TestPermissionDataSourceFlagBits(t *testing.T) {
	for name, bit := range discordPermissionBits {
		want := int64(1) << bit

		allow, deny := readPermissionBits(t, map[string]string{name: "allow"})
		if allow != want || deny != 0 {
			t.Errorf("%s = allow: got allow_bits %d, deny_bits %d; want %d, 0", name, allow, deny, want)
		}

		allow, deny = readPermissionBits(t, map[string]string{name: "deny"})
		if allow != 0 || deny != want {
			t.Errorf("%s = deny: got allow_bits %d, deny_bits %d; want 0, %d", name, allow, deny, want)
		}
	}
}

// Every permission attribute in the schema must be a known Discord flag, and
// every known flag must be exposed, so a constant can't be defined without an
// attribute (or the reverse).
func TestPermissionDataSourceSchemaMatchesFlags(t *testing.T) {
	var schemaResp datasource.SchemaResponse
	NewPermissionDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	var unknown []string
	for name := range schemaResp.Schema.Attributes {
		if name == "allow_bits" || name == "deny_bits" {
			continue
		}
		if _, ok := discordPermissionBits[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	var missing []string
	for name := range discordPermissionBits {
		if _, ok := schemaResp.Schema.Attributes[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(unknown)
	sort.Strings(missing)
	if len(unknown) > 0 {
		t.Errorf("schema attributes without a Discord flag in the test table: %v", unknown)
	}
	if len(missing) > 0 {
		t.Errorf("Discord flags missing from the schema: %v", missing)
	}
}

func TestPermissionDataSourceCombinesFlags(t *testing.T) {
	allow, deny := readPermissionBits(t, map[string]string{
		"view_channel":         "allow",
		"read_message_history": "allow",
		"send_messages":        "deny",
		"pin_messages":         "deny",
	})
	wantAllow := int64(1<<10 | 1<<16)
	wantDeny := int64(1<<11 | 1<<51)
	if allow != wantAllow || deny != wantDeny {
		t.Errorf("got allow_bits %d, deny_bits %d; want %d, %d", allow, deny, wantAllow, wantDeny)
	}

	allow, deny = readPermissionBits(t, nil)
	if allow != 0 || deny != 0 {
		t.Errorf("no settings: got allow_bits %d, deny_bits %d; want 0, 0", allow, deny)
	}
}
