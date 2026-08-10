package guild

import (
	"context"
	"encoding/json"
	"fmt"

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
	_ resource.Resource                = (*MessageResource)(nil)
	_ resource.ResourceWithImportState = (*MessageResource)(nil)
)

type MessageResource struct {
	client *client.DiscordClient
}

type MessageResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ChannelID types.String `tfsdk:"channel_id"`
	Content   types.String `tfsdk:"content"`
	Embeds    types.List   `tfsdk:"embed"`
}

type EmbedModel struct {
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	Color       types.Int64  `tfsdk:"color"`
	URL         types.String `tfsdk:"url"`
}

var embedAttrTypes = map[string]attr.Type{
	"title":       types.StringType,
	"description": types.StringType,
	"color":       types.Int64Type,
	"url":         types.StringType,
}

func NewMessageResource() resource.Resource {
	return &MessageResource{}
}

func (r *MessageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_message"
}

func (r *MessageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord message in a channel.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the message.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"channel_id": schema.StringAttribute{
				Required:    true,
				Description: "The channel ID to send the message in.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				Optional:    true,
				Description: "The text content of the message.",
			},
		},
		Blocks: map[string]schema.Block{
			"embed": schema.ListNestedBlock{
				Description: "Embedded rich content.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"title": schema.StringAttribute{
							Optional:    true,
							Description: "Title of the embed.",
						},
						"description": schema.StringAttribute{
							Optional:    true,
							Description: "Description of the embed.",
						},
						"color": schema.Int64Attribute{
							Optional:    true,
							Description: "Color code of the embed (decimal).",
						},
						"url": schema.StringAttribute{
							Optional:    true,
							Description: "URL of the embed.",
						},
					},
				},
			},
		},
	}
}

func (r *MessageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MessageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MessageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	msgSend := &discordgo.MessageSend{
		Content: plan.Content.ValueString(),
		Embeds:  r.buildEmbeds(ctx, &plan, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	msg, err := r.client.Session.ChannelMessageSendComplex(plan.ChannelID.ValueString(), msgSend)
	if err != nil {
		resp.Diagnostics.AddError("Error Sending Message", err.Error())
		return
	}

	tflog.Debug(ctx, "created message", map[string]interface{}{"id": msg.ID})
	plan.ID = types.StringValue(msg.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MessageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MessageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Session.ChannelMessage(state.ChannelID.ValueString(), state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Message", err.Error())
		return
	}

	// Keep state as-is — message content is managed by Terraform, not read back.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MessageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MessageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	embeds := r.buildEmbeds(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	edit := &discordgo.MessageEdit{
		ID:      plan.ID.ValueString(),
		Channel: plan.ChannelID.ValueString(),
		Content: strPtr(plan.Content.ValueString()),
		Embeds:  &embeds,
	}

	_, err := r.client.Session.ChannelMessageEditComplex(edit)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Message", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MessageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MessageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Session.ChannelMessageDelete(state.ChannelID.ValueString(), state.ID.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Message", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted message", map[string]interface{}{"id": state.ID.ValueString()})
}

func (r *MessageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: channel_id:message_id
	parts := splitImportID(req.ID, 2)
	if parts == nil {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID must be in the format `channel_id:message_id`.")
		return
	}

	msg, err := r.client.Session.ChannelMessage(parts[0], parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Message", err.Error())
		return
	}

	state := MessageResourceModel{
		ID:        types.StringValue(msg.ID),
		ChannelID: types.StringValue(parts[0]),
		Content:   types.StringValue(msg.Content),
		Embeds:    types.ListNull(types.ObjectType{AttrTypes: embedAttrTypes}),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MessageResource) buildEmbeds(ctx context.Context, plan *MessageResourceModel, diags *diag.Diagnostics) []*discordgo.MessageEmbed {
	if plan.Embeds.IsNull() || plan.Embeds.IsUnknown() {
		return nil
	}

	var embedModels []EmbedModel
	diags.Append(plan.Embeds.ElementsAs(ctx, &embedModels, false)...)
	if diags.HasError() {
		return nil
	}

	embeds := make([]*discordgo.MessageEmbed, len(embedModels))
	for i, em := range embedModels {
		embeds[i] = &discordgo.MessageEmbed{
			Title:       em.Title.ValueString(),
			Description: em.Description.ValueString(),
			Color:       int(em.Color.ValueInt64()),
			URL:         em.URL.ValueString(),
		}
	}
	return embeds
}

func splitImportID(id string, n int) []string {
	parts := make([]string, 0, n)
	for i := 0; i < n-1; i++ {
		idx := indexOf(id, ':')
		if idx == -1 {
			return nil
		}
		parts = append(parts, id[:idx])
		id = id[idx+1:]
	}
	parts = append(parts, id)
	if len(parts) != n {
		return nil
	}
	return parts
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func strPtr(s string) *string {
	return &s
}

// Ensure json is used (for potential future use with raw API calls).
var _ = json.Marshal
