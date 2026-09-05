package discordgox

import (
	"encoding/json"

	"github.com/bwmarrin/discordgo"
)

// GuildWidgetSettings is the editable guild widget configuration. discordgo has
// no typed helper for reading/editing the widget settings object.
type GuildWidgetSettings struct {
	Enabled   bool   `json:"enabled"`
	ChannelID string `json:"channel_id,omitempty"`
}

// GuildWidget fetches the guild's widget settings.
func (c *Client) GuildWidget(guildID string) (*GuildWidgetSettings, error) {
	endpoint := discordgo.EndpointGuildWidget(guildID)
	resp, err := c.RequestWithBucketID("GET", endpoint, nil, endpoint)
	if err != nil {
		return nil, err
	}

	var widget GuildWidgetSettings
	if err := json.Unmarshal(resp, &widget); err != nil {
		return nil, err
	}
	return &widget, nil
}

// GuildWidgetEdit updates the guild's widget settings.
func (c *Client) GuildWidgetEdit(guildID string, settings GuildWidgetSettings) error {
	endpoint := discordgo.EndpointGuildWidget(guildID)
	_, err := c.RequestWithBucketID("PATCH", endpoint, settings, endpoint)
	return err
}
