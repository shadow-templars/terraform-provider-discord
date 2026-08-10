package channel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var (
	_ resource.Resource                = (*WebhookResource)(nil)
	_ resource.ResourceWithImportState = (*WebhookResource)(nil)
)

type WebhookResource struct {
	client *client.DiscordClient
}

type WebhookResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ChannelID types.String `tfsdk:"channel_id"`
	Name      types.String `tfsdk:"name"`
	Avatar    types.String `tfsdk:"avatar_data_uri"`
	URL       types.String `tfsdk:"url"`
	Token     types.String `tfsdk:"token"`
}

func NewWebhookResource() resource.Resource {
	return &WebhookResource{}
}

func (r *WebhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *WebhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord webhook.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the webhook.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"channel_id": schema.StringAttribute{
				Required:    true,
				Description: "The channel ID to create the webhook in.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the webhook (1-80 characters).",
			},
			"avatar_data_uri": schema.StringAttribute{
				Optional:    true,
				Description: "The avatar data URI for the webhook (data:image/...;base64,...).",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The full webhook URL including the token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"token": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The webhook token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *WebhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	avatar := ""
	if !plan.Avatar.IsNull() && !plan.Avatar.IsUnknown() {
		avatar = plan.Avatar.ValueString()
	}

	wh, err := r.client.Session.WebhookCreate(plan.ChannelID.ValueString(), name, avatar)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Webhook", err.Error())
		return
	}

	tflog.Debug(ctx, "created webhook", map[string]interface{}{
		"id":   wh.ID,
		"name": wh.Name,
	})

	plan.ID = types.StringValue(wh.ID)
	plan.Token = types.StringValue(wh.Token)
	plan.URL = types.StringValue(fmt.Sprintf("https://discord.com/api/webhooks/%s/%s", wh.ID, wh.Token))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wh, err := r.client.Session.Webhook(state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Webhook", err.Error())
		return
	}

	state.ChannelID = types.StringValue(wh.ChannelID)
	state.Name = types.StringValue(wh.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	avatar := ""
	if !plan.Avatar.IsNull() && !plan.Avatar.IsUnknown() {
		avatar = plan.Avatar.ValueString()
	}
	channelID := plan.ChannelID.ValueString()

	_, err := r.client.Session.WebhookEdit(plan.ID.ValueString(), name, avatar, channelID)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Webhook", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Session.WebhookDelete(state.ID.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Webhook", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted webhook", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	wh, err := r.client.Session.Webhook(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Webhook", err.Error())
		return
	}

	state := WebhookResourceModel{
		ID:        types.StringValue(wh.ID),
		ChannelID: types.StringValue(wh.ChannelID),
		Name:      types.StringValue(wh.Name),
		Token:     types.StringValue(wh.Token),
		URL:       types.StringValue(fmt.Sprintf("https://discord.com/api/webhooks/%s/%s", wh.ID, wh.Token)),
	}
	if wh.Avatar != "" {
		state.Avatar = types.StringValue(wh.Avatar)
	} else {
		state.Avatar = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
