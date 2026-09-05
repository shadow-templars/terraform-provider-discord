package client

import "github.com/shadow-templars/terraform-provider-discord/internal/discordgox"

// DiscordClient wraps the Discord session and is passed to resources/data
// sources. Session is a discordgox client, which embeds *discordgo.Session and
// adds endpoints discordgo does not yet model.
type DiscordClient struct {
	Session *discordgox.Client
}
