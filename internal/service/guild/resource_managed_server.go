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

var _ resource.Resource = (*ManagedServerResource)(nil)

type ManagedServerResource struct {
	client *client.DiscordClient
}

type ManagedServerResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	ServerID                    types.String `tfsdk:"server_id"`
	Name                        types.String `tfsdk:"name"`
	Description                 types.String `tfsdk:"description"`
	DefaultMessageNotifications types.Int64  `tfsdk:"default_message_notifications"`
	ExplicitContentFilter       types.Int64  `tfsdk:"explicit_content_filter"`
	VerificationLevel           types.Int64  `tfsdk:"verification_level"`
	IconDataURI                 types.String `tfsdk:"icon_data_uri"`
	SplashDataURI               types.String `tfsdk:"splash_data_uri"`
}

func NewManagedServerResource() resource.Resource {
	return &ManagedServerResource{}
}

func (r *ManagedServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_server"
}

func (r *ManagedServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages settings of an existing Discord server (guild). Does not create or delete the server.",
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
				Description: "The ID of the existing server to manage.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The server name (2-100 characters).",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The server description (for Community servers).",
			},
			"default_message_notifications": schema.Int64Attribute{
				Optional:    true,
				Description: "Default notification level: 0 = all messages, 1 = only mentions.",
			},
			"explicit_content_filter": schema.Int64Attribute{
				Optional:    true,
				Description: "Explicit content filter level: 0 = disabled, 1 = members without roles, 2 = all members.",
			},
			"verification_level": schema.Int64Attribute{
				Optional:    true,
				Description: "Verification level: 0 = none, 1 = low, 2 = medium, 3 = high, 4 = very high.",
			},
			"icon_data_uri": schema.StringAttribute{
				Optional:    true,
				Description: "Server icon as a data URI (data:image/...;base64,...).",
			},
			"splash_data_uri": schema.StringAttribute{
				Optional:    true,
				Description: "Invite splash image as a data URI (requires INVITE_SPLASH feature).",
			},
		},
	}
}

func (r *ManagedServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ManagedServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ManagedServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Managed server doesn't create — it just applies settings to an existing server.
	plan.ID = plan.ServerID

	r.applySettings(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ManagedServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ManagedServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	guild, err := r.client.Session.Guild(state.ServerID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Server", err.Error())
		return
	}

	state.Name = types.StringValue(guild.Name)
	if guild.Description != "" {
		state.Description = types.StringValue(guild.Description)
	}
	state.DefaultMessageNotifications = types.Int64Value(int64(guild.DefaultMessageNotifications))
	state.ExplicitContentFilter = types.Int64Value(int64(guild.ExplicitContentFilter))
	state.VerificationLevel = types.Int64Value(int64(guild.VerificationLevel))
	// icon_data_uri and splash_data_uri are write-only — can't read them back.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ManagedServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ManagedServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applySettings(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ManagedServerResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Managed servers are never deleted — removing from state only.
}

func (r *ManagedServerResource) applySettings(_ context.Context, plan *ManagedServerResourceModel, diags *diag.Diagnostics) {
	params := discordgo.GuildParams{
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = plan.Description.ValueString()
	}
	if !plan.DefaultMessageNotifications.IsNull() {
		params.DefaultMessageNotifications = int(plan.DefaultMessageNotifications.ValueInt64())
	}
	if !plan.ExplicitContentFilter.IsNull() {
		params.ExplicitContentFilter = int(plan.ExplicitContentFilter.ValueInt64())
	}
	if !plan.VerificationLevel.IsNull() {
		vl := discordgo.VerificationLevel(plan.VerificationLevel.ValueInt64())
		params.VerificationLevel = &vl
	}
	if !plan.IconDataURI.IsNull() && !plan.IconDataURI.IsUnknown() {
		params.Icon = plan.IconDataURI.ValueString()
	}
	if !plan.SplashDataURI.IsNull() && !plan.SplashDataURI.IsUnknown() {
		params.Splash = plan.SplashDataURI.ValueString()
	}

	_, err := r.client.Session.GuildEdit(plan.ServerID.ValueString(), &params)
	if err != nil {
		diags.AddError("Error Updating Server Settings", err.Error())
	}
}


