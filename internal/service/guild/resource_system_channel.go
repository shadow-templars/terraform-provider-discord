package guild

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var _ resource.Resource = (*SystemChannelResource)(nil)

type SystemChannelResource struct {
	client *client.DiscordClient
}

type SystemChannelResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ServerID           types.String `tfsdk:"server_id"`
	SystemChannelID    types.String `tfsdk:"system_channel_id"`
	SystemChannelFlags types.Int64  `tfsdk:"system_channel_flags"`
}

func NewSystemChannelResource() resource.Resource {
	return &SystemChannelResource{}
}

func (r *SystemChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_channel"
}

func (r *SystemChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the system channel settings for a Discord server.",
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
			"system_channel_id": schema.StringAttribute{
				Required:    true,
				Description: "The channel ID for system messages (welcome, boost, etc.).",
			},
			"system_channel_flags": schema.Int64Attribute{
				Optional:    true,
				Description: "Bitfield of system channel flags (suppress join notifications = 1, boost = 2, tips = 4, sticker reply = 8).",
			},
		},
	}
}

func (r *SystemChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SystemChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SystemChannelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = plan.ServerID
	r.apply(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SystemChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SystemChannelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	guild, err := r.client.Session.Guild(state.ServerID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Server", err.Error())
		return
	}

	if guild.SystemChannelID != "" {
		state.SystemChannelID = types.StringValue(guild.SystemChannelID)
	}
	state.SystemChannelFlags = types.Int64Value(int64(guild.SystemChannelFlags))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SystemChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SystemChannelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SystemChannelResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Removing from state only — server always has system channel settings.
}

func (r *SystemChannelResource) apply(_ context.Context, plan *SystemChannelResourceModel, diags *diag.Diagnostics) {
	params := discordgo.GuildParams{
		SystemChannelID: plan.SystemChannelID.ValueString(),
	}
	if !plan.SystemChannelFlags.IsNull() {
		params.SystemChannelFlags = discordgo.SystemChannelFlag(plan.SystemChannelFlags.ValueInt64())
	}

	_, err := r.client.Session.GuildEdit(plan.ServerID.ValueString(), &params)
	if err != nil {
		diags.AddError("Error Updating System Channel", err.Error())
	}
}
