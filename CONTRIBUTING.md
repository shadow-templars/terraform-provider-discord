# Contributing

## Development

```bash
# Build
go build ./...

# Run tests
go test ./...

# Regenerate documentation (docs/)
go generate ./...
```

Documentation under `docs/` is generated from the schema with `tfplugindocs`;
edit schema descriptions and example files rather than the generated Markdown,
then run `go generate ./...`.

### Git hooks

This repo uses [pre-commit](https://pre-commit.com) for `gofmt` and Conventional
Commit validation. Enable it once after cloning:

```bash
pre-commit install --hook-type pre-commit --hook-type commit-msg
```

## Project structure

```
├── main.go
├── internal/
│   ├── client/       # Discord client wrapper passed to resources
│   ├── discordgox/   # Extension layer for endpoints discordgo does not model
│   ├── provider/     # Provider configuration and registration
│   └── service/
│       ├── channel/  # Channel resources (text, voice, category, forum, permission, invite, webhook)
│       ├── datasource/
│       ├── guild/    # Guild resources (managed server, community, onboarding, stickers, widget, ...)
│       └── role/     # Role resources (role, @everyone, member roles)
```

## discordgox

`internal/discordgox` embeds `*discordgo.Session` and adds the Discord API
endpoints discordgo does not yet model (currently guild Community edits with
`safety_alerts_channel_id`, guild stickers, and the server widget).

It is the only place in the provider that issues raw Discord API requests.
Resource code must call its typed methods rather than
`RequestWithBucketID`/`RequestRaw` directly. Each method is written to be
removed once discordgo ships an equivalent helper: delete the method here and
switch its callers to the embedded session.

When adding an endpoint discordgo lacks, put it in `discordgox` following the
existing methods, and prefer discordgo's own `Endpoint*` helpers and request
primitives so rate limiting and auth are inherited.

## Releasing

Releases are cut by pushing a `vMAJOR.MINOR.PATCH` tag, which triggers the
GoReleaser workflow. Update [`CHANGELOG.md`](CHANGELOG.md) before tagging.
