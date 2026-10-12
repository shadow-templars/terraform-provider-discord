package discordgox

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

type emojiTransport func(*http.Request) (*http.Response, error)

func (f emojiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGuildEmojiEdit(t *testing.T) {
	cases := []struct {
		name    string
		roles   []string
		status  int
		body    string
		wantErr bool
	}{
		{"clear nil", nil, 200, `{"id":"456","name":"renamed","roles":[]}`, false},
		{"clear empty", []string{}, 200, `{"id":"456","name":"renamed","roles":[]}`, false},
		{"restrict", []string{"789"}, 200, `{"id":"456","name":"renamed","roles":["789"]}`, false},
		{"permission error", nil, 403, `{"code":50013,"message":"Missing Permissions"}`, true},
		{"invalid response", nil, 200, `invalid JSON`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := discordgo.New("Bot test")
			if err != nil {
				t.Fatal(err)
			}
			s.MaxRestRetries = 0
			ctx := context.WithValue(context.Background(), struct{}{}, "request context")
			s.Client = &http.Client{Transport: emojiTransport(func(req *http.Request) (*http.Response, error) {
				if req.Context().Value(struct{}{}) != "request context" {
					t.Fatal("request context was not propagated")
				}
				if req.Method != http.MethodPatch || !strings.HasSuffix(req.URL.Path, "/guilds/123/emojis/456") {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				var payload map[string]json.RawMessage
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				wantRoles := `[]`
				if len(c.roles) != 0 {
					wantRoles = `["789"]`
				}
				if string(payload["roles"]) != wantRoles || string(payload["name"]) != `"renamed"` || len(payload) != 2 {
					t.Fatalf("wrong edit payload: %s", payload)
				}
				return &http.Response{StatusCode: c.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(c.body)), Request: req}, nil
			})}
			emoji, err := New(s).GuildEmojiEdit("123", "456", &discordgo.EmojiParams{Name: "renamed", Roles: c.roles}, discordgo.WithContext(ctx))
			if (err != nil) != c.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if !c.wantErr && (emoji.ID != "456" || emoji.Name != "renamed") {
				t.Fatalf("incorrect emoji: %+v", emoji)
			}
		})
	}
}
