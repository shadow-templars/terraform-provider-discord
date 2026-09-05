# Terraform Provider: Discord

A Terraform/OpenTofu provider for managing Discord servers declaratively.

> **v3 rewrite** — This is a complete rewrite using the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework), replacing the legacy SDK v2 implementation.

## Requirements

- [OpenTofu](https://opentofu.org/) >= 1.6 or [Terraform](https://www.terraform.io/) >= 1.0
- [Go](https://go.dev/) >= 1.25 (for building from source)
- A Discord bot token with appropriate permissions

## Usage

```hcl
terraform {
  required_providers {
    discord = {
      source  = "shadow-templars/discord"
      version = "~> 3.0"
    }
  }
}

provider "discord" {
  token = var.discord_bot_token  # or set DISCORD_TOKEN env var
}
```

## Resources

| Resource                       | Group    | Description                                                                    |
| ------------------------------ | -------- | ------------------------------------------------------------------------------ |
| `discord_text_channel`         | Channels | Manage text channels                                                           |
| `discord_voice_channel`        | Channels | Manage voice channels                                                          |
| `discord_category_channel`     | Channels | Manage category channels                                                       |
| `discord_forum_channel`        | Channels | Manage forum channels                                                          |
| `discord_channel_permission`   | Channels | Manage channel permission overwrites                                           |
| `discord_invite`               | Channels | Manage invites                                                                 |
| `discord_webhook`              | Channels | Manage channel webhooks                                                        |
| `discord_role`                 | Roles    | Manage roles                                                                   |
| `discord_role_everyone`        | Roles    | Manage the `@everyone` role                                                    |
| `discord_member_roles`         | Roles    | Assign roles to a member                                                       |
| `discord_managed_server`       | Server   | Manage settings of an existing server                                          |
| `discord_community`            | Server   | Enable and configure the Community feature (see [Community](#community) below) |
| `discord_system_channel`       | Server   | Manage the system channel                                                      |
| `discord_server_widget`        | Server   | Manage the server widget                                                       |
| `discord_server_onboarding`    | Server   | Manage the onboarding flow                                                     |
| `discord_message`              | Server   | Manage messages (e.g. welcome embeds)                                          |
| `discord_guild_sticker`        | Server   | Manage custom stickers                                                         |
| `discord_auto_moderation_rule` | Server   | Manage AutoMod rules                                                           |

## Community

The `discord_community` resource enables and configures the Discord **Community** feature. Its existence is the enablement signal: creating it turns a regular server into a Community server, and destroying it turns Community off.

```hcl
resource "discord_community" "example" {
  server_id                 = var.server_id
  rules_channel_id          = discord_text_channel.rules.id
  public_updates_channel_id = discord_text_channel.mod_updates.id
  preferred_locale          = "en-US" # optional, defaults to en-US
}
```

### Managed fields

These mirror Discord's **Community Settings** page:

| Field                       | Community Settings label    | Notes                                                                            |
| --------------------------- | --------------------------- | -------------------------------------------------------------------------------- |
| `rules_channel_id`          | Rules or guidelines channel | Must be viewable by `@everyone`.                                                 |
| `public_updates_channel_id` | Community Updates Channel   | Where Discord posts admin/mod notices. A role-restricted channel is recommended. |
| `preferred_locale`          | Server Primary Language     | Defaults to `en-US`.                                                             |

### Deliberately not managed

- **Safety Notifications Channel** (`safety_alerts_channel_id`): not exposed by discordgo v0.29.0, so it cannot be managed yet. Planned for a later release. Set it manually in Server Settings for now.
- **Server Description**: intentionally owned by `discord_managed_server`. Managing the same field in two resources would cause them to fight on every plan.

### Preconditions

Discord requires the server to already have `verification_level >= 1` and `explicit_content_filter = 2` before Community can be enabled (set via `discord_managed_server`). If unmet, the API rejects the edit and the apply fails.

### Dependency: announcement channels

Announcement channels (`discord_news_channel`, Discord type 5) require the `NEWS` feature, which is only granted once `COMMUNITY` is enabled. Any announcement channel must therefore `depends_on` the `discord_community` resource.

### Destroying

Destroying `discord_community` **disables Community** (strips the `COMMUNITY` feature). This is consequential: it also tears down the welcome screen, onboarding prompts, and breaks announcement channels. In production, guard the resource with `lifecycle { prevent_destroy = true }`.

### Importing

The resource ID is the server ID:

```bash
tofu import discord_community.example <server_id>
```

## Data Sources

| Data source           | Description                                    |
| --------------------- | ---------------------------------------------- |
| `discord_color`       | Resolve a color name/hex to its integer value  |
| `discord_permission`  | Compute permission bit values                  |
| `discord_local_image` | Load a local image as a data URI               |
| `discord_role`        | Look up an existing role                       |

## Development

```bash
# Build
go build ./...

# Run tests
go test ./...

# Install locally for testing
go install .
```

### Project Structure

```
├── main.go                          # Provider server entrypoint
├── internal/
│   ├── client/                      # Discord API client wrapper
│   ├── provider/                    # Provider configuration and registration
│   └── service/
│       └── channel/                 # Channel resources (text, category, voice, forum)
```

## Migration from v2

This provider is a fork of [Lucky3028/terraform-provider-discord](https://github.com/Lucky3028/terraform-provider-discord) rewritten from scratch using the Plugin Framework. The resource schemas are designed to be compatible where possible, but this is a new major version with potential breaking changes.

To migrate:

1. Update `required_providers` source to `shadow-templars/discord`
2. Run `tofu state replace-provider Lucky3028/discord shadow-templars/discord`
3. Run `tofu init -upgrade`
4. Run `tofu plan` to verify no unexpected changes

## License

[GPL-3.0](LICENSE)
