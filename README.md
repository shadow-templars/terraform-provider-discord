# Discord Provider for Terraform and OpenTofu

[![Release](https://img.shields.io/github/v/release/shadow-templars/terraform-provider-discord)](https://github.com/shadow-templars/terraform-provider-discord/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/shadow-templars/terraform-provider-discord/ci.yml?branch=main)](https://github.com/shadow-templars/terraform-provider-discord/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/shadow-templars/terraform-provider-discord)](LICENSE)

Manage Discord configuration as code: version server settings, review planned changes, and reproduce your setup. A maintained successor to the discontinued [Lucky3028 provider](https://github.com/Lucky3028/terraform-provider-discord).

## Capabilities

| Area | Supported |
| --- | --- |
| Servers | Settings, Community, onboarding, widgets |
| Channels | Categories, text, voice, forums, messages, invites, webhooks |
| Access | Roles, member roles, channel permissions |
| Moderation | Auto Moderation rules |
| Expressions | Emojis with name-based grouping; stickers with optional sequential collections |

## Example

```hcl
terraform {
  required_providers {
    discord = {
      source  = "shadow-templars/discord"
      version = "~> 3.5.3"
    }
  }
}

provider "discord" {}

resource "discord_text_channel" "general" {
  server_id = "123456789012345678"
  name      = "general"
}
```

Replace `server_id` with your guild ID. Set `DISCORD_TOKEN` to your bot token.

## Requirements

- OpenTofu >= 1.6 or Terraform >= 1.0
- Discord bot permissions for the resources you manage

## Documentation

- [Terraform reference](https://registry.terraform.io/providers/shadow-templars/discord/latest/docs) · [OpenTofu Registry](https://search.opentofu.org/provider/shadow-templars/discord/latest)
- [Expression adoption and rebuilds](docs/guides/emojis-and-stickers.md)
- [Migrating from Lucky3028](docs/guides/migration.md)
- [Examples](examples/) · [Changelog](CHANGELOG.md) · [Contributing](CONTRIBUTING.md)
