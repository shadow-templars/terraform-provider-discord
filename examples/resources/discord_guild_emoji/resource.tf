data "discord_local_image" "emoji" {
  file = "${path.module}/assets/star-yes.png"
}

resource "discord_guild_emoji" "example" {
  server_id = var.server_id
  name      = "star_yes"
  image     = data.discord_local_image.emoji.data_uri

  # Omit role_ids to allow everyone. An empty set clears existing restrictions.
  # role_ids = ["123456789012345678"]
}
