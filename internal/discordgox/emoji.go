package discordgox

import (
	"encoding/json"

	"github.com/bwmarrin/discordgo"
)

// GuildEmojiEdit sends roles even when empty so restrictions can be cleared.
// discordgo v0.29.0 omits empty EmojiParams.Roles from its PATCH payload.
func (c *Client) GuildEmojiEdit(guildID, emojiID string, data *discordgo.EmojiParams, options ...discordgo.RequestOption) (*discordgo.Emoji, error) {
	roles := data.Roles
	if roles == nil {
		roles = []string{}
	}
	params := struct {
		Name  string   `json:"name"`
		Roles []string `json:"roles"`
	}{Name: data.Name, Roles: roles}

	body, err := c.RequestWithBucketID("PATCH", discordgo.EndpointGuildEmoji(guildID, emojiID), params, discordgo.EndpointGuildEmojis(guildID), options...)
	if err != nil {
		return nil, err
	}
	var emoji discordgo.Emoji
	if err := json.Unmarshal(body, &emoji); err != nil {
		return nil, err
	}
	return &emoji, nil
}
