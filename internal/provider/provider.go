package provider

import (
	"context"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
	"github.com/shadow-templars/terraform-provider-discord/internal/service/channel"
	discorddata "github.com/shadow-templars/terraform-provider-discord/internal/service/datasource"
	"github.com/shadow-templars/terraform-provider-discord/internal/service/guild"
	"github.com/shadow-templars/terraform-provider-discord/internal/service/role"
)

var _ provider.Provider = (*DiscordProvider)(nil)

// DiscordProvider implements the Terraform Plugin Framework provider for Discord.
type DiscordProvider struct {
	version string
}

// DiscordProviderModel maps provider schema data to a Go type.
type DiscordProviderModel struct {
	Token types.String `tfsdk:"token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DiscordProvider{
			version: version,
		}
	}
}

func (p *DiscordProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "discord"
	resp.Version = p.version
}

func (p *DiscordProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Discord servers declaratively with OpenTofu/Terraform.",
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Discord bot token (without the `Bot` prefix). Can also be set via the `DISCORD_TOKEN` environment variable.",
			},
		},
	}
}

func (p *DiscordProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config DiscordProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve token: config value takes precedence over environment variable.
	token := config.Token.ValueString()
	if token == "" {
		token = os.Getenv("DISCORD_TOKEN")
	}
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing Discord Token",
			"The `token` attribute or `DISCORD_TOKEN` environment variable must be set.",
		)
		return
	}

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Discord Client",
			"An unexpected error occurred creating the Discord API client: "+err.Error(),
		)
		return
	}
	session.UserAgent = "terraform-provider-discord/" + p.version

	c := &client.DiscordClient{Session: session}
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *DiscordProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		channel.NewTextChannelResource,
		channel.NewCategoryChannelResource,
		channel.NewVoiceChannelResource,
		channel.NewForumChannelResource,
		channel.NewChannelPermissionResource,
		channel.NewInviteResource,
		channel.NewWebhookResource,
		role.NewRoleResource,
		role.NewRoleEveryoneResource,
		role.NewMemberRolesResource,
		guild.NewGuildStickerResource,
		guild.NewManagedServerResource,
		guild.NewSystemChannelResource,
		guild.NewServerWidgetResource,
		guild.NewMessageResource,
		guild.NewAutoModerationRuleResource,
		guild.NewServerOnboardingResource,
	}
}

func (p *DiscordProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		discorddata.NewColorDataSource,
		discorddata.NewPermissionDataSource,
		discorddata.NewLocalImageDataSource,
		discorddata.NewRoleDataSource,
	}
}
