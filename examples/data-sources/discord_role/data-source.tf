data "discord_role" "members" {
  server_id = var.server_id
  name      = "Members"
}

output "members_role_id" {
  description = "ID of the existing Members role."
  value       = data.discord_role.members.id
}
