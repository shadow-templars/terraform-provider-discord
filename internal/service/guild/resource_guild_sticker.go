package guild

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var (
	_ resource.Resource                = (*GuildStickerResource)(nil)
	_ resource.ResourceWithImportState = (*GuildStickerResource)(nil)
)

type GuildStickerResource struct {
	client *client.DiscordClient
}

type GuildStickerResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ServerID    types.String `tfsdk:"server_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Tags        types.String `tfsdk:"tags"`
	File        types.String `tfsdk:"file"`
	FormatType  types.Int64  `tfsdk:"format_type"`
}

func NewGuildStickerResource() resource.Resource {
	return &GuildStickerResource{}
}

func (r *GuildStickerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_guild_sticker"
}

func (r *GuildStickerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord guild sticker.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the sticker.",
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
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the sticker (2-30 characters).",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the sticker (empty or 2-100 characters).",
			},
			"tags": schema.StringAttribute{
				Required:    true,
				Description: "The autocomplete/suggestion tags for the sticker (max 200 characters).",
			},
			"file": schema.StringAttribute{
				Required:    true,
				Description: "Path to the sticker image file (PNG, APNG, GIF, or Lottie JSON). Max 512 KB.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"format_type": schema.Int64Attribute{
				Computed:    true,
				Description: "The format type of the sticker (1 = PNG, 2 = APNG, 3 = LOTTIE, 4 = GIF).",
			},
		},
	}
}

func (r *GuildStickerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GuildStickerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GuildStickerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filePath := plan.File.ValueString()
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Sticker File", fmt.Sprintf("Could not read file %q: %s", filePath, err.Error()))
		return
	}

	sticker, err := r.client.Session.GuildStickerCreate(
		plan.ServerID.ValueString(),
		plan.Name.ValueString(),
		plan.Description.ValueString(),
		plan.Tags.ValueString(),
		filePath,
		fileData,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Guild Sticker", err.Error())
		return
	}

	tflog.Debug(ctx, "created guild sticker", map[string]interface{}{
		"id":   sticker.ID,
		"name": sticker.Name,
	})

	plan.ID = types.StringValue(sticker.ID)
	plan.FormatType = types.Int64Value(int64(sticker.FormatType))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GuildStickerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GuildStickerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sticker, err := r.client.Session.GuildSticker(state.ServerID.ValueString(), state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Guild Sticker", err.Error())
		return
	}

	state.Name = types.StringValue(sticker.Name)
	state.Description = types.StringValue(sticker.Description)
	state.Tags = types.StringValue(sticker.Tags)
	state.FormatType = types.Int64Value(int64(sticker.FormatType))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GuildStickerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GuildStickerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Session.GuildStickerEdit(
		plan.ServerID.ValueString(),
		plan.ID.ValueString(),
		plan.Name.ValueString(),
		plan.Description.ValueString(),
		plan.Tags.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Guild Sticker", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GuildStickerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GuildStickerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Session.GuildStickerDelete(state.ServerID.ValueString(), state.ID.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Guild Sticker", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted guild sticker", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

func (r *GuildStickerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID must be in the format `server_id:sticker_id`.")
		return
	}

	serverID := parts[0]
	stickerID := parts[1]

	sticker, err := r.client.Session.GuildSticker(serverID, stickerID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Guild Sticker", err.Error())
		return
	}

	state := GuildStickerResourceModel{
		ID:          types.StringValue(sticker.ID),
		ServerID:    types.StringValue(serverID),
		Name:        types.StringValue(sticker.Name),
		Description: types.StringValue(sticker.Description),
		Tags:        types.StringValue(sticker.Tags),
		File:        types.StringValue(""),
		FormatType:  types.Int64Value(int64(sticker.FormatType)),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
