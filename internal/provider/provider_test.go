package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestGuildEmojiSchemaRegistration(t *testing.T) {
	server := providerserver.NewProtocol6(New("test")())()
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range resp.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("invalid provider schema: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
	if resp.ResourceSchemas["discord_guild_emoji"] == nil {
		t.Fatal("discord_guild_emoji is not registered in the provider schema")
	}
}
