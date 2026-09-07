package role

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var (
	_ resource.Resource                = (*MemberRolesResource)(nil)
	_ resource.ResourceWithImportState = (*MemberRolesResource)(nil)
)

type MemberRolesResource struct {
	client *client.DiscordClient
}

type MemberRolesResourceModel struct {
	ID       types.String `tfsdk:"id"`
	ServerID types.String `tfsdk:"server_id"`
	UserID   types.String `tfsdk:"user_id"`
	Roles    types.Set    `tfsdk:"role"`
}

type MemberRoleModel struct {
	RoleID types.String `tfsdk:"role_id"`
}

var memberRoleAttrTypes = map[string]attr.Type{
	"role_id": types.StringType,
}

func NewMemberRolesResource() resource.Resource {
	return &MemberRolesResource{}
}

func (r *MemberRolesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_member_roles"
}

func (r *MemberRolesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages role assignments for a Discord server member.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite ID in the format `server_id:user_id`.",
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
			"user_id": schema.StringAttribute{
				Required:    true,
				Description: "The user ID of the member.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"role": schema.SetNestedBlock{
				Description: "Role to assign to the member.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"role_id": schema.StringAttribute{
							Required:    true,
							Description: "The ID of the role to assign.",
						},
					},
				},
			},
		},
	}
}

func (r *MemberRolesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MemberRolesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MemberRolesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(plan.ServerID.ValueString() + ":" + plan.UserID.ValueString())

	roleIDs := r.extractRoleIDs(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, roleID := range roleIDs {
		err := r.client.Session.GuildMemberRoleAdd(plan.ServerID.ValueString(), plan.UserID.ValueString(), roleID)
		if err != nil {
			resp.Diagnostics.AddError("Error Adding Role to Member", fmt.Sprintf("Failed to add role %s: %s", roleID, err.Error()))
			return
		}
	}

	tflog.Debug(ctx, "assigned member roles", map[string]interface{}{
		"user_id": plan.UserID.ValueString(),
		"roles":   roleIDs,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MemberRolesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MemberRolesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.client.Session.GuildMember(state.ServerID.ValueString(), state.UserID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Member", err.Error())
		return
	}

	// Only track roles that are in our config, ignore externally added roles.
	configuredRoleIDs := r.extractRoleIDs(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	memberRoleSet := make(map[string]bool, len(member.Roles))
	for _, roleID := range member.Roles {
		memberRoleSet[roleID] = true
	}

	var remaining []MemberRoleModel
	for _, roleID := range configuredRoleIDs {
		if memberRoleSet[roleID] {
			remaining = append(remaining, MemberRoleModel{RoleID: types.StringValue(roleID)})
		}
	}

	if len(remaining) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	roleSet, diags := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: memberRoleAttrTypes}, remaining)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Roles = roleSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MemberRolesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state MemberRolesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldRoles := r.extractRoleIDs(ctx, &state, &resp.Diagnostics)
	newRoles := r.extractRoleIDs(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	oldSet := make(map[string]bool, len(oldRoles))
	for _, id := range oldRoles {
		oldSet[id] = true
	}
	newSet := make(map[string]bool, len(newRoles))
	for _, id := range newRoles {
		newSet[id] = true
	}

	for _, id := range newRoles {
		if !oldSet[id] {
			err := r.client.Session.GuildMemberRoleAdd(plan.ServerID.ValueString(), plan.UserID.ValueString(), id)
			if err != nil {
				resp.Diagnostics.AddError("Error Adding Role", fmt.Sprintf("Failed to add role %s: %s", id, err.Error()))
				return
			}
		}
	}

	for _, id := range oldRoles {
		if !newSet[id] {
			err := r.client.Session.GuildMemberRoleRemove(plan.ServerID.ValueString(), plan.UserID.ValueString(), id)
			if err != nil {
				resp.Diagnostics.AddError("Error Removing Role", fmt.Sprintf("Failed to remove role %s: %s", id, err.Error()))
				return
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MemberRolesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MemberRolesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleIDs := r.extractRoleIDs(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, roleID := range roleIDs {
		err := r.client.Session.GuildMemberRoleRemove(state.ServerID.ValueString(), state.UserID.ValueString(), roleID)
		if err != nil && !isNotFound(err) {
			resp.Diagnostics.AddError("Error Removing Role", fmt.Sprintf("Failed to remove role %s: %s", roleID, err.Error()))
			return
		}
	}

	tflog.Debug(ctx, "removed member roles", map[string]interface{}{
		"user_id": state.UserID.ValueString(),
	})
}

func (r *MemberRolesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID must be in the format `server_id:user_id`.")
		return
	}

	serverID := parts[0]
	userID := parts[1]

	member, err := r.client.Session.GuildMember(serverID, userID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Member Roles", err.Error())
		return
	}

	var roles []MemberRoleModel
	for _, roleID := range member.Roles {
		roles = append(roles, MemberRoleModel{RoleID: types.StringValue(roleID)})
	}

	roleSet, diags := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: memberRoleAttrTypes}, roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := MemberRolesResourceModel{
		ID:       types.StringValue(serverID + ":" + userID),
		ServerID: types.StringValue(serverID),
		UserID:   types.StringValue(userID),
		Roles:    roleSet,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MemberRolesResource) extractRoleIDs(ctx context.Context, model *MemberRolesResourceModel, diags *diag.Diagnostics) []string {
	if model.Roles.IsNull() || model.Roles.IsUnknown() {
		return nil
	}

	var roles []MemberRoleModel
	diags.Append(model.Roles.ElementsAs(ctx, &roles, false)...)
	if diags.HasError() {
		return nil
	}

	ids := make([]string, len(roles))
	for i, r := range roles {
		ids[i] = r.RoleID.ValueString()
	}
	return ids
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	restErr, ok := err.(*discordgo.RESTError)
	if ok && restErr.Response != nil && restErr.Response.StatusCode == 404 {
		return true
	}
	return strings.Contains(err.Error(), "404")
}
