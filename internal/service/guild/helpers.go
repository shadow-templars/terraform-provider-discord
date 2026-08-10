package guild

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

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
