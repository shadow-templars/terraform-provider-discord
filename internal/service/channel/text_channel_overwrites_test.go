package channel

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTextChannelAuthoritativeOverwrites(t *testing.T) {
	ctx := context.Background()
	guild := &discordgo.PermissionOverwrite{ID: "100", Deny: discordgo.PermissionViewChannel}
	officer := &discordgo.PermissionOverwrite{ID: "200", Allow: discordgo.PermissionViewChannel}
	extra := &discordgo.PermissionOverwrite{ID: "300", Allow: discordgo.PermissionViewChannel}
	want, diags := textChannelOverwriteState(ctx, []*discordgo.PermissionOverwrite{guild, officer})
	if diags.HasError() {
		t.Fatal(diags)
	}
	live, diags := textChannelOverwriteState(ctx, []*discordgo.PermissionOverwrite{extra, officer, guild})
	if diags.HasError() || want.Equal(live) {
		t.Fatalf("expected the unexpected live overwrite to produce drift: %v", diags)
	}
	reordered, diags := textChannelOverwriteState(ctx, []*discordgo.PermissionOverwrite{officer, guild})
	if diags.HasError() || !want.Equal(reordered) {
		t.Fatalf("Discord response order should not cause drift: %v", diags)
	}
	plan := TextChannelResourceModel{PermissionOverwrites: want, SyncPermsWithCategory: types.BoolValue(false)}
	sent, diags := plan.channelOverwrites(ctx, false)
	if diags.HasError() || len(sent) != 2 || sent[0].ID == extra.ID || sent[1].ID == extra.ID {
		t.Fatalf("expected only the two configured overwrites: %v, %v", sent, diags)
	}
	plan.SyncPermsWithCategory = types.BoolValue(true)
	if _, diags = plan.channelOverwrites(ctx, true); !diags.HasError() {
		t.Fatal("expected explicit overwrites and category sync to conflict")
	}
	plan.PermissionOverwrites = types.SetNull(textChannelOverwriteType)
	if sent, diags = plan.channelOverwrites(ctx, false); diags.HasError() || sent != nil {
		t.Fatalf("omitted overwrites should leave existing behavior in place: %v, %v", sent, diags)
	}
}
