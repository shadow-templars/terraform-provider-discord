package discordgox

import (
	"encoding/json"

	"github.com/bwmarrin/discordgo"
)

// GuildCommunityParams holds the Community-related guild settings applied in a
// single Modify Guild (PATCH /guilds/{id}) call.
//
// It exists because discordgo's GuildParams (v0.29.0) omits
// safety_alerts_channel_id, which Discord documents as an editable guild field.
// omitempty leaves any unset field untouched by Discord.
type GuildCommunityParams struct {
	Features               []discordgo.GuildFeature `json:"features,omitempty"`
	RulesChannelID         string                   `json:"rules_channel_id,omitempty"`
	PublicUpdatesChannelID string                   `json:"public_updates_channel_id,omitempty"`
	SafetyAlertsChannelID  string                   `json:"safety_alerts_channel_id,omitempty"`
	PreferredLocale        string                   `json:"preferred_locale,omitempty"`
}

// GuildCommunityEdit applies Community guild settings, including
// safety_alerts_channel_id, via Modify Guild. Returns the updated guild.
func (c *Client) GuildCommunityEdit(guildID string, params GuildCommunityParams) (*discordgo.Guild, error) {
	endpoint := discordgo.EndpointGuild(guildID)
	body, err := c.RequestWithBucketID("PATCH", endpoint, params, endpoint)
	if err != nil {
		return nil, err
	}

	var guild discordgo.Guild
	if err := json.Unmarshal(body, &guild); err != nil {
		return nil, err
	}
	return &guild, nil
}

// GuildExtended embeds discordgo's Guild and adds fields discordgo does not
// model on read, currently safety_alerts_channel_id.
type GuildExtended struct {
	*discordgo.Guild
	SafetyAlertsChannelID string `json:"safety_alerts_channel_id"`
}

// GuildWithExtras fetches a guild and unmarshals the extra fields discordgo
// omits. Use it when a resource needs safety_alerts_channel_id; otherwise the
// embedded session's Guild() is sufficient.
func (c *Client) GuildWithExtras(guildID string) (*GuildExtended, error) {
	endpoint := discordgo.EndpointGuild(guildID)
	body, err := c.RequestWithBucketID("GET", endpoint, nil, endpoint)
	if err != nil {
		return nil, err
	}

	var guild GuildExtended
	if err := json.Unmarshal(body, &guild); err != nil {
		return nil, err
	}
	return &guild, nil
}
