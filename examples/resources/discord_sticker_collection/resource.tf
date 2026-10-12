resource "discord_sticker_collection" "star" {
  server_id         = var.server_id
  rebuild_on_change = true

  # File content changes are detected even when these paths stay the same.
  sticker {
    key         = "welcome"
    name        = "Welcome Templar"
    description = "Welcome to the clan"
    tags        = "👋"
    file        = "${path.module}/stickers/welcome.png"
  }

  sticker {
    key         = "ready"
    name        = "Ready"
    description = "Ready for battle"
    tags        = "✅"
    file        = "${path.module}/stickers/ready.png"
  }
}
