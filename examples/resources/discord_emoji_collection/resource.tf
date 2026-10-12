resource "discord_emoji_collection" "star" {
  server_id         = var.server_id
  rebuild_on_change = true

  # Blocks upload sequentially, from top to bottom.
  emoji {
    key   = "barbarian"
    name  = "star_barbarian"
    image = "data:image/png;base64,${filebase64("${path.module}/emojis/barbarian.png")}"
  }

  emoji {
    key   = "wizard"
    name  = "star_wizard"
    image = "data:image/png;base64,${filebase64("${path.module}/emojis/wizard.png")}"
  }

  emoji {
    key      = "templar_salute"
    name     = "star_templar_salute"
    image    = "data:image/png;base64,${filebase64("${path.module}/emojis/salute.png")}"
    role_ids = [discord_role.templar.id]
  }
}

output "barbarian_emoji_id" {
  value = discord_emoji_collection.star.ids["barbarian"]
}
