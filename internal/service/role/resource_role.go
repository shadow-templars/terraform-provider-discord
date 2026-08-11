package role

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var (
	_ resource.Resource                = (*RoleResource)(nil)
	_ resource.ResourceWithImportState = (*RoleResource)(nil)
)

type RoleResource struct {
	client *client.DiscordClient
}

type RoleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ServerID    types.String `tfsdk:"server_id"`
	Name        types.String `tfsdk:"name"`
	Color       types.Int64  `tfsdk:"color"`
	Hoist       types.Bool   `tfsdk:"hoist"`
	Mentionable types.Bool   `tfsdk:"mentionable"`
	Permissions types.Int64  `tfsdk:"permissions"`
	Position    types.Int64  `tfsdk:"position"`
}

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

func (r *RoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the role.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_id": schema.StringAttribute{
				Required:    true,
				Description: "The server (guild) ID where the role will be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the role.",
			},
			"color": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The color of the role as a decimal integer.",
			},
			"hoist": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the role should be displayed separately in the member list.",
			},
			"mentionable": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the role can be mentioned by anyone.",
			},
			"permissions": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The bitwise value of the role's permissions.",
			},
			"position": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The position of the role in the role hierarchy.",
			},
		},
	}
}

func (r *RoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &discordgo.RoleParams{
		Name:        plan.Name.ValueString(),
		Color:       intPtr(int(plan.Color.ValueInt64())),
		Hoist:       boolPtr(plan.Hoist.ValueBool()),
		Mentionable: boolPtr(plan.Mentionable.ValueBool()),
	}

	perms := plan.Permissions.ValueInt64()
	params.Permissions = &perms

	role, err := r.client.Session.GuildRoleCreate(plan.ServerID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Role", err.Error())
		return
	}

	tflog.Debug(ctx, "created role", map[string]interface{}{
		"id":   role.ID,
		"name": role.Name,
	})

	plan.ID = types.StringValue(role.ID)
	plan.Position = types.Int64Value(int64(role.Position))

	// Set position if specified.
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		desiredPos := int(plan.Position.ValueInt64())
		if role.Position != desiredPos {
			roles := []*discordgo.Role{{ID: role.ID, Position: desiredPos}}
			_, err := r.client.Session.GuildRoleReorder(plan.ServerID.ValueString(), roles)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Failed to Set Role Position",
					"Role was created but position could not be set: "+err.Error(),
				)
			} else {
				plan.Position = types.Int64Value(int64(desiredPos))
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := r.client.Session.GuildRoles(state.ServerID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Roles", err.Error())
		return
	}

	var found *discordgo.Role
	for _, role := range roles {
		if role.ID == state.ID.ValueString() {
			found = role
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(found.Name)
	state.Color = types.Int64Value(int64(found.Color))
	state.Hoist = types.BoolValue(found.Hoist)
	state.Mentionable = types.BoolValue(found.Mentionable)
	state.Permissions = types.Int64Value(found.Permissions)
	state.Position = types.Int64Value(int64(found.Position))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &discordgo.RoleParams{
		Name:        plan.Name.ValueString(),
		Color:       intPtr(int(plan.Color.ValueInt64())),
		Hoist:       boolPtr(plan.Hoist.ValueBool()),
		Mentionable: boolPtr(plan.Mentionable.ValueBool()),
	}

	perms := plan.Permissions.ValueInt64()
	params.Permissions = &perms

	role, err := r.client.Session.GuildRoleEdit(plan.ServerID.ValueString(), plan.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Role", err.Error())
		return
	}

	plan.Position = types.Int64Value(int64(role.Position))

	// Update position if specified and different.
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		desiredPos := int(plan.Position.ValueInt64())
		if role.Position != desiredPos {
			roles := []*discordgo.Role{{ID: role.ID, Position: desiredPos}}
			_, err := r.client.Session.GuildRoleReorder(plan.ServerID.ValueString(), roles)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Failed to Update Role Position",
					"Role was updated but position could not be changed: "+err.Error(),
				)
			} else {
				plan.Position = types.Int64Value(int64(desiredPos))
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Session.GuildRoleDelete(state.ServerID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Role", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted role", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import requires server_id:role_id format since roles are server-scoped.
	parts := splitImportID(req.ID, 2)
	if parts == nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in the format `server_id:role_id`.",
		)
		return
	}

	serverID := parts[0]
	roleID := parts[1]

	roles, err := r.client.Session.GuildRoles(serverID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Role", err.Error())
		return
	}

	var found *discordgo.Role
	for _, role := range roles {
		if role.ID == roleID {
			found = role
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError(
			"Role Not Found",
			fmt.Sprintf("No role with ID %s found in server %s.", roleID, serverID),
		)
		return
	}

	state := RoleResourceModel{
		ID:          types.StringValue(found.ID),
		ServerID:    types.StringValue(serverID),
		Name:        types.StringValue(found.Name),
		Color:       types.Int64Value(int64(found.Color)),
		Hoist:       types.BoolValue(found.Hoist),
		Mentionable: types.BoolValue(found.Mentionable),
		Permissions: types.Int64Value(found.Permissions),
		Position:    types.Int64Value(int64(found.Position)),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
