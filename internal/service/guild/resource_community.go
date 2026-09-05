package guild

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
	"github.com/shadow-templars/terraform-provider-discord/internal/discordgox"
)

var (
	_ resource.Resource                = (*CommunityResource)(nil)
	_ resource.ResourceWithImportState = (*CommunityResource)(nil)
)

// CommunityResource enables and configures the Community feature of a Discord
// server. The resource's existence is the enablement signal: creating it turns
// a regular server into a Community server; destroying it turns Community off.
type CommunityResource struct {
	client *client.DiscordClient
}

type CommunityResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	ServerID               types.String `tfsdk:"server_id"`
	RulesChannelID         types.String `tfsdk:"rules_channel_id"`
	PublicUpdatesChannelID types.String `tfsdk:"public_updates_channel_id"`
	SafetyAlertsChannelID  types.String `tfsdk:"safety_alerts_channel_id"`
	PreferredLocale        types.String `tfsdk:"preferred_locale"`
}

func NewCommunityResource() resource.Resource {
	return &CommunityResource{}
}

func (r *CommunityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_community"
}

func (r *CommunityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Enables and configures the Community feature of a Discord server. " +
			"Creating this resource enables Community (adds the COMMUNITY guild feature); " +
			"destroying it disables Community. The server must already have verification_level >= 1 " +
			"and explicit_content_filter = 2 (set via discord_managed_server) before Community can be enabled.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Same as server_id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_id": schema.StringAttribute{
				Required:    true,
				Description: "The server (guild) ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rules_channel_id": schema.StringAttribute{
				Required:    true,
				Description: "Channel ID that hosts the server rules or guidelines. Must be viewable by @everyone.",
			},
			"public_updates_channel_id": schema.StringAttribute{
				Required:    true,
				Description: "Channel ID where Discord sends Community updates for admins and moderators. A role-restricted channel is recommended.",
			},
			"safety_alerts_channel_id": schema.StringAttribute{
				Optional:    true,
				Description: "Channel ID where Discord sends safety alerts for admins and moderators. A role-restricted channel is recommended.",
			},
			"preferred_locale": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("en-US"),
				Description: "The server's primary language (e.g. en-US). Used by Discord to customize features and in server discovery. Defaults to en-US.",
			},
		},
	}
}

func (r *CommunityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.DiscordClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.DiscordClient, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *CommunityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CommunityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = plan.ServerID
	r.enable(&plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CommunityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CommunityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	guild, err := r.client.Session.GuildWithExtras(state.ServerID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Server", err.Error())
		return
	}

	// Community being off means the resource no longer exists, so drop it from
	// state and let a plan re-enable it.
	if !hasCommunityFeature(guild.Features) {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(guild.ID)
	state.ServerID = types.StringValue(guild.ID)
	if guild.RulesChannelID != "" {
		state.RulesChannelID = types.StringValue(guild.RulesChannelID)
	}
	if guild.PublicUpdatesChannelID != "" {
		state.PublicUpdatesChannelID = types.StringValue(guild.PublicUpdatesChannelID)
	}
	if guild.SafetyAlertsChannelID != "" {
		state.SafetyAlertsChannelID = types.StringValue(guild.SafetyAlertsChannelID)
	}
	if guild.PreferredLocale != "" {
		state.PreferredLocale = types.StringValue(guild.PreferredLocale)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CommunityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CommunityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.enable(&plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete disables Community by stripping the COMMUNITY feature. This is a real,
// consequential un-enablement: it also tears down the welcome screen, onboarding
// prompts, and breaks announcement channels that depend on the NEWS feature.
func (r *CommunityResource) Delete(_ context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CommunityResourceModel
	resp.Diagnostics.Append(req.State.Get(context.Background(), &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	guild, err := r.client.Session.Guild(state.ServerID.ValueString())
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error Reading Server", err.Error())
		return
	}

	params := discordgo.GuildParams{
		Features: removeCommunityFeature(guild.Features),
	}
	if _, err := r.client.Session.GuildEdit(state.ServerID.ValueString(), &params); err != nil {
		resp.Diagnostics.AddError("Error Disabling Community", err.Error())
	}
}

func (r *CommunityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_id"), req, resp)
}

// enable turns Community on (merging COMMUNITY onto existing guild features) and
// applies the rules/updates/locale settings in a single guild edit. Discord
// requires all of these together to enable Community, so they are sent as one
// call rather than piecemeal.
func (r *CommunityResource) enable(plan *CommunityResourceModel, diags *diag.Diagnostics) {
	guild, err := r.client.Session.Guild(plan.ServerID.ValueString())
	if err != nil {
		diags.AddError("Error Reading Server", err.Error())
		return
	}

	params := discordgox.GuildCommunityParams{
		Features:               addCommunityFeature(guild.Features),
		RulesChannelID:         plan.RulesChannelID.ValueString(),
		PublicUpdatesChannelID: plan.PublicUpdatesChannelID.ValueString(),
		PreferredLocale:        plan.PreferredLocale.ValueString(),
	}
	if !plan.SafetyAlertsChannelID.IsNull() && !plan.SafetyAlertsChannelID.IsUnknown() {
		params.SafetyAlertsChannelID = plan.SafetyAlertsChannelID.ValueString()
	}

	if _, err := r.client.Session.GuildCommunityEdit(plan.ServerID.ValueString(), params); err != nil {
		diags.AddError("Error Enabling Community", err.Error())
	}
}

func hasCommunityFeature(features []discordgo.GuildFeature) bool {
	for _, f := range features {
		if f == discordgo.GuildFeatureCommunity {
			return true
		}
	}
	return false
}

func addCommunityFeature(features []discordgo.GuildFeature) []discordgo.GuildFeature {
	if hasCommunityFeature(features) {
		return features
	}
	return append(features, discordgo.GuildFeatureCommunity)
}

func removeCommunityFeature(features []discordgo.GuildFeature) []discordgo.GuildFeature {
	result := make([]discordgo.GuildFeature, 0, len(features))
	for _, f := range features {
		if f != discordgo.GuildFeatureCommunity {
			result = append(result, f)
		}
	}
	return result
}
