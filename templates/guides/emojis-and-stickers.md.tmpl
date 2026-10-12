---
page_title: "Emojis and Stickers"
subcategory: "Emojis and Stickers"
description: |-
  Choose emoji names and migrate expression ownership.
---

# Emojis and stickers

Use [individual emojis](../resources/guild_emoji.md) with shared name prefixes such as `class_monk` and `class_wizard` for picker grouping. Emoji collections are deprecated; upload sequence does not control emoji picker order.

For stickers, choose [individual resources](../resources/guild_sticker.md) for independent updates or a [collection](../resources/sticker_collection.md) for sequential uploads.

## Before enabling collection rebuilds

- `prevent_destroy` does not block an in-place rebuild. Keep `rebuild_on_change = false` to refuse one.
- Artwork must be available and unchanged between plan and apply.
- After a partial failure, review the next plan before retrying. Abrupt termination or a lost API response can leave objects outside state; reconcile their IDs first.

## Migrating individual stickers

1. Record each managed Discord ID and assign its collection key.
2. Remove individual resource configuration and relinquish its state using `removed` blocks with `destroy = false`. Never let both resources own the same ID.
3. Import the IDs using the collection's documented format. The first configured apply rebuilds the adopted entries and requires `rebuild_on_change = true`.

Keep individual resources if you do not want a full reupload.

## Leaving an emoji collection

Record its `ids` map, relinquish collection ownership using a `removed` block with `destroy = false`, and import those IDs into individual emoji resources. Do not destroy the collection after importing its IDs elsewhere. Imported artwork is unknown; configuring managed images replaces the adopted emojis.

## Adopting an individual emoji

Import cannot recover its artwork. To retain the existing image, configure `image` and temporarily use `lifecycle { ignore_changes = [image] }`. Match existing role restrictions. Remove the lifecycle rule when ready to replace the artwork.
