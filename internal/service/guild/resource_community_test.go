package guild

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestCommunityFeatureHelpers(t *testing.T) {
	base := []discordgo.GuildFeature{discordgo.GuildFeatureNews, discordgo.GuildFeatureInviteSplash}

	if hasCommunityFeature(base) {
		t.Error("hasCommunityFeature: base set should not contain COMMUNITY")
	}

	added := addCommunityFeature(base)
	if !hasCommunityFeature(added) {
		t.Error("addCommunityFeature: result should contain COMMUNITY")
	}
	if len(added) != len(base)+1 {
		t.Errorf("addCommunityFeature: len = %d, want %d", len(added), len(base)+1)
	}

	// Idempotent: adding when already present does not duplicate.
	again := addCommunityFeature(added)
	if len(again) != len(added) {
		t.Errorf("addCommunityFeature (idempotent): len = %d, want %d", len(again), len(added))
	}

	removed := removeCommunityFeature(added)
	if hasCommunityFeature(removed) {
		t.Error("removeCommunityFeature: result should not contain COMMUNITY")
	}
	if len(removed) != len(base) {
		t.Errorf("removeCommunityFeature: len = %d, want %d", len(removed), len(base))
	}
}
