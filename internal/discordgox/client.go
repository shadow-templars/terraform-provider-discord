// Package discordgox extends the discordgo session with Discord API endpoints
// that discordgo does not yet model. It embeds *discordgo.Session, so every
// method discordgo already provides is available unchanged; only the missing
// endpoints are added here.
//
// This package is the single sanctioned home for raw Discord API access in the
// provider. Resource code must call these typed methods rather than issuing
// discordgo Request/RequestWithBucketID calls directly. Each method is written
// to be deletable: when discordgo ships an equivalent typed helper, the method
// here is removed and its call sites switch to the embedded session.
package discordgox

import "github.com/bwmarrin/discordgo"

// Client wraps a discordgo session and adds endpoints missing from discordgo.
type Client struct {
	*discordgo.Session
}

// New wraps a discordgo session.
func New(s *discordgo.Session) *Client {
	return &Client{Session: s}
}
