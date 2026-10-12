---
page_title: "Migrating from Lucky3028"
description: |-
  Review resource compatibility and transfer provider state when migrating from the discontinued Lucky3028 provider.
---

# Migrating from Lucky3028

[Lucky3028/terraform-provider-discord](https://github.com/Lucky3028/terraform-provider-discord) is archived and its README marks it discontinued. This provider began as a fork and was rewritten on the Terraform Plugin Framework. It is maintained independently; changing the provider address does not guarantee compatibility with every upstream configuration or state schema.

## Check compatibility

Compare your resource types and attributes with the [current reference](https://registry.terraform.io/providers/shadow-templars/discord/latest/docs) and [examples](https://github.com/shadow-templars/terraform-provider-discord/tree/main/examples/). Upstream's `discord_server` and `discord_news_channel` resources are not available here. `discord_managed_server` manages an existing guild's settings; `discord_text_channel` creates ordinary text channels, so these are not automatic replacements for the missing resources.

Keep existing resource addresses where compatible. Resolve unsupported resources and attribute differences before planning a migration. Individual emojis and stickers can remain individual resources; moving them into collections is a separate [ownership migration](emojis-and-stickers.md#migrating-individual-resources) that recreates the adopted expressions.

## Transfer provider ownership

1. Run `tofu providers` (or `terraform providers`) to identify the exact provider address recorded in state. Preserve a state backup using your backend's supported backup mechanism.
2. Update `required_providers` to use `shadow-templars/discord` and the version shown in the [README](https://github.com/shadow-templars/terraform-provider-discord/blob/main/README.md#get-started).
3. Use `tofu state replace-provider OLD_PROVIDER_ADDRESS NEW_PROVIDER_ADDRESS`, substituting the complete addresses. Match the target registry hostname to your configuration: unqualified addresses default to `registry.opentofu.org` in OpenTofu and `registry.terraform.io` in Terraform. This transfers the provider address; it does not translate resource schemas.
4. Run `tofu init -upgrade`, then `tofu plan`. Review all changes, especially replacements and deletions, before applying through your normal deployment process.

If a resource cannot be adopted through the existing state, follow its documented import format. Do not assume every upstream resource can be imported under a different type without changing its behavior.
