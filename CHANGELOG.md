# Changelog

All notable changes to this provider are documented here. The format is based
on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.2.0] - 2026-09-05

### Added

- `internal/discordgox`: an extension layer over the discordgo session that
  hosts Discord API endpoints discordgo does not yet model. It embeds
  `*discordgo.Session`, so existing typed calls are unchanged. This is the only
  place in the provider that issues raw Discord API requests.
- `safety_alerts_channel_id` on `discord_community`, applied and read through
  `discordgox` (the field is absent from discordgo's `GuildParams` and `Guild`
  structs in v0.29.0).
- Generated provider documentation (`docs/`) via `tfplugindocs`, wired as a
  `go:generate` step with a tracked tool dependency.

### Changed

- `discord_guild_sticker` and `discord_server_widget` now call `discordgox`
  methods instead of issuing raw `RequestWithBucketID`/`RequestRaw` calls
  inline. No behavior change.

## [3.1.0] - 2026-09-05

### Added

- `discord_community` resource to enable and configure the Discord Community
  feature (rules channel, public updates channel, preferred locale). Creating
  the resource enables Community; destroying it disables Community. Supports
  import by server ID.

### Fixed

- Reconciled the README resource and data-source lists with the resources the
  provider actually registers.

## [3.0.0] - 2026-08-11

### Changed

- Complete rewrite onto the Terraform Plugin Framework, replacing the legacy
  SDKv2 implementation. See the README migration notes for moving from the
  upstream `Lucky3028/discord` provider.

[3.2.0]: https://github.com/shadow-templars/terraform-provider-discord/releases/tag/v3.2.0
[3.1.0]: https://github.com/shadow-templars/terraform-provider-discord/releases/tag/v3.1.0
[3.0.0]: https://github.com/shadow-templars/terraform-provider-discord/releases/tag/v3.0.0
