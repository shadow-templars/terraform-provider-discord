package client

import "github.com/bwmarrin/discordgo"

// DiscordClient wraps the discordgo session and is passed to resources/data sources.
type DiscordClient struct {
	Session *discordgo.Session
}
