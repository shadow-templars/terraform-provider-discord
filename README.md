# Discord Provider for Terraform and OpenTofu

[![Release](https://img.shields.io/github/v/release/shadow-templars/terraform-provider-discord)](https://github.com/shadow-templars/terraform-provider-discord/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/shadow-templars/terraform-provider-discord/ci.yml?branch=main)](https://github.com/shadow-templars/terraform-provider-discord/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/shadow-templars/terraform-provider-discord)](LICENSE)

Manage Discord servers, channels, roles, permissions, moderation, emojis, and stickers as code.

A maintained successor to [Lucky3028/terraform-provider-discord](https://github.com/Lucky3028/terraform-provider-discord), whose repository is archived and marked discontinued. Built on the Terraform Plugin Framework, with signed releases for Linux, macOS, and Windows.

## Get started

Requires OpenTofu >= 1.6 or Terraform >= 1.0, and a Discord bot token with permissions for the resources you manage.

```hcl
terraform {
  required_providers {
    discord = {
      source  = "shadow-templars/discord"
      version = "~> 3.5.1"
    }
  }
}

provider "discord" {}
```

Set `DISCORD_TOKEN` to your bot token, then run `tofu init` and `tofu plan` (or the equivalent Terraform commands).

## What you can manage

| Area | Resources |
| --- | --- |
| Servers | Server settings, Community, onboarding, and widgets |
| Channels | Categories, text, voice, forums, messages, invites, and webhooks |
| Access | Roles, member roles, and channel permissions |
| Moderation | Auto Moderation rules |
| Expressions | Individual emojis and stickers, or ordered collections with sequential uploads |

## Documentation

- [Terraform Registry reference](https://registry.terraform.io/providers/shadow-templars/discord/latest/docs) · [OpenTofu Registry](https://search.opentofu.org/provider/shadow-templars/discord/latest)
- [Emojis and stickers](docs/guides/emojis-and-stickers.md) — artwork, access restrictions, ordered rebuilds, and imports
- [Community servers](docs/guides/community.md) — prerequisites, lifecycle, and ownership
- [Migrating from Lucky3028](docs/guides/migration.md) — compatibility checks and provider state addresses
- [Examples](examples/) · [Changelog](CHANGELOG.md) · [Contributing](CONTRIBUTING.md)

Ordered collection rebuilds delete and recreate every managed entry, changing its Discord ID. Read the [expression guide](docs/guides/emojis-and-stickers.md) before enabling `rebuild_on_change`.
