# Changelog

All notable changes to this provider are documented here. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.5.3] - 2026-10-26

### Fixed

- Text and forum channels without a `topic` no longer plan a hidden `"" -> null` update on every run; an empty topic from Discord now reads as null.
- `discord_server_widget` treats an unset `channel_id` as unmanaged instead of planning to clear the channel Discord keeps on every run.

### Changed

- Update provider usage examples to require the 3.5.3 patch series.

## [3.5.2] - 2026-10-12

### Changed

- Deprecate emoji collections in favor of individual emojis with shared name prefixes; sequential uploads do not control picker grouping. Existing collection state remains supported for migration. Sticker collections are unchanged.
- Document naming guidance on the regular emoji resource and update examples to use class prefixes.
- Include the streamlined adoption guides, top-level data source grouping, and server, webhook, sticker, and data source examples added since 3.5.1.

## [3.5.1] - 2026-10-12

### Changed

- Group generated resource and data source documentation by Channels, Server Settings, Roles and Permissions, Emojis and Stickers, and Utilities using reusable generation templates.
- Use the existing `discord_local_image` data source in ordered emoji examples; sticker collections continue to accept file paths and detect artwork changes through file-content fingerprints.
- Update provider usage examples to require the 3.5.1 patch series.

## [3.5.0] - 2026-10-12

### Added

- Ordered `discord_emoji_collection` and `discord_sticker_collection` resources with list-ordered nested entries, sequential upload and verification, stable-key ID outputs, explicit opt-in rebuilds and destructive-change plan warnings. Rebuilds delete only collection-owned IDs before uploading any replacements.
- Collection drift checks for missing IDs and observable metadata, file-content fingerprinting for sticker artwork, partial-failure ownership tracking, and explicit-ID import for migrating individual resources. Remote image and picker-position drift are not exposed by Discord.

### Changed

- Guild sticker create, read and delete extension methods accept Discord request options, allowing collection requests to carry Terraform cancellation contexts. Existing callers remain compatible.

## [3.4.0] - 2026-10-12

### Added

- `discord_guild_emoji`: manage static and animated guild emojis with a base64 image data URI, editable names and optional role restrictions. Changing the image or server replaces the emoji; import uses `server_id:emoji_id`. Includes configuration validation and mocked HTTP lifecycle tests covering role clearing, null-versus-empty state, drift, import and API failures.
- `discordgox.GuildEmojiEdit`: explicitly sends empty role lists so existing restrictions can be cleared, unlike discordgo v0.29.0's edit payload.

### Fixed

- Guild resource 404 handling now checks typed HTTP responses, including wrapped errors, rather than matching `404` anywhere in an error message. Other API errors no longer risk removing resources from state because their message happens to contain those digits.

## [3.3.2] - 2026-09-07

### Fixed

- `discord_auto_moderation_rule`: map unset action-metadata fields to null instead of their zero values (`0` / `""`), completing the 3.3.1 fix. An action that sets only some metadata (e.g. an alert action with just `channel_id`) no longer triggers "Provider produced inconsistent result after apply". Covered by a `refreshState` unit test.

## [3.3.1] - 2026-09-07

### Fixed

- `discord_auto_moderation_rule`: avoid a "Provider produced inconsistent result after apply" error on rules without trigger metadata or action metadata (e.g. SPAM rules, BLOCK_MESSAGE actions). Discord returns empty metadata objects, which are now collapsed to null to match a configuration that omits them.

## [3.3.0] - 2026-09-06

### Added

- `mention_total_limit` and `mention_raid_protection_enabled` on `discord_auto_moderation_rule` trigger metadata, enabling MENTION_SPAM (trigger type 5) rules. `mention_raid_protection_enabled` is applied and read through `discordgox`, since discordgo's `AutoModerationTriggerMetadata` does not model it in v0.29.0.

## [3.2.0] - 2026-09-05

### Added

- `internal/discordgox`: an extension layer over the discordgo session that hosts Discord API endpoints discordgo does not yet model. It embeds `*discordgo.Session`, so existing typed calls are unchanged. This is the only place in the provider that issues raw Discord API requests.
- `safety_alerts_channel_id` on `discord_community`, applied and read through `discordgox` (the field is absent from discordgo's `GuildParams` and `Guild` structs in v0.29.0).
- Generated provider documentation (`docs/`) via `tfplugindocs`, wired as a `go:generate` step with a tracked tool dependency.

### Changed

- `discord_guild_sticker` and `discord_server_widget` now call `discordgox` methods instead of issuing raw `RequestWithBucketID`/`RequestRaw` calls inline. No behavior change.

## [3.1.0] - 2026-09-05

### Added

- `discord_community` resource to enable and configure the Discord Community feature (rules channel, public updates channel, preferred locale). Creating the resource enables Community; destroying it disables Community. Supports import by server ID.

### Fixed

- Reconciled the README resource and data-source lists with the resources the provider actually registers.

## [3.0.0] - 2026-08-11

### Changed

- Complete rewrite onto the Terraform Plugin Framework, replacing the legacy SDKv2 implementation. See the README migration notes for moving from the upstream `Lucky3028/discord` provider.

[3.5.3]: https://github.com/shadow-templars/terraform-provider-discord/releases/tag/v3.5.3
[3.5.2]: https://github.com/shadow-templars/terraform-provider-discord/releases/tag/v3.5.2
[3.5.1]: https://github.com/shadow-templars/terraform-provider-discord/releases/tag/v3.5.1
[3.5.0]: https://github.com/shadow-templars/terraform-provider-discord/releases/tag/v3.5.0
[3.4.0]: https://github.com/shadow-templars/terraform-provider-discord/releases/tag/v3.4.0
[3.3.0]: https://github.com/shadow-templars/terraform-provider-discord/releases/tag/v3.3.0
[3.2.0]: https://github.com/shadow-templars/terraform-provider-discord/releases/tag/v3.2.0
[3.1.0]: https://github.com/shadow-templars/terraform-provider-discord/releases/tag/v3.1.0
[3.0.0]: https://github.com/shadow-templars/terraform-provider-discord/releases/tag/v3.0.0
