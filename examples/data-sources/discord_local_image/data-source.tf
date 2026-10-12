data "discord_local_image" "icon" {
  file = "${path.module}/assets/icon.png"
}

resource "discord_managed_server" "example" {
  server_id     = var.server_id
  name          = "Example Server"
  icon_data_uri = data.discord_local_image.icon.data_uri
}
