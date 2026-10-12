data "discord_local_image" "emoji" {
  for_each = toset(["barbarian", "wizard", "salute"])
  file     = "${path.module}/emojis/${each.key}.png"
}

resource "discord_emoji_collection" "star" {
  server_id         = var.server_id
  rebuild_on_change = true

  # Blocks upload sequentially, from top to bottom.
  emoji {
    key   = "barbarian"
    name  = "star_barbarian"
    image = data.discord_local_image.emoji["barbarian"].data_uri
  }

  emoji {
    key   = "wizard"
    name  = "star_wizard"
    image = data.discord_local_image.emoji["wizard"].data_uri
  }

  emoji {
    key      = "templar_salute"
    name     = "star_templar_salute"
    image    = data.discord_local_image.emoji["salute"].data_uri
    role_ids = [discord_role.templar.id]
  }
}

output "barbarian_emoji_id" {
  value = discord_emoji_collection.star.ids["barbarian"]
}
