package channel

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var (
	_ resource.Resource                = (*InviteResource)(nil)
	_ resource.ResourceWithImportState = (*InviteResource)(nil)
)

type InviteResource struct {
	client *client.DiscordClient
}

type InviteResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ChannelID types.String `tfsdk:"channel_id"`
	MaxAge    types.Int64  `tfsdk:"max_age"`
	MaxUses   types.Int64  `tfsdk:"max_uses"`
	Temporary types.Bool   `tfsdk:"temporary"`
	Unique    types.Bool   `tfsdk:"unique"`
	Code      types.String `tfsdk:"code"`
}

func NewInviteResource() resource.Resource {
	return &InviteResource{}
}

func (r *InviteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_invite"
}

func (r *InviteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord channel invite.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The invite code (same as `code`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"channel_id": schema.StringAttribute{
				Required:    true,
				Description: "The channel ID to create the invite for.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"max_age": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Duration of invite in seconds before expiry. 0 means never.",
			},
			"max_uses": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Max number of uses. 0 means unlimited.",
			},
			"temporary": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether this invite only grants temporary membership.",
			},
			"unique": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to guarantee a unique invite code.",
			},
			"code": schema.StringAttribute{
				Computed:    true,
				Description: "The invite code.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *InviteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *InviteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InviteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	invite := discordgo.Invite{
		MaxAge:    int(plan.MaxAge.ValueInt64()),
		MaxUses:   int(plan.MaxUses.ValueInt64()),
		Temporary: plan.Temporary.ValueBool(),
		Unique:    plan.Unique.ValueBool(),
	}

	created, err := r.client.Session.ChannelInviteCreate(plan.ChannelID.ValueString(), invite)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Invite", err.Error())
		return
	}

	tflog.Debug(ctx, "created invite", map[string]interface{}{
		"code": created.Code,
	})

	plan.ID = types.StringValue(created.Code)
	plan.Code = types.StringValue(created.Code)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InviteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InviteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	invite, err := r.client.Session.Invite(state.Code.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Invite", err.Error())
		return
	}

	state.ChannelID = types.StringValue(invite.Channel.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *InviteResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Invites are immutable — changes require recreation.
	resp.Diagnostics.AddError("Invites Cannot Be Updated", "Discord invites are immutable. Delete and recreate instead.")
}

func (r *InviteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InviteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Session.InviteDelete(state.Code.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Invite", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted invite", map[string]interface{}{
		"code": state.Code.ValueString(),
	})
}

func (r *InviteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	invite, err := r.client.Session.Invite(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Invite", err.Error())
		return
	}

	state := InviteResourceModel{
		ID:        types.StringValue(invite.Code),
		ChannelID: types.StringValue(invite.Channel.ID),
		Code:      types.StringValue(invite.Code),
		MaxAge:    types.Int64Value(int64(invite.MaxAge)),
		MaxUses:   types.Int64Value(int64(invite.MaxUses)),
		Temporary: types.BoolValue(invite.Temporary),
		Unique:    types.BoolValue(false),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
