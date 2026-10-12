package guild

import (
	"errors"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

func isNotFound(err error) bool {
	var restErr *discordgo.RESTError
	return errors.As(err, &restErr) && restErr != nil && restErr.Response != nil && restErr.Response.StatusCode == http.StatusNotFound
}
