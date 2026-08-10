package channel

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// syncChannelPermissions copies permission overwrites from the parent category
// to the given channel.
func syncChannelPermissions(session *discordgo.Session, channelID, parentID string) error {
	parent, err := session.Channel(parentID)
	if err != nil {
		return err
	}

	overwrites := make([]*discordgo.PermissionOverwrite, len(parent.PermissionOverwrites))
	for i, po := range parent.PermissionOverwrites {
		overwrites[i] = &discordgo.PermissionOverwrite{
			ID:    po.ID,
			Type:  po.Type,
			Deny:  po.Deny,
			Allow: po.Allow,
		}
	}

	_, err = session.ChannelEdit(channelID, &discordgo.ChannelEdit{
		PermissionOverwrites: overwrites,
	})
	return err
}

// isNotFound checks if the error from discordgo indicates a 404.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	restErr, ok := err.(*discordgo.RESTError)
	if ok && restErr.Response != nil && restErr.Response.StatusCode == 404 {
		return true
	}
	return strings.Contains(err.Error(), "404")
}

// boolPtr returns a pointer to a bool value.
func boolPtr(v bool) *bool {
	return &v
}
