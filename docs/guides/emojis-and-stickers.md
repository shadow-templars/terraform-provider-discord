---
page_title: "Emojis and Stickers"
subcategory: "Emojis and Stickers"
description: |-
  Choose individual expressions or ordered collections, manage artwork and roles, and migrate existing IDs.
---

# Emojis and stickers

## Emojis

`discord_guild_emoji` manages static and animated custom emojis. Feed `image` from `discord_local_image.data_uri` so changing the file contents is detected, even when the path stays the same. Names and `role_ids` update in place; omit roles or use an empty set to allow everyone.

Discord cannot update an emoji's image. Changing `image` or `server_id` replaces the emoji and changes its ID; existing reactions still refer to the old ID. The default destroy-before-create replacement avoids requiring a spare emoji slot.

Import uses `server_id:emoji_id`. Discord does not return the original image data, so the imported `image` remains null. Configure `image` and use `lifecycle { ignore_changes = [image] }` to adopt the existing emoji without replacing it; remove that lifecycle rule when ready to manage a new image. Match any imported role restrictions in `role_ids` to retain them.

See the [generated emoji reference](../resources/guild_emoji.md) for the schema and upload example. The bot needs `CREATE_GUILD_EXPRESSIONS`; editing or deleting emojis created by others needs `MANAGE_GUILD_EXPRESSIONS`.

## Ordered collections

Use `discord_local_image.data_uri` for each emoji's `image`. Discord accepts emoji artwork as a base64 data URI, while stickers require a multipart file upload. Set each sticker's `file` to its local path; the collection fingerprints the file contents to detect artwork changes at the same path. The image data source reads and encodes files; collection resources own rebuild decisions.

Use `discord_emoji_collection` or `discord_sticker_collection` when upload order matters. Nested `emoji` / `sticker` blocks form an ordered list: the provider uploads one entry, verifies its returned ID and metadata, then proceeds to the next. Keep all guild emojis in one collection if they must share a single sequence. Separate collections and individual resources can still execute concurrently; use an explicit `depends_on` if sticker and emoji collections should also wait for one another.

New collections upload without deleting existing, unowned expressions. Subsequent entry additions, removals, reorders, metadata changes, local artwork changes, or observed drift require `rebuild_on_change = true`. The default is false and refuses such updates. Enabling this flag alone does not rebuild an unchanged collection.

**A rebuild deletes every ID owned by that collection before uploading replacements. All managed IDs change, existing emoji/sticker references may become invalid, and the collection may be unavailable for an extended period while Discord enforces rate limits.** The provider emits a warning during planning. Unmanaged expressions are never deleted. Do not use `create_before_destroy`; for guild changes it can interleave uploads and exceed capacity. This is an in-place rebuild, so `prevent_destroy` alone does not guard it; leave `rebuild_on_change` false to refuse rebuilding.

Keys identify entries in the computed `ids` map; block order controls upload order. Emoji role restrictions are preserved during recreation. Sticker files are hashed by content, so replacing bytes at the same path produces a change. Files must be available when planning and applying; changes between those stages are rejected before deletion.

Refresh detects missing managed IDs and changed names, roles, descriptions or tags. It does not adopt a replacement with the same name: replacement IDs are unowned and must be explicitly imported if desired. Discord does not expose picker positions or original image hashes, so remote artwork and client display-order drift cannot be detected. The provider enforces creation sequence within the collection, not an undocumented client sorting contract.

When an API request or verification fails, uploads stop. IDs successfully returned before the failure remain in the error response state. On the next apply a rebuild cleans up that partial collection before starting again; a failed initial creation may be tainted and replaced by Terraform. Partial deletion retains only the remaining IDs. This is not transactional: abrupt process termination or a response lost after Discord accepted a POST can prevent Terraform from saving an ID. Inspect and reconcile the guild after such a failure rather than assuming rollback.

## Migrating individual resources

A collection cannot share ownership with individual expression resources. Capture their current Discord IDs, remove the individual resource configuration, and relinquish its state without deleting the remote objects (for example with `removed` blocks using `lifecycle { destroy = false }`, where supported). Then import only those IDs into the matching collection using stable keys:

```hcl
import {
  to = discord_emoji_collection.star
  id = "GUILD_ID:barbarian=EMOJI_ID,wizard=OTHER_EMOJI_ID"
}
```

Match the keys to the configured nested blocks. Import cannot recover artwork or the configured upload sequence: the first configured plan explicitly rebuilds the adopted IDs and requires `rebuild_on_change = true`. Never leave an individual resource managing an imported ID.

See the generated [emoji collection reference](../resources/emoji_collection.md) and [sticker collection reference](../resources/sticker_collection.md) for ordered examples and schema details.

