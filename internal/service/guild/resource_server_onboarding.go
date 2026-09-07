package guild

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var _ resource.Resource = (*ServerOnboardingResource)(nil)

type ServerOnboardingResource struct {
	client *client.DiscordClient
}

type onboardingPromptOptionModel struct {
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	EmojiName   types.String `tfsdk:"emoji_name"`
	ChannelIDs  types.List   `tfsdk:"channel_ids"`
	RoleIDs     types.List   `tfsdk:"role_ids"`
}

type onboardingPromptModel struct {
	Title        types.String                  `tfsdk:"title"`
	SingleSelect types.Bool                    `tfsdk:"single_select"`
	Required     types.Bool                    `tfsdk:"required"`
	InOnboarding types.Bool                    `tfsdk:"in_onboarding"`
	Options      []onboardingPromptOptionModel `tfsdk:"options"`
}

type serverOnboardingModel struct {
	ID                types.String            `tfsdk:"id"`
	ServerID          types.String            `tfsdk:"server_id"`
	Enabled           types.Bool              `tfsdk:"enabled"`
	Mode              types.Int64             `tfsdk:"mode"`
	DefaultChannelIDs types.List              `tfsdk:"default_channel_ids"`
	Prompts           []onboardingPromptModel `tfsdk:"prompts"`
}

func NewServerOnboardingResource() resource.Resource {
	return &ServerOnboardingResource{}
}

func (r *ServerOnboardingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_onboarding"
}

func (r *ServerOnboardingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Discord server onboarding configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID (same as server_id).",
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
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether onboarding is enabled.",
			},
			"mode": schema.Int64Attribute{
				Required:    true,
				Description: "Onboarding mode (0 = default, 1 = advanced).",
			},
			"default_channel_ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Channel IDs that members get opted into automatically.",
			},
		},
		Blocks: map[string]schema.Block{
			"prompts": schema.ListNestedBlock{
				Description: "Prompts shown during onboarding.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"title": schema.StringAttribute{
							Required:    true,
							Description: "Title of the prompt.",
						},
						"single_select": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "Whether users are limited to selecting one option.",
						},
						"required": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "Whether the prompt is required before completing onboarding.",
						},
						"in_onboarding": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
							Description: "Whether the prompt is shown in the onboarding flow.",
						},
					},
					Blocks: map[string]schema.Block{
						"options": schema.ListNestedBlock{
							Description: "Options within the prompt.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"title": schema.StringAttribute{
										Required:    true,
										Description: "Title of the option.",
									},
									"description": schema.StringAttribute{
										Optional:    true,
										Description: "Description of the option.",
									},
									"emoji_name": schema.StringAttribute{
										Optional:    true,
										Description: "Name of the emoji for this option (Unicode or custom emoji name).",
									},
									"channel_ids": schema.ListAttribute{
										Optional:    true,
										ElementType: types.StringType,
										Description: "Channel IDs a member is added to when the option is selected.",
									},
									"role_ids": schema.ListAttribute{
										Optional:    true,
										ElementType: types.StringType,
										Description: "Role IDs assigned to a member when the option is selected.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *ServerOnboardingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServerOnboardingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverOnboardingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiOnboarding := r.buildAPIOnboarding(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Session.GuildOnboardingEdit(plan.ServerID.ValueString(), apiOnboarding)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Server Onboarding", err.Error())
		return
	}

	tflog.Debug(ctx, "created server onboarding", map[string]interface{}{
		"guild_id": plan.ServerID.ValueString(),
	})

	r.refreshState(ctx, result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ServerOnboardingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverOnboardingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Session.GuildOnboarding(state.ServerID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Server Onboarding", err.Error())
		return
	}

	r.refreshState(ctx, result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ServerOnboardingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serverOnboardingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiOnboarding := r.buildAPIOnboarding(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Session.GuildOnboardingEdit(plan.ServerID.ValueString(), apiOnboarding)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Server Onboarding", err.Error())
		return
	}

	r.refreshState(ctx, result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the resource from state only — onboarding cannot be deleted via the API.
func (r *ServerOnboardingResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Debug(ctx, "removing server onboarding from state (cannot be deleted via API)")
}

// buildAPIOnboarding converts the Terraform model to a discordgo GuildOnboarding.
func (r *ServerOnboardingResource) buildAPIOnboarding(ctx context.Context, plan *serverOnboardingModel, diags *diag.Diagnostics) *discordgo.GuildOnboarding {
	enabled := plan.Enabled.ValueBool()
	mode := discordgo.GuildOnboardingMode(plan.Mode.ValueInt64())

	onboarding := &discordgo.GuildOnboarding{
		GuildID: plan.ServerID.ValueString(),
		Enabled: &enabled,
		Mode:    &mode,
	}

	if !plan.DefaultChannelIDs.IsNull() && !plan.DefaultChannelIDs.IsUnknown() {
		var channelIDs []string
		diags.Append(plan.DefaultChannelIDs.ElementsAs(ctx, &channelIDs, false)...)
		onboarding.DefaultChannelIDs = channelIDs
	}

	if len(plan.Prompts) > 0 {
		prompts := make([]discordgo.GuildOnboardingPrompt, len(plan.Prompts))
		for i, p := range plan.Prompts {
			prompt := discordgo.GuildOnboardingPrompt{
				// Use "0" as ID for new prompts per Discord API requirement.
				ID:           "0",
				Title:        p.Title.ValueString(),
				SingleSelect: p.SingleSelect.ValueBool(),
				Required:     p.Required.ValueBool(),
				InOnboarding: p.InOnboarding.ValueBool(),
				Type:         discordgo.GuildOnboardingPromptTypeMultipleChoice,
			}

			if p.SingleSelect.ValueBool() {
				prompt.Type = discordgo.GuildOnboardingPromptTypeDropdown
			}

			prompt.Options = make([]discordgo.GuildOnboardingPromptOption, len(p.Options))
			for j, o := range p.Options {
				opt := discordgo.GuildOnboardingPromptOption{
					ID:    "0",
					Title: o.Title.ValueString(),
				}

				if !o.Description.IsNull() && !o.Description.IsUnknown() {
					opt.Description = o.Description.ValueString()
				}

				if !o.EmojiName.IsNull() && !o.EmojiName.IsUnknown() {
					opt.EmojiName = o.EmojiName.ValueString()
				}

				if !o.ChannelIDs.IsNull() && !o.ChannelIDs.IsUnknown() {
					var ids []string
					diags.Append(o.ChannelIDs.ElementsAs(ctx, &ids, false)...)
					opt.ChannelIDs = ids
				} else {
					opt.ChannelIDs = []string{}
				}

				if !o.RoleIDs.IsNull() && !o.RoleIDs.IsUnknown() {
					var ids []string
					diags.Append(o.RoleIDs.ElementsAs(ctx, &ids, false)...)
					opt.RoleIDs = ids
				} else {
					opt.RoleIDs = []string{}
				}

				prompt.Options[j] = opt
			}

			prompts[i] = prompt
		}
		onboarding.Prompts = &prompts
	}

	return onboarding
}

// refreshState maps a discordgo GuildOnboarding response to the Terraform state model.
func (r *ServerOnboardingResource) refreshState(_ context.Context, onboarding *discordgo.GuildOnboarding, state *serverOnboardingModel) {
	state.ID = types.StringValue(onboarding.GuildID)
	state.ServerID = types.StringValue(onboarding.GuildID)

	if onboarding.Enabled != nil {
		state.Enabled = types.BoolValue(*onboarding.Enabled)
	} else {
		state.Enabled = types.BoolValue(false)
	}

	if onboarding.Mode != nil {
		state.Mode = types.Int64Value(int64(*onboarding.Mode))
	} else {
		state.Mode = types.Int64Value(0)
	}

	if len(onboarding.DefaultChannelIDs) > 0 {
		vals := make([]attr.Value, len(onboarding.DefaultChannelIDs))
		for i, v := range onboarding.DefaultChannelIDs {
			vals[i] = types.StringValue(v)
		}
		state.DefaultChannelIDs, _ = types.ListValue(types.StringType, vals)
	} else {
		state.DefaultChannelIDs = types.ListNull(types.StringType)
	}

	if onboarding.Prompts != nil && len(*onboarding.Prompts) > 0 {
		prompts := *onboarding.Prompts
		state.Prompts = make([]onboardingPromptModel, len(prompts))
		for i, p := range prompts {
			pm := onboardingPromptModel{
				Title:        types.StringValue(p.Title),
				SingleSelect: types.BoolValue(p.SingleSelect),
				Required:     types.BoolValue(p.Required),
				InOnboarding: types.BoolValue(p.InOnboarding),
			}

			pm.Options = make([]onboardingPromptOptionModel, len(p.Options))
			for j, o := range p.Options {
				om := onboardingPromptOptionModel{
					Title:       types.StringValue(o.Title),
					Description: types.StringValue(o.Description),
				}

				if o.EmojiName != "" {
					om.EmojiName = types.StringValue(o.EmojiName)
				} else if o.Emoji != nil && o.Emoji.Name != "" {
					om.EmojiName = types.StringValue(o.Emoji.Name)
				} else {
					om.EmojiName = types.StringNull()
				}

				if len(o.ChannelIDs) > 0 {
					vals := make([]attr.Value, len(o.ChannelIDs))
					for k, v := range o.ChannelIDs {
						vals[k] = types.StringValue(v)
					}
					om.ChannelIDs, _ = types.ListValue(types.StringType, vals)
				} else {
					om.ChannelIDs = types.ListNull(types.StringType)
				}

				if len(o.RoleIDs) > 0 {
					vals := make([]attr.Value, len(o.RoleIDs))
					for k, v := range o.RoleIDs {
						vals[k] = types.StringValue(v)
					}
					om.RoleIDs, _ = types.ListValue(types.StringType, vals)
				} else {
					om.RoleIDs = types.ListNull(types.StringType)
				}

				pm.Options[j] = om
			}

			state.Prompts[i] = pm
		}
	} else {
		state.Prompts = []onboardingPromptModel{}
	}
}
