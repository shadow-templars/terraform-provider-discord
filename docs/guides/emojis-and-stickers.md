---
page_title: "Emojis and Stickers"
subcategory: "Emojis and Stickers"
description: |-
  Choose resource ownership and migrate existing expressions into ordered collections.
---

# Emojis and stickers

Use individual resources for independent updates. Use a collection when upload order matters:

- [Individual emojis](../resources/guild_emoji.md) · [Individual stickers](../resources/guild_sticker.md)
- [Emoji collections](../resources/emoji_collection.md) · [Sticker collections](../resources/sticker_collection.md)

## Before enabling collection rebuilds

- `prevent_destroy` does not block an in-place rebuild. Keep `rebuild_on_change = false` to refuse one.
- Artwork must be available and unchanged between plan and apply.
- After a partial failure, review the next plan before retrying. Abrupt termination or a lost API response can leave objects outside state; reconcile their IDs first.

## Migrating individual resources

1. Record each managed Discord ID and assign its collection key.
2. Remove individual resource configuration and relinquish its state using `removed` blocks with `destroy = false`. Never let both resources own the same ID.
3. Import the IDs using the collection's documented format. The first configured apply rebuilds the adopted entries and requires `rebuild_on_change = true`.

Keep individual resources if you do not want a full reupload.

## Adopting an individual emoji

Import cannot recover its artwork. To retain the existing image, configure `image` and temporarily use `lifecycle { ignore_changes = [image] }`. Match existing role restrictions. Remove the lifecycle rule when ready to replace the artwork.
