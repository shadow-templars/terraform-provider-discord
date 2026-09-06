# Terraform Provider: Discord

[![Release](https://img.shields.io/github/v/release/shadow-templars/terraform-provider-discord)](https://github.com/shadow-templars/terraform-provider-discord/releases)
[![Build](https://img.shields.io/github/actions/workflow/status/shadow-templars/terraform-provider-discord/release.yml)](https://github.com/shadow-templars/terraform-provider-discord/actions/workflows/release.yml)
[![License](https://img.shields.io/github/license/shadow-templars/terraform-provider-discord)](LICENSE)

A Terraform/OpenTofu provider for managing Discord servers declaratively, built on the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework).

## Requirements

- [OpenTofu](https://opentofu.org/) >= 1.6 or [Terraform](https://www.terraform.io/) >= 1.0
- [Go](https://go.dev/) >= 1.25 (for building from source)
- A Discord bot token with appropriate permissions

## Usage

```hcl
terraform {
  required_providers {
    discord = {
      source  = "shadow-templars/discord"
      version = "~> 3.3"
    }
  }
}

provider "discord" {
  token = var.discord_bot_token # or set DISCORD_TOKEN
}
```

## Documentation

The full reference for every resource and data source is generated from the schema and published on the Terraform Registry:

**<https://registry.terraform.io/providers/shadow-templars/discord/latest/docs>**

The same pages live in [`docs/`](docs/) in this repository, generated with `tfplugindocs` (see [CONTRIBUTING.md](CONTRIBUTING.md)). This README covers what the generated reference does not: setup, architecture, and the conceptual notes below.

## Community

`discord_community` is the one resource whose behaviour needs more than a field reference, so its concepts are documented here; see the [generated docs](https://registry.terraform.io/providers/shadow-templars/discord/latest/docs/resources/community) for the field-by-field reference.

- **Existence is the enablement signal.** Creating the resource turns a regular server into a Community server; destroying it turns Community off.
- **Preconditions.** Discord requires `verification_level >= 1` and `explicit_content_filter = 2` (set via `discord_managed_server`) before Community can be enabled. If unmet, the apply fails.
- **Announcement-channel dependency.** Announcement channels (`discord_news_channel`, type 5) need the `NEWS` feature, granted only once `COMMUNITY` is enabled, so they must `depends_on` this resource.
- **Destroying is consequential.** It strips the `COMMUNITY` feature, which also tears down the welcome screen and onboarding prompts and breaks announcement channels. Guard it with `lifecycle { prevent_destroy = true }` in production.

### Not managed

- **Server Description**: owned by `discord_managed_server`; managing it in two resources would make them fight on every plan.
- **Membership Screening** (the rules-acceptance gate): Discord has withdrawn the get/edit API docs while the object is being reworked, so it cannot be managed responsibly yet.
- **Age-Restriction** (`nsfw_level`): readable but absent from Discord's Modify Guild parameters, so it cannot be written via the API.

## Migration from the upstream provider

This provider began as a fork of [Lucky3028/terraform-provider-discord](https://github.com/Lucky3028/terraform-provider-discord) and was rewritten onto the Plugin Framework. To migrate:

1. Point `required_providers` at `shadow-templars/discord`.
2. Run `tofu state replace-provider Lucky3028/discord shadow-templars/discord`.
3. Run `tofu init -upgrade`.
4. Run `tofu plan` and confirm no unexpected changes.
