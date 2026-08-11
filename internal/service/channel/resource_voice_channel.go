package channel

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var (
	_ resource.Resource                = (*VoiceChannelResource)(nil)
	_ resource.ResourceWithImportState = (*VoiceChannelResource)(nil)
)

type VoiceChannelResource struct {
	client *client.DiscordClient
}

type VoiceChannelResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	ServerID              types.String `tfsdk:"server_id"`
	Name                  types.String `tfsdk:"name"`
	Category              types.String `tfsdk:"category"`
	Position              types.Int64  `tfsdk:"position"`
	Bitrate               types.Int64  `tfsdk:"bitrate"`
	UserLimit             types.Int64  `tfsdk:"user_limit"`
	SyncPermsWithCategory types.Bool   `tfsdk:"sync_perms_with_category"`
}

func NewVoiceChannelResource() resource.Resource {
	return &VoiceChannelResource{}
}

func (r *VoiceChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_voice_channel"
}

func (r *VoiceChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord voice channel.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the channel.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_id": schema.StringAttribute{
				Required:    true,
				Description: "The server (guild) ID where the channel will be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the voice channel (1-100 characters).",
			},
			"category": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the parent category channel.",
			},
			"position": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The position of the channel in the channel list.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"bitrate": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The bitrate of the voice channel (8000-96000 for normal, up to 384000 for boosted servers).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"user_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "The user limit of the voice channel (0-99). 0 means unlimited.",
			},
			"sync_perms_with_category": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to sync permissions with the parent category.",
			},
		},
	}
}

func (r *VoiceChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VoiceChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VoiceChannelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := discordgo.GuildChannelCreateData{
		Name: plan.Name.ValueString(),
		Type: discordgo.ChannelTypeGuildVoice,
	}

	if !plan.Category.IsNull() && !plan.Category.IsUnknown() {
		params.ParentID = plan.Category.ValueString()
	}
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		params.Position = int(plan.Position.ValueInt64())
	}
	if !plan.Bitrate.IsNull() && !plan.Bitrate.IsUnknown() {
		params.Bitrate = int(plan.Bitrate.ValueInt64())
	}
	if !plan.UserLimit.IsNull() && !plan.UserLimit.IsUnknown() {
		params.UserLimit = int(plan.UserLimit.ValueInt64())
	}

	ch, err := r.client.Session.GuildChannelCreateComplex(plan.ServerID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Voice Channel", err.Error())
		return
	}

	tflog.Debug(ctx, "created voice channel", map[string]interface{}{
		"id":   ch.ID,
		"name": ch.Name,
	})

	plan.ID = types.StringValue(ch.ID)
	plan.Position = types.Int64Value(int64(ch.Position))
	plan.Bitrate = types.Int64Value(int64(ch.Bitrate))

	if plan.SyncPermsWithCategory.ValueBool() && params.ParentID != "" {
		if err := syncChannelPermissions(r.client.Session, ch.ID, params.ParentID); err != nil {
			resp.Diagnostics.AddWarning(
				"Failed to Sync Permissions with Category",
				"Channel was created but permission sync failed: "+err.Error(),
			)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VoiceChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VoiceChannelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ch, err := r.client.Session.Channel(state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Voice Channel", err.Error())
		return
	}

	state.Name = types.StringValue(ch.Name)
	state.Position = types.Int64Value(int64(ch.Position))
	state.Bitrate = types.Int64Value(int64(ch.Bitrate))
	state.ServerID = types.StringValue(ch.GuildID)

	if ch.ParentID != "" {
		state.Category = types.StringValue(ch.ParentID)
	} else {
		state.Category = types.StringNull()
	}

	if ch.UserLimit > 0 {
		state.UserLimit = types.Int64Value(int64(ch.UserLimit))
	} else {
		state.UserLimit = types.Int64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VoiceChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VoiceChannelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	edit := &discordgo.ChannelEdit{
		Name: plan.Name.ValueString(),
	}

	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		pos := int(plan.Position.ValueInt64())
		edit.Position = &pos
	}
	if !plan.Category.IsNull() && !plan.Category.IsUnknown() {
		edit.ParentID = plan.Category.ValueString()
	}
	if !plan.Bitrate.IsNull() && !plan.Bitrate.IsUnknown() {
		bitrate := int(plan.Bitrate.ValueInt64())
		edit.Bitrate = bitrate
	}
	if !plan.UserLimit.IsNull() && !plan.UserLimit.IsUnknown() {
		ul := int(plan.UserLimit.ValueInt64())
		edit.UserLimit = ul
	}

	ch, err := r.client.Session.ChannelEdit(plan.ID.ValueString(), edit)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Voice Channel", err.Error())
		return
	}

	plan.Position = types.Int64Value(int64(ch.Position))
	plan.Bitrate = types.Int64Value(int64(ch.Bitrate))

	if plan.SyncPermsWithCategory.ValueBool() && !plan.Category.IsNull() {
		if err := syncChannelPermissions(r.client.Session, ch.ID, plan.Category.ValueString()); err != nil {
			resp.Diagnostics.AddWarning(
				"Failed to Sync Permissions with Category",
				"Channel was updated but permission sync failed: "+err.Error(),
			)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VoiceChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VoiceChannelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Session.ChannelDelete(state.ID.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Voice Channel", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted voice channel", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

func (r *VoiceChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ch, err := r.client.Session.Channel(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Voice Channel", err.Error())
		return
	}

	state := VoiceChannelResourceModel{
		ID:                    types.StringValue(ch.ID),
		ServerID:              types.StringValue(ch.GuildID),
		Name:                  types.StringValue(ch.Name),
		Position:              types.Int64Value(int64(ch.Position)),
		Bitrate:               types.Int64Value(int64(ch.Bitrate)),
		SyncPermsWithCategory: types.BoolValue(false),
	}

	if ch.ParentID != "" {
		state.Category = types.StringValue(ch.ParentID)
	} else {
		state.Category = types.StringNull()
	}

	if ch.UserLimit > 0 {
		state.UserLimit = types.Int64Value(int64(ch.UserLimit))
	} else {
		state.UserLimit = types.Int64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
