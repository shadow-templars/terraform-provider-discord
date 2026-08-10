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

### Available

- `discord_category_channel` — Manage category channels
- `discord_text_channel` — Manage text channels

### Planned

- `discord_voice_channel`
- `discord_forum_channel`
- `discord_role`
- `discord_channel_permission`
- `discord_channel_order`
- `discord_invite`
- `discord_webhook`
- `discord_message`
- `discord_server`
- `discord_guild_sticker`
- `discord_member_roles`

## Data Sources

### Planned

- `discord_permission`
- `discord_color`
- `discord_local_image`
- `discord_role`
- `discord_server`
- `discord_member`

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
