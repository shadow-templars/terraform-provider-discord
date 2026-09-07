package guild

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"

	"github.com/shadow-templars/terraform-provider-discord/internal/discordgox"
)

// A mention-spam rule with a block action (no metadata) and an alert action
// that sets only channel_id mirrors what Discord returns: empty metadata objects
// and unset sibling fields. refreshState must map those to null, not zero values,
// or Terraform reports "inconsistent result after apply".
func TestRefreshStateActionMetadataNulls(t *testing.T) {
	r := &AutoModerationRuleResource{}
	enabled := true
	raid := true
	rule := &discordgox.AutoModRule{
		ID:          "1",
		Name:        "Block Mention Spam",
		EventType:   1,
		TriggerType: 5,
		Enabled:     &enabled,
		TriggerMetadata: &discordgox.AutoModTriggerMetadata{
			MentionTotalLimit:            20,
			MentionRaidProtectionEnabled: &raid,
		},
		Actions: []discordgo.AutoModerationAction{
			{Type: 1, Metadata: &discordgo.AutoModerationActionMetadata{}},
			{Type: 2, Metadata: &discordgo.AutoModerationActionMetadata{ChannelID: "123"}},
		},
	}

	var state autoModRuleModel
	r.refreshState(context.Background(), rule, &state)

	if len(state.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(state.Actions))
	}

	// Block action: empty metadata object must become no metadata.
	if state.Actions[0].Metadata != nil {
		t.Errorf("block action metadata: want nil, got %+v", state.Actions[0].Metadata)
	}

	// Alert action: channel_id set, siblings must be null (not 0 / "").
	alert := state.Actions[1].Metadata
	if alert == nil {
		t.Fatal("alert action metadata: want set, got nil")
	}
	if alert.ChannelID.ValueString() != "123" {
		t.Errorf("alert channel_id: want 123, got %q", alert.ChannelID.ValueString())
	}
	if !alert.DurationSeconds.IsNull() {
		t.Error("alert duration_seconds: want null, got a value")
	}
	if !alert.CustomMessage.IsNull() {
		t.Error("alert custom_message: want null, got a value")
	}
}

// A SPAM rule has no trigger metadata; Discord still returns an empty object,
// which must collapse to nil to match a config that omits the block.
func TestRefreshStateEmptyTriggerMetadataNil(t *testing.T) {
	r := &AutoModerationRuleResource{}
	enabled := true
	rule := &discordgox.AutoModRule{
		ID:              "2",
		Name:            "Block Spam Content",
		EventType:       1,
		TriggerType:     3,
		Enabled:         &enabled,
		TriggerMetadata: &discordgox.AutoModTriggerMetadata{},
		Actions:         []discordgo.AutoModerationAction{{Type: 1}},
	}

	var state autoModRuleModel
	r.refreshState(context.Background(), rule, &state)

	if state.TriggerMetadata != nil {
		t.Errorf("empty trigger metadata: want nil, got %+v", state.TriggerMetadata)
	}
}
