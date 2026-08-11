package role

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var _ resource.Resource = (*RoleEveryoneResource)(nil)

type RoleEveryoneResource struct {
	client *client.DiscordClient
}

type RoleEveryoneResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ServerID    types.String `tfsdk:"server_id"`
	Permissions types.Int64  `tfsdk:"permissions"`
}

func NewRoleEveryoneResource() resource.Resource {
	return &RoleEveryoneResource{}
}

func (r *RoleEveryoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_everyone"
}

func (r *RoleEveryoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the @everyone role permissions for a Discord server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the @everyone role (same as server_id).",
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
			"permissions": schema.Int64Attribute{
				Required:    true,
				Description: "The bitwise value of the @everyone role's permissions.",
			},
		},
	}
}

func (r *RoleEveryoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleEveryoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleEveryoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The @everyone role ID is always equal to the server ID.
	plan.ID = plan.ServerID

	if err := r.apply(&plan); err != nil {
		resp.Diagnostics.AddError("Error Updating @everyone Role", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleEveryoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleEveryoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := r.client.Session.GuildRoles(state.ServerID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Roles", err.Error())
		return
	}

	for _, role := range roles {
		if role.ID == state.ServerID.ValueString() {
			state.Permissions = types.Int64Value(role.Permissions)
			break
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RoleEveryoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RoleEveryoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(&plan); err != nil {
		resp.Diagnostics.AddError("Error Updating @everyone Role", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleEveryoneResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Can't delete @everyone — removing from state only.
}

func (r *RoleEveryoneResource) apply(plan *RoleEveryoneResourceModel) error {
	perms := plan.Permissions.ValueInt64()
	params := &discordgo.RoleParams{
		Permissions: &perms,
	}

	_, err := r.client.Session.GuildRoleEdit(plan.ServerID.ValueString(), plan.ServerID.ValueString(), params)
	return err
}
