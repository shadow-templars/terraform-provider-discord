package channel

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var (
	_ resource.Resource                = (*CategoryChannelResource)(nil)
	_ resource.ResourceWithImportState = (*CategoryChannelResource)(nil)
)

type CategoryChannelResource struct {
	client *client.DiscordClient
}

type CategoryChannelResourceModel struct {
	ID       types.String `tfsdk:"id"`
	ServerID types.String `tfsdk:"server_id"`
	Name     types.String `tfsdk:"name"`
	Position types.Int64  `tfsdk:"position"`
}

func NewCategoryChannelResource() resource.Resource {
	return &CategoryChannelResource{}
}

func (r *CategoryChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_category_channel"
}

func (r *CategoryChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord category channel.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the category.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_id": schema.StringAttribute{
				Required:    true,
				Description: "The server (guild) ID where the category will be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the category channel (1-100 characters).",
			},
			"position": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The position of the category in the channel list.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *CategoryChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CategoryChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CategoryChannelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := discordgo.GuildChannelCreateData{
		Name: plan.Name.ValueString(),
		Type: discordgo.ChannelTypeGuildCategory,
	}

	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		params.Position = int(plan.Position.ValueInt64())
	}

	ch, err := r.client.Session.GuildChannelCreateComplex(plan.ServerID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Category Channel", err.Error())
		return
	}

	tflog.Debug(ctx, "created category channel", map[string]interface{}{
		"id":   ch.ID,
		"name": ch.Name,
	})

	plan.ID = types.StringValue(ch.ID)
	plan.Position = types.Int64Value(int64(ch.Position))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CategoryChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CategoryChannelResourceModel
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
		resp.Diagnostics.AddError("Error Reading Category Channel", err.Error())
		return
	}

	state.Name = types.StringValue(ch.Name)
	state.Position = types.Int64Value(int64(ch.Position))
	state.ServerID = types.StringValue(ch.GuildID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CategoryChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CategoryChannelResourceModel
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

	ch, err := r.client.Session.ChannelEdit(plan.ID.ValueString(), edit)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Category Channel", err.Error())
		return
	}

	plan.Position = types.Int64Value(int64(ch.Position))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CategoryChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CategoryChannelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Session.ChannelDelete(state.ID.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Category Channel", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted category channel", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

func (r *CategoryChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ch, err := r.client.Session.Channel(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Category Channel", err.Error())
		return
	}

	state := CategoryChannelResourceModel{
		ID:       types.StringValue(ch.ID),
		ServerID: types.StringValue(ch.GuildID),
		Name:     types.StringValue(ch.Name),
		Position: types.Int64Value(int64(ch.Position)),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
