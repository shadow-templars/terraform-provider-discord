resource "discord_text_channel" "officer_log" {
  server_id                = "123456789012345678"
  name                     = "officer-log"
  sync_perms_with_category = false

  permission_overwrites = [
    {
      id    = "123456789012345678" # @everyone
      type  = "role"
      allow = 0
      deny  = 3072 # View Channel and Send Messages
    },
    {
      id    = "234567890123456789" # Officers
      type  = "role"
      allow = 1024 # View Channel
      deny  = 0
    },
  ]
}
