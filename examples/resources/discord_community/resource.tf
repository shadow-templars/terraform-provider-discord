resource "discord_community" "example" {
  server_id                 = var.server_id
  rules_channel_id          = discord_text_channel.rules.id
  public_updates_channel_id = discord_text_channel.mod_updates.id
  preferred_locale          = "en-US"

  lifecycle {
    prevent_destroy = true
  }
}
