---
page_title: "Migrating from Lucky3028"
description: |-
  Check resource compatibility and transfer provider ownership.
---

# Migrating from Lucky3028

Check your configuration against the [resource reference](https://registry.terraform.io/providers/shadow-templars/discord/latest/docs) before changing provider addresses.

| Unsupported upstream resource | Compatibility |
| --- | --- |
| `discord_server` | `discord_managed_server` configures existing guilds; it does not create them. |
| `discord_news_channel` | `discord_text_channel` creates ordinary text channels, not announcement channels. |

## Migration

1. Back up state. Run `tofu providers` to identify its current provider address.
2. Update `required_providers` using the [installation example](https://github.com/shadow-templars/terraform-provider-discord/blob/main/README.md#get-started).
3. Run `tofu state replace-provider OLD_PROVIDER_ADDRESS NEW_PROVIDER_ADDRESS` with complete addresses. Use the registry hostname matching your configuration.
4. Run `tofu init -upgrade` and `tofu plan`. Resolve unexpected replacements or deletions before applying.

Use equivalent `terraform` commands if applicable. Replacing a provider address does not convert incompatible resource schemas.

Moving individual expressions into collections is a separate [migration](emojis-and-stickers.md#migrating-individual-resources) that recreates their Discord IDs.
