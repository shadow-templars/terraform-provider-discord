package datasource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*PermissionDataSource)(nil)

// Discord permission bit flags.
// See: https://discord.com/developers/docs/topics/permissions#permissions-bitwise-permission-flags
const (
	permCreateInstantInvite     = 1 << 0
	permKickMembers             = 1 << 1
	permBanMembers              = 1 << 2
	permAdministrator           = 1 << 3
	permManageChannels          = 1 << 4
	permManageGuild             = 1 << 5
	permAddReactions            = 1 << 6
	permViewAuditLog            = 1 << 7
	permPrioritySpeaker         = 1 << 8
	permStream                  = 1 << 9
	permViewChannel             = 1 << 10
	permSendMessages            = 1 << 11
	permSendTTSMessages         = 1 << 12
	permManageMessages          = 1 << 13
	permEmbedLinks              = 1 << 14
	permAttachFiles             = 1 << 15
	permReadMessageHistory      = 1 << 16
	permMentionEveryone         = 1 << 17
	permUseExternalEmojis       = 1 << 18
	permViewGuildInsights       = 1 << 19
	permConnect                 = 1 << 20
	permSpeak                   = 1 << 21
	permMuteMembers             = 1 << 22
	permDeafenMembers           = 1 << 23
	permMoveMembers             = 1 << 24
	permUseVAD                  = 1 << 25
	permChangeNickname          = 1 << 26
	permManageNicknames         = 1 << 27
	permManageRoles             = 1 << 28
	permManageWebhooks          = 1 << 29
	permManageExpressions       = 1 << 30
	permUseAppCommands          = 1 << 31
	permRequestToSpeak          = 1 << 32
	permManageEvents            = 1 << 33
	permManageThreads           = 1 << 34
	permCreatePublicThreads     = 1 << 35
	permCreatePrivateThreads    = 1 << 36
	permUseExternalStickers     = 1 << 37
	permSendMessagesInThreads   = 1 << 38
	permUseEmbeddedActivities   = 1 << 39
	permModerateMembers         = 1 << 40
	permViewCreatorMonetization = 1 << 41
	permUseSoundboard           = 1 << 42
	permCreateExpressions       = 1 << 43
	permCreateEvents            = 1 << 44
	permUseExternalSounds       = 1 << 45
	permSendVoiceMessages       = 1 << 46
	permSendPolls               = 1 << 49
	permUseExternalApps         = 1 << 50
	permSetVoiceChannelStatus   = 1 << 48
)

type PermissionDataSource struct{}

type PermissionDataSourceModel struct {
	CreateInstantInvite     types.String `tfsdk:"create_instant_invite"`
	KickMembers             types.String `tfsdk:"kick_members"`
	BanMembers              types.String `tfsdk:"ban_members"`
	Administrator           types.String `tfsdk:"administrator"`
	ManageChannels          types.String `tfsdk:"manage_channels"`
	ManageGuild             types.String `tfsdk:"manage_guild"`
	AddReactions            types.String `tfsdk:"add_reactions"`
	ViewAuditLog            types.String `tfsdk:"view_audit_log"`
	PrioritySpeaker         types.String `tfsdk:"priority_speaker"`
	Stream                  types.String `tfsdk:"stream"`
	ViewChannel             types.String `tfsdk:"view_channel"`
	SendMessages            types.String `tfsdk:"send_messages"`
	SendTTSMessages         types.String `tfsdk:"send_tts_messages"`
	ManageMessages          types.String `tfsdk:"manage_messages"`
	EmbedLinks              types.String `tfsdk:"embed_links"`
	AttachFiles             types.String `tfsdk:"attach_files"`
	ReadMessageHistory      types.String `tfsdk:"read_message_history"`
	MentionEveryone         types.String `tfsdk:"mention_everyone"`
	UseExternalEmojis       types.String `tfsdk:"use_external_emojis"`
	ViewGuildInsights       types.String `tfsdk:"view_guild_insights"`
	Connect                 types.String `tfsdk:"connect"`
	Speak                   types.String `tfsdk:"speak"`
	MuteMembers             types.String `tfsdk:"mute_members"`
	DeafenMembers           types.String `tfsdk:"deafen_members"`
	MoveMembers             types.String `tfsdk:"move_members"`
	UseVAD                  types.String `tfsdk:"use_vad"`
	ChangeNickname          types.String `tfsdk:"change_nickname"`
	ManageNicknames         types.String `tfsdk:"manage_nicknames"`
	ManageRoles             types.String `tfsdk:"manage_roles"`
	ManageWebhooks          types.String `tfsdk:"manage_webhooks"`
	ManageExpressions       types.String `tfsdk:"manage_expressions"`
	UseAppCommands          types.String `tfsdk:"use_app_commands"`
	RequestToSpeak          types.String `tfsdk:"request_to_speak"`
	ManageEvents            types.String `tfsdk:"manage_events"`
	ManageThreads           types.String `tfsdk:"manage_threads"`
	CreatePublicThreads     types.String `tfsdk:"create_public_threads"`
	CreatePrivateThreads    types.String `tfsdk:"create_private_threads"`
	UseExternalStickers     types.String `tfsdk:"use_external_stickers"`
	SendMessagesInThreads   types.String `tfsdk:"send_messages_in_threads"`
	UseEmbeddedActivities   types.String `tfsdk:"use_embedded_activities"`
	ModerateMembers         types.String `tfsdk:"moderate_members"`
	UseSoundboard           types.String `tfsdk:"use_soundboard"`
	UseExternalSounds       types.String `tfsdk:"use_external_sounds"`
	SendVoiceMessages       types.String `tfsdk:"send_voice_messages"`
	SendPolls               types.String `tfsdk:"send_polls"`
	UseApplicationCommands  types.String `tfsdk:"use_application_commands"`
	SendThreadMessages      types.String `tfsdk:"send_thread_messages"`
	StartEmbeddedActivities types.String `tfsdk:"start_embedded_activities"`
	SetVoiceChannelStatus   types.String `tfsdk:"set_voice_channel_status"`
	AllowBits               types.Int64  `tfsdk:"allow_bits"`
	DenyBits                types.Int64  `tfsdk:"deny_bits"`
}

func NewPermissionDataSource() datasource.DataSource {
	return &PermissionDataSource{}
}

func (d *PermissionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

func (d *PermissionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	permAttr := schema.StringAttribute{
		Optional:    true,
		Description: "Set to `allow` or `deny` to include this permission in the respective bits output.",
	}

	resp.Schema = schema.Schema{
		Description: "Computes permission bit values from named permission flags. Set each permission to `allow` or `deny`.",
		Attributes: map[string]schema.Attribute{
			"create_instant_invite":     permAttr,
			"kick_members":              permAttr,
			"ban_members":               permAttr,
			"administrator":             permAttr,
			"manage_channels":           permAttr,
			"manage_guild":              permAttr,
			"add_reactions":             permAttr,
			"view_audit_log":            permAttr,
			"priority_speaker":          permAttr,
			"stream":                    permAttr,
			"view_channel":              permAttr,
			"send_messages":             permAttr,
			"send_tts_messages":         permAttr,
			"manage_messages":           permAttr,
			"embed_links":               permAttr,
			"attach_files":              permAttr,
			"read_message_history":      permAttr,
			"mention_everyone":          permAttr,
			"use_external_emojis":       permAttr,
			"view_guild_insights":       permAttr,
			"connect":                   permAttr,
			"speak":                     permAttr,
			"mute_members":              permAttr,
			"deafen_members":            permAttr,
			"move_members":              permAttr,
			"use_vad":                   permAttr,
			"change_nickname":           permAttr,
			"manage_nicknames":          permAttr,
			"manage_roles":              permAttr,
			"manage_webhooks":           permAttr,
			"manage_expressions":        permAttr,
			"use_app_commands":          permAttr,
			"request_to_speak":          permAttr,
			"manage_events":             permAttr,
			"manage_threads":            permAttr,
			"create_public_threads":     permAttr,
			"create_private_threads":    permAttr,
			"use_external_stickers":     permAttr,
			"send_messages_in_threads":  permAttr,
			"use_embedded_activities":   permAttr,
			"moderate_members":          permAttr,
			"use_soundboard":            permAttr,
			"use_external_sounds":       permAttr,
			"send_voice_messages":       permAttr,
			"send_polls":                permAttr,
			"use_application_commands":  permAttr,
			"send_thread_messages":      permAttr,
			"start_embedded_activities": permAttr,
			"set_voice_channel_status":  permAttr,
			"allow_bits": schema.Int64Attribute{
				Computed:    true,
				Description: "The computed bitwise value of all permissions set to `allow`.",
			},
			"deny_bits": schema.Int64Attribute{
				Computed:    true,
				Description: "The computed bitwise value of all permissions set to `deny`.",
			},
		},
	}
}

func (d *PermissionDataSource) Configure(_ context.Context, _ datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
}

func (d *PermissionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PermissionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var allow, deny int64

	// Map each permission field to its bit flag.
	perms := []struct {
		value types.String
		bit   int64
	}{
		{data.CreateInstantInvite, permCreateInstantInvite},
		{data.KickMembers, permKickMembers},
		{data.BanMembers, permBanMembers},
		{data.Administrator, permAdministrator},
		{data.ManageChannels, permManageChannels},
		{data.ManageGuild, permManageGuild},
		{data.AddReactions, permAddReactions},
		{data.ViewAuditLog, permViewAuditLog},
		{data.PrioritySpeaker, permPrioritySpeaker},
		{data.Stream, permStream},
		{data.ViewChannel, permViewChannel},
		{data.SendMessages, permSendMessages},
		{data.SendTTSMessages, permSendTTSMessages},
		{data.ManageMessages, permManageMessages},
		{data.EmbedLinks, permEmbedLinks},
		{data.AttachFiles, permAttachFiles},
		{data.ReadMessageHistory, permReadMessageHistory},
		{data.MentionEveryone, permMentionEveryone},
		{data.UseExternalEmojis, permUseExternalEmojis},
		{data.ViewGuildInsights, permViewGuildInsights},
		{data.Connect, permConnect},
		{data.Speak, permSpeak},
		{data.MuteMembers, permMuteMembers},
		{data.DeafenMembers, permDeafenMembers},
		{data.MoveMembers, permMoveMembers},
		{data.UseVAD, permUseVAD},
		{data.ChangeNickname, permChangeNickname},
		{data.ManageNicknames, permManageNicknames},
		{data.ManageRoles, permManageRoles},
		{data.ManageWebhooks, permManageWebhooks},
		{data.ManageExpressions, permManageExpressions},
		{data.UseAppCommands, permUseAppCommands},
		{data.RequestToSpeak, permRequestToSpeak},
		{data.ManageEvents, permManageEvents},
		{data.ManageThreads, permManageThreads},
		{data.CreatePublicThreads, permCreatePublicThreads},
		{data.CreatePrivateThreads, permCreatePrivateThreads},
		{data.UseExternalStickers, permUseExternalStickers},
		{data.SendMessagesInThreads, permSendMessagesInThreads},
		{data.UseEmbeddedActivities, permUseEmbeddedActivities},
		{data.ModerateMembers, permModerateMembers},
		{data.UseSoundboard, permUseSoundboard},
		{data.UseExternalSounds, permUseExternalSounds},
		{data.SendVoiceMessages, permSendVoiceMessages},
		{data.SendPolls, permSendPolls},
		{data.UseApplicationCommands, permUseAppCommands},
		{data.SendThreadMessages, permSendMessagesInThreads},
		{data.StartEmbeddedActivities, permUseEmbeddedActivities},
		{data.SetVoiceChannelStatus, permSetVoiceChannelStatus},
	}

	for _, p := range perms {
		if p.value.IsNull() || p.value.IsUnknown() {
			continue
		}
		switch p.value.ValueString() {
		case "allow":
			allow |= p.bit
		case "deny":
			deny |= p.bit
		}
	}

	data.AllowBits = types.Int64Value(allow)
	data.DenyBits = types.Int64Value(deny)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
