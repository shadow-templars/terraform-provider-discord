data "discord_color" "purple" {
  hex = "#5b2c8e"
}

resource "discord_role" "example" {
  server_id = var.server_id
  name      = "Members"
  color     = data.discord_color.purple.dec
}
