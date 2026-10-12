data "discord_permission" "read_only" {
  view_channel         = "allow"
  read_message_history = "allow"
  send_messages        = "deny"
}

resource "discord_channel_permission" "read_only" {
  channel_id   = var.channel_id
  overwrite_id = var.role_id
  type         = "role"
  allow        = data.discord_permission.read_only.allow_bits
  deny         = data.discord_permission.read_only.deny_bits
}
