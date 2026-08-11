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
	_ resource.Resource                = (*ForumChannelResource)(nil)
	_ resource.ResourceWithImportState = (*ForumChannelResource)(nil)
)

type ForumChannelResource struct {
	client *client.DiscordClient
}

type ForumChannelResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	ServerID              types.String `tfsdk:"server_id"`
	Name                  types.String `tfsdk:"name"`
	Topic                 types.String `tfsdk:"topic"`
	Category              types.String `tfsdk:"category"`
	Position              types.Int64  `tfsdk:"position"`
	SyncPermsWithCategory types.Bool   `tfsdk:"sync_perms_with_category"`
}

func NewForumChannelResource() resource.Resource {
	return &ForumChannelResource{}
}

func (r *ForumChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_forum_channel"
}

func (r *ForumChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord forum channel.",
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
				Description: "The name of the forum channel (1-100 characters).",
			},
			"topic": schema.StringAttribute{
				Optional:    true,
				Description: "The channel topic / guidelines (0-4096 characters).",
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
			"sync_perms_with_category": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to sync permissions with the parent category.",
			},
		},
	}
}

func (r *ForumChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ForumChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ForumChannelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := discordgo.GuildChannelCreateData{
		Name:  plan.Name.ValueString(),
		Type:  discordgo.ChannelTypeGuildForum,
		Topic: plan.Topic.ValueString(),
	}

	if !plan.Category.IsNull() && !plan.Category.IsUnknown() {
		params.ParentID = plan.Category.ValueString()
	}
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		params.Position = int(plan.Position.ValueInt64())
	}

	ch, err := r.client.Session.GuildChannelCreateComplex(plan.ServerID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Forum Channel", err.Error())
		return
	}

	tflog.Debug(ctx, "created forum channel", map[string]interface{}{
		"id":   ch.ID,
		"name": ch.Name,
	})

	plan.ID = types.StringValue(ch.ID)
	plan.Position = types.Int64Value(int64(ch.Position))

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

func (r *ForumChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ForumChannelResourceModel
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
		resp.Diagnostics.AddError("Error Reading Forum Channel", err.Error())
		return
	}

	state.Name = types.StringValue(ch.Name)
	state.Topic = types.StringValue(ch.Topic)
	state.Position = types.Int64Value(int64(ch.Position))
	state.ServerID = types.StringValue(ch.GuildID)

	if ch.ParentID != "" {
		state.Category = types.StringValue(ch.ParentID)
	} else {
		state.Category = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ForumChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ForumChannelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	edit := &discordgo.ChannelEdit{
		Name:  plan.Name.ValueString(),
		Topic: plan.Topic.ValueString(),
	}

	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		pos := int(plan.Position.ValueInt64())
		edit.Position = &pos
	}
	if !plan.Category.IsNull() && !plan.Category.IsUnknown() {
		edit.ParentID = plan.Category.ValueString()
	}

	ch, err := r.client.Session.ChannelEdit(plan.ID.ValueString(), edit)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Forum Channel", err.Error())
		return
	}

	plan.Position = types.Int64Value(int64(ch.Position))

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

func (r *ForumChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ForumChannelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Session.ChannelDelete(state.ID.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Forum Channel", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted forum channel", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

func (r *ForumChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ch, err := r.client.Session.Channel(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Forum Channel", err.Error())
		return
	}

	state := ForumChannelResourceModel{
		ID:                    types.StringValue(ch.ID),
		ServerID:              types.StringValue(ch.GuildID),
		Name:                  types.StringValue(ch.Name),
		Topic:                 types.StringValue(ch.Topic),
		Position:              types.Int64Value(int64(ch.Position)),
		SyncPermsWithCategory: types.BoolValue(false),
	}

	if ch.ParentID != "" {
		state.Category = types.StringValue(ch.ParentID)
	} else {
		state.Category = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
