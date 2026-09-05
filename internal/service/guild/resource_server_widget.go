package guild

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
	"github.com/shadow-templars/terraform-provider-discord/internal/discordgox"
)

var _ resource.Resource = (*ServerWidgetResource)(nil)

type ServerWidgetResource struct {
	client *client.DiscordClient
}

type ServerWidgetResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ServerID  types.String `tfsdk:"server_id"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	ChannelID types.String `tfsdk:"channel_id"`
}

func NewServerWidgetResource() resource.Resource {
	return &ServerWidgetResource{}
}

func (r *ServerWidgetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_widget"
}

func (r *ServerWidgetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the widget settings for a Discord server.",
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
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether the widget is enabled.",
			},
			"channel_id": schema.StringAttribute{
				Optional:    true,
				Description: "The channel ID for the widget invite.",
			},
		},
	}
}

func (r *ServerWidgetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServerWidgetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServerWidgetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = plan.ServerID
	if err := r.apply(&plan); err != nil {
		resp.Diagnostics.AddError("Error Updating Server Widget", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ServerWidgetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServerWidgetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	widget, err := r.client.Session.GuildWidget(state.ServerID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Server Widget", err.Error())
		return
	}

	state.Enabled = types.BoolValue(widget.Enabled)
	if widget.ChannelID != "" {
		state.ChannelID = types.StringValue(widget.ChannelID)
	} else {
		state.ChannelID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ServerWidgetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServerWidgetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(&plan); err != nil {
		resp.Diagnostics.AddError("Error Updating Server Widget", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ServerWidgetResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Removing from state only.
}

func (r *ServerWidgetResource) apply(plan *ServerWidgetResourceModel) error {
	settings := discordgox.GuildWidgetSettings{
		Enabled: plan.Enabled.ValueBool(),
	}
	if !plan.ChannelID.IsNull() && !plan.ChannelID.IsUnknown() {
		settings.ChannelID = plan.ChannelID.ValueString()
	}

	return r.client.Session.GuildWidgetEdit(plan.ServerID.ValueString(), settings)
}
