terraform {
  required_providers {
    discord = {
      source  = "shadow-templars/discord"
      version = "~> 3.5"
    }
  }
}

provider "discord" {
  token = var.discord_bot_token # or set DISCORD_TOKEN
}
