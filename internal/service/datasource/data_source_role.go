package datasource

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var _ datasource.DataSource = (*RoleDataSource)(nil)

type RoleDataSource struct {
	client *client.DiscordClient
}

type RoleDataSourceModel struct {
	ServerID    types.String `tfsdk:"server_id"`
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
	Color       types.Int64  `tfsdk:"color"`
	Position    types.Int64  `tfsdk:"position"`
	Permissions types.Int64  `tfsdk:"permissions"`
	Hoist       types.Bool   `tfsdk:"hoist"`
	Mentionable types.Bool   `tfsdk:"mentionable"`
	Managed     types.Bool   `tfsdk:"managed"`
}

func NewRoleDataSource() datasource.DataSource {
	return &RoleDataSource{}
}

func (d *RoleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *RoleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a Discord role by name within a server.",
		Attributes: map[string]schema.Attribute{
			"server_id": schema.StringAttribute{
				Required:    true,
				Description: "The server (guild) ID to look up the role in.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the role to look up.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the role.",
			},
			"color": schema.Int64Attribute{
				Computed:    true,
				Description: "The integer representation of the role color.",
			},
			"position": schema.Int64Attribute{
				Computed:    true,
				Description: "The position of the role in the guild hierarchy.",
			},
			"permissions": schema.Int64Attribute{
				Computed:    true,
				Description: "The permission bit set for this role.",
			},
			"hoist": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this role is hoisted (displayed separately in the member list).",
			},
			"mentionable": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this role is mentionable.",
			},
			"managed": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this role is managed by an integration.",
			},
		},
	}
}

func (d *RoleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.DiscordClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.DiscordClient, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RoleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := d.client.Session.GuildRoles(data.ServerID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Guild Roles",
			fmt.Sprintf("Could not fetch roles for server %s: %s", data.ServerID.ValueString(), err.Error()),
		)
		return
	}

	name := data.Name.ValueString()
	for _, role := range roles {
		if role.Name == name {
			data.ID = types.StringValue(role.ID)
			data.Color = types.Int64Value(int64(role.Color))
			data.Position = types.Int64Value(int64(role.Position))
			data.Permissions = types.Int64Value(role.Permissions)
			data.Hoist = types.BoolValue(role.Hoist)
			data.Mentionable = types.BoolValue(role.Mentionable)
			data.Managed = types.BoolValue(role.Managed)

			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"Role Not Found",
		fmt.Sprintf("No role with name %q found in server %s.", name, data.ServerID.ValueString()),
	)
}
