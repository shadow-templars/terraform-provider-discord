resource "discord_guild_sticker" "welcome" {
  server_id   = var.server_id
  name        = "Welcome"
  description = "Welcome to the server"
  tags        = "👋"
  file        = "${path.module}/stickers/welcome.png"
}
