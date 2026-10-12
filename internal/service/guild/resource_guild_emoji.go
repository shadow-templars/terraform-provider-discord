package guild

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var (
	_                resource.Resource                   = (*GuildEmojiResource)(nil)
	_                resource.ResourceWithImportState    = (*GuildEmojiResource)(nil)
	_                resource.ResourceWithValidateConfig = (*GuildEmojiResource)(nil)
	emojiNamePattern                                     = regexp.MustCompile(`^[A-Za-z0-9_]{2,32}$`)
)

type GuildEmojiResource struct {
	client *client.DiscordClient
}

type GuildEmojiResourceModel struct {
	ID       types.String `tfsdk:"id"`
	ServerID types.String `tfsdk:"server_id"`
	Name     types.String `tfsdk:"name"`
	Image    types.String `tfsdk:"image"`
	RoleIDs  types.Set    `tfsdk:"role_ids"`
	Animated types.Bool   `tfsdk:"animated"`
}

func NewGuildEmojiResource() resource.Resource {
	return &GuildEmojiResource{}
}

func (r *GuildEmojiResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_guild_emoji"
}

func (r *GuildEmojiResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord guild custom emoji. Supports static and animated images, role restrictions, and import by server_id:emoji_id. The bot requires CREATE_GUILD_EXPRESSIONS; managing emojis created by others requires MANAGE_GUILD_EXPRESSIONS.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the emoji.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"server_id": schema.StringAttribute{
				Required:      true,
				Description:   "The server (guild) ID. Changing it replaces the emoji.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The emoji name: 2-32 ASCII letters, digits, or underscores. Use shared prefixes such as class_monk and class_wizard to group emojis alphabetically in the picker; upload order does not control grouping.",
			},
			"image": schema.StringAttribute{
				Required:      true,
				Description:   "Base64 image data URI, for example discord_local_image.data_uri. Supports PNG, JPEG, GIF, WebP, and AVIF, up to 256 KiB decoded. Use 128x128 images. Changing the image replaces the emoji and changes its ID; image data cannot be read back from Discord after import.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Role IDs allowed to use this emoji. Omit or set to an empty set to allow everyone. Removing restrictions updates the existing emoji.",
			},
			"animated": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether Discord identifies this emoji as animated.",
			},
		},
	}
}

func (r *GuildEmojiResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.DiscordClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.DiscordClient, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *GuildEmojiResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config GuildEmojiResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !config.Name.IsNull() && !config.Name.IsUnknown() && !emojiNamePattern.MatchString(config.Name.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid Emoji Name", "Emoji names must contain 2-32 ASCII letters, digits, or underscores.")
	}
	if !config.Image.IsNull() && !config.Image.IsUnknown() {
		if err := validateEmojiImage(config.Image.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("image"), "Invalid Emoji Image", err.Error())
		}
	}
}

func (r *GuildEmojiResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GuildEmojiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateEmojiImage(plan.Image.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("image"), "Invalid Emoji Image", err.Error())
		return
	}
	roles := []string{}
	resp.Diagnostics.Append(plan.RoleIDs.ElementsAs(ctx, &roles, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	emoji, err := r.client.Session.GuildEmojiCreate(plan.ServerID.ValueString(), &discordgo.EmojiParams{
		Name: plan.Name.ValueString(), Image: plan.Image.ValueString(), Roles: roles,
	}, discordgo.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Guild Emoji", err.Error())
		return
	}
	plan.ID = types.StringValue(emoji.ID)
	plan.Animated = types.BoolValue(emoji.Animated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GuildEmojiResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GuildEmojiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	emoji, err := r.client.Session.GuildEmoji(state.ServerID.ValueString(), state.ID.ValueString(), discordgo.WithContext(ctx))
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Guild Emoji", err.Error())
		return
	}
	state.Name = types.StringValue(emoji.Name)
	state.Animated = types.BoolValue(emoji.Animated)
	// Preserve omitted roles as null, but retain an explicitly configured empty set.
	if len(emoji.Roles) != 0 || !state.RoleIDs.IsNull() {
		roles := emoji.Roles
		if roles == nil {
			roles = []string{}
		}
		value, diags := types.SetValueFrom(ctx, types.StringType, roles)
		resp.Diagnostics.Append(diags...)
		state.RoleIDs = value
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GuildEmojiResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GuildEmojiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	roles := []string{}
	resp.Diagnostics.Append(plan.RoleIDs.ElementsAs(ctx, &roles, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	emoji, err := r.client.Session.GuildEmojiEdit(plan.ServerID.ValueString(), plan.ID.ValueString(), &discordgo.EmojiParams{
		Name: plan.Name.ValueString(), Roles: roles,
	}, discordgo.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Guild Emoji", err.Error())
		return
	}
	plan.Animated = types.BoolValue(emoji.Animated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GuildEmojiResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GuildEmojiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Session.GuildEmojiDelete(state.ServerID.ValueString(), state.ID.ValueString(), discordgo.WithContext(ctx)); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Guild Emoji", err.Error())
	}
}

func (r *GuildEmojiResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID must be in the format `server_id:emoji_id` with both IDs nonempty.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func validateEmojiImage(image string) error {
	header, encoded, ok := strings.Cut(image, ",")
	if !ok {
		return fmt.Errorf("Expected a base64 image data URI, such as data:image/png;base64,...")
	}
	switch header {
	case "data:image/png;base64", "data:image/jpeg;base64", "data:image/gif;base64", "data:image/webp;base64", "data:image/avif;base64":
	default:
		return fmt.Errorf("Expected a base64 PNG, JPEG, GIF, WebP, or AVIF image data URI")
	}
	// Reject oversized input before allocating its decoded bytes.
	if len(encoded) > base64.StdEncoding.EncodedLen(256*1024) {
		return fmt.Errorf("Decoded emoji image must not exceed 256 KiB")
	}
	data, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(data) == 0 {
		return fmt.Errorf("Emoji image must contain nonempty, valid base64 data")
	}
	if len(data) > 256*1024 {
		return fmt.Errorf("Decoded emoji image must not exceed 256 KiB")
	}
	return nil
}
