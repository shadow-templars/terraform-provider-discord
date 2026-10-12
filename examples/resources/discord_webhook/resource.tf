data "discord_local_image" "avatar" {
  file = "${path.module}/assets/avatar.png"
}

resource "discord_webhook" "example" {
  channel_id      = var.channel_id
  name            = "Notifications"
  avatar_data_uri = data.discord_local_image.avatar.data_uri
}
