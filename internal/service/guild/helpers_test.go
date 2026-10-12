package guild

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestIsNotFound(t *testing.T) {
	missing := &discordgo.RESTError{Response: &http.Response{StatusCode: http.StatusNotFound}}
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"404", missing, true},
		{"wrapped 404", fmt.Errorf("reading emoji: %w", missing), true},
		{"403 mentioning 404", &discordgo.RESTError{Response: &http.Response{StatusCode: http.StatusForbidden}, ResponseBody: []byte("emoji 404 is forbidden")}, false},
		{"untyped 404", errors.New("request 404 failed"), false},
		{"missing response", &discordgo.RESTError{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isNotFound(c.err); got != c.want {
				t.Errorf("isNotFound: got %t, want %t", got, c.want)
			}
		})
	}
}
