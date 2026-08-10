package channel

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = (*ChannelPermissionResource)(nil)
	_ resource.ResourceWithImportState = (*ChannelPermissionResource)(nil)
)

type ChannelPermissionResource struct {
	client *client.DiscordClient
}

type ChannelPermissionResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ChannelID   types.String `tfsdk:"channel_id"`
	Type        types.String `tfsdk:"type"`
	OverwriteID types.String `tfsdk:"overwrite_id"`
	Allow       types.Int64  `tfsdk:"allow"`
	Deny        types.Int64  `tfsdk:"deny"`
}

func NewChannelPermissionResource() resource.Resource {
	return &ChannelPermissionResource{}
}

func (r *ChannelPermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel_permission"
}

func (r *ChannelPermissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord channel permission overwrite for a role or member.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite ID in the format `channel_id:overwrite_id`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"channel_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the channel to apply the permission overwrite to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The type of overwrite: `role` or `member`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"overwrite_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the role or member to apply the permission overwrite for.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"allow": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The bitwise value of allowed permissions.",
			},
			"deny": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The bitwise value of denied permissions.",
			},
		},
	}
}

func (r *ChannelPermissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ChannelPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ChannelPermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	overwriteType, diags := parseOverwriteType(plan.Type.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Session.ChannelPermissionSet(
		plan.ChannelID.ValueString(),
		plan.OverwriteID.ValueString(),
		overwriteType,
		plan.Allow.ValueInt64(),
		plan.Deny.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Channel Permission", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.ChannelID.ValueString() + ":" + plan.OverwriteID.ValueString())

	tflog.Debug(ctx, "created channel permission", map[string]interface{}{
		"channel_id":   plan.ChannelID.ValueString(),
		"overwrite_id": plan.OverwriteID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ChannelPermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ch, err := r.client.Session.Channel(state.ChannelID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Channel Permission", err.Error())
		return
	}

	// Find the matching permission overwrite.
	var found *discordgo.PermissionOverwrite
	for _, po := range ch.PermissionOverwrites {
		if po.ID == state.OverwriteID.ValueString() {
			found = po
			break
		}
	}

	if found == nil {
		// Permission overwrite was removed externally.
		resp.State.RemoveResource(ctx)
		return
	}

	state.Allow = types.Int64Value(found.Allow)
	state.Deny = types.Int64Value(found.Deny)

	// Map the type back from discordgo's int to our string.
	switch found.Type {
	case discordgo.PermissionOverwriteTypeRole:
		state.Type = types.StringValue("role")
	case discordgo.PermissionOverwriteTypeMember:
		state.Type = types.StringValue("member")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ChannelPermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	overwriteType, diags := parseOverwriteType(plan.Type.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Session.ChannelPermissionSet(
		plan.ChannelID.ValueString(),
		plan.OverwriteID.ValueString(),
		overwriteType,
		plan.Allow.ValueInt64(),
		plan.Deny.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Channel Permission", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ChannelPermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Session.ChannelPermissionDelete(
		state.ChannelID.ValueString(),
		state.OverwriteID.ValueString(),
	)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Channel Permission", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted channel permission", map[string]interface{}{
		"channel_id":   state.ChannelID.ValueString(),
		"overwrite_id": state.OverwriteID.ValueString(),
	})
}

func (r *ChannelPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: channel_id:overwrite_id
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in the format `channel_id:overwrite_id`.",
		)
		return
	}

	channelID := parts[0]
	overwriteID := parts[1]

	ch, err := r.client.Session.Channel(channelID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Channel Permission", err.Error())
		return
	}

	var found *discordgo.PermissionOverwrite
	for _, po := range ch.PermissionOverwrites {
		if po.ID == overwriteID {
			found = po
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError(
			"Permission Overwrite Not Found",
			fmt.Sprintf("No permission overwrite with ID %s found on channel %s.", overwriteID, channelID),
		)
		return
	}

	state := ChannelPermissionResourceModel{
		ID:          types.StringValue(channelID + ":" + overwriteID),
		ChannelID:   types.StringValue(channelID),
		OverwriteID: types.StringValue(overwriteID),
		Allow:       types.Int64Value(found.Allow),
		Deny:        types.Int64Value(found.Deny),
	}

	switch found.Type {
	case discordgo.PermissionOverwriteTypeRole:
		state.Type = types.StringValue("role")
	case discordgo.PermissionOverwriteTypeMember:
		state.Type = types.StringValue("member")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func parseOverwriteType(t string) (discordgo.PermissionOverwriteType, diag.Diagnostics) {
	var diags diag.Diagnostics
	switch t {
	case "role":
		return discordgo.PermissionOverwriteTypeRole, diags
	case "member":
		return discordgo.PermissionOverwriteTypeMember, diags
	default:
		diags.AddError("Invalid Permission Overwrite Type", fmt.Sprintf("Expected 'role' or 'member', got: %s", t))
		return 0, diags
	}
}
