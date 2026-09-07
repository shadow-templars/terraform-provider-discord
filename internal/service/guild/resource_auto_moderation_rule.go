package guild

import (
	"context"
	"fmt"
	"strings"

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
	"github.com/shadow-templars/terraform-provider-discord/internal/discordgox"
)

var (
	_ resource.Resource                = (*AutoModerationRuleResource)(nil)
	_ resource.ResourceWithImportState = (*AutoModerationRuleResource)(nil)
)

type AutoModerationRuleResource struct {
	client *client.DiscordClient
}

type autoModActionMetadataModel struct {
	ChannelID       types.String `tfsdk:"channel_id"`
	DurationSeconds types.Int64  `tfsdk:"duration_seconds"`
	CustomMessage   types.String `tfsdk:"custom_message"`
}

type autoModActionModel struct {
	Type     types.Int64                 `tfsdk:"type"`
	Metadata *autoModActionMetadataModel `tfsdk:"metadata"`
}

type autoModTriggerMetadataModel struct {
	KeywordFilter                types.List  `tfsdk:"keyword_filter"`
	RegexPatterns                types.List  `tfsdk:"regex_patterns"`
	Presets                      types.List  `tfsdk:"presets"`
	AllowList                    types.List  `tfsdk:"allow_list"`
	MentionTotalLimit            types.Int64 `tfsdk:"mention_total_limit"`
	MentionRaidProtectionEnabled types.Bool  `tfsdk:"mention_raid_protection_enabled"`
}

type autoModRuleModel struct {
	ID              types.String                 `tfsdk:"id"`
	ServerID        types.String                 `tfsdk:"server_id"`
	Name            types.String                 `tfsdk:"name"`
	EventType       types.Int64                  `tfsdk:"event_type"`
	TriggerType     types.Int64                  `tfsdk:"trigger_type"`
	TriggerMetadata *autoModTriggerMetadataModel `tfsdk:"trigger_metadata"`
	Actions         []autoModActionModel         `tfsdk:"actions"`
	Enabled         types.Bool                   `tfsdk:"enabled"`
	ExemptRoles     types.List                   `tfsdk:"exempt_roles"`
	ExemptChannels  types.List                   `tfsdk:"exempt_channels"`
}

func NewAutoModerationRuleResource() resource.Resource {
	return &AutoModerationRuleResource{}
}

func (r *AutoModerationRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auto_moderation_rule"
}

func (r *AutoModerationRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord auto moderation rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the auto moderation rule.",
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
				Description: "The name of the rule.",
			},
			"event_type": schema.Int64Attribute{
				Required:    true,
				Description: "The event type (1 = MESSAGE_SEND).",
			},
			"trigger_type": schema.Int64Attribute{
				Required:    true,
				Description: "The trigger type (1 = KEYWORD, 3 = SPAM, 4 = KEYWORD_PRESET, 5 = MENTION_SPAM).",
			},
			"trigger_metadata": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "The trigger metadata.",
				Attributes: map[string]schema.Attribute{
					"keyword_filter": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Keywords to filter (max 1000, each max 60 chars).",
					},
					"regex_patterns": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Regex patterns to match (max 10, each max 260 chars).",
					},
					"presets": schema.ListAttribute{
						Optional:    true,
						ElementType: types.Int64Type,
						Description: "Preset keyword lists (1 = PROFANITY, 2 = SEXUAL_CONTENT, 3 = SLURS).",
					},
					"allow_list": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Substrings which should not trigger the rule.",
					},
					"mention_total_limit": schema.Int64Attribute{
						Optional:    true,
						Description: "Total unique role and user mentions allowed per message (trigger type 5, max 50).",
					},
					"mention_raid_protection_enabled": schema.BoolAttribute{
						Optional:    true,
						Description: "Whether to automatically detect mention raids (trigger type 5).",
					},
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the rule is enabled.",
			},
			"exempt_roles": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Role IDs exempt from this rule (max 20).",
			},
			"exempt_channels": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Channel IDs exempt from this rule (max 50).",
			},
		},
		Blocks: map[string]schema.Block{
			"actions": schema.ListNestedBlock{
				Description: "Actions to execute when the rule is triggered.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.Int64Attribute{
							Required:    true,
							Description: "The action type (1 = BLOCK_MESSAGE, 2 = SEND_ALERT_MESSAGE, 3 = TIMEOUT).",
						},
						"metadata": schema.SingleNestedAttribute{
							Optional:    true,
							Description: "Additional metadata for the action.",
							Attributes: map[string]schema.Attribute{
								"channel_id": schema.StringAttribute{
									Optional:    true,
									Description: "Channel to send alert messages to (for type 2).",
								},
								"duration_seconds": schema.Int64Attribute{
									Optional:    true,
									Description: "Timeout duration in seconds (for type 3, max 2419200).",
								},
								"custom_message": schema.StringAttribute{
									Optional:    true,
									Description: "Custom message shown to the user when blocked (for type 1, max 150 chars).",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *AutoModerationRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AutoModerationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan autoModRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiRule := r.buildAPIRule(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.Session.AutoModerationRuleCreate(plan.ServerID.ValueString(), apiRule)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Auto Moderation Rule", err.Error())
		return
	}

	tflog.Debug(ctx, "created auto moderation rule", map[string]interface{}{
		"id":   rule.ID,
		"name": rule.Name,
	})

	r.refreshState(ctx, rule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AutoModerationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state autoModRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.Session.AutoModerationRule(state.ServerID.ValueString(), state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Auto Moderation Rule", err.Error())
		return
	}

	r.refreshState(ctx, rule, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AutoModerationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan autoModRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiRule := r.buildAPIRule(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.Session.AutoModerationRuleEdit(plan.ServerID.ValueString(), plan.ID.ValueString(), apiRule)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Auto Moderation Rule", err.Error())
		return
	}

	r.refreshState(ctx, rule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AutoModerationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state autoModRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Session.AutoModerationRuleDelete(state.ServerID.ValueString(), state.ID.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Auto Moderation Rule", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted auto moderation rule", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

func (r *AutoModerationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID must be in the format `server_id:rule_id`.")
		return
	}

	serverID := parts[0]
	ruleID := parts[1]

	rule, err := r.client.Session.AutoModerationRule(serverID, ruleID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Auto Moderation Rule", err.Error())
		return
	}

	var state autoModRuleModel
	state.ServerID = types.StringValue(serverID)
	r.refreshState(ctx, rule, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// buildAPIRule converts the Terraform model to a discordgox AutoModRule for create/update.
func (r *AutoModerationRuleResource) buildAPIRule(ctx context.Context, plan *autoModRuleModel, diags *diag.Diagnostics) discordgox.AutoModRule {
	enabled := plan.Enabled.ValueBool()
	apiRule := discordgox.AutoModRule{
		Name:        plan.Name.ValueString(),
		EventType:   discordgo.AutoModerationRuleEventType(plan.EventType.ValueInt64()),
		TriggerType: discordgo.AutoModerationRuleTriggerType(plan.TriggerType.ValueInt64()),
		Enabled:     &enabled,
	}

	if plan.TriggerMetadata != nil {
		meta := &discordgox.AutoModTriggerMetadata{}

		if !plan.TriggerMetadata.KeywordFilter.IsNull() && !plan.TriggerMetadata.KeywordFilter.IsUnknown() {
			var keywords []string
			diags.Append(plan.TriggerMetadata.KeywordFilter.ElementsAs(ctx, &keywords, false)...)
			meta.KeywordFilter = keywords
		}

		if !plan.TriggerMetadata.RegexPatterns.IsNull() && !plan.TriggerMetadata.RegexPatterns.IsUnknown() {
			var patterns []string
			diags.Append(plan.TriggerMetadata.RegexPatterns.ElementsAs(ctx, &patterns, false)...)
			meta.RegexPatterns = patterns
		}

		if !plan.TriggerMetadata.Presets.IsNull() && !plan.TriggerMetadata.Presets.IsUnknown() {
			var presets []int64
			diags.Append(plan.TriggerMetadata.Presets.ElementsAs(ctx, &presets, false)...)
			presetVals := make([]discordgo.AutoModerationKeywordPreset, len(presets))
			for i, p := range presets {
				presetVals[i] = discordgo.AutoModerationKeywordPreset(p)
			}
			meta.Presets = presetVals
		}

		if !plan.TriggerMetadata.AllowList.IsNull() && !plan.TriggerMetadata.AllowList.IsUnknown() {
			var allowList []string
			diags.Append(plan.TriggerMetadata.AllowList.ElementsAs(ctx, &allowList, false)...)
			meta.AllowList = &allowList
		}

		if !plan.TriggerMetadata.MentionTotalLimit.IsNull() && !plan.TriggerMetadata.MentionTotalLimit.IsUnknown() {
			meta.MentionTotalLimit = int(plan.TriggerMetadata.MentionTotalLimit.ValueInt64())
		}

		if !plan.TriggerMetadata.MentionRaidProtectionEnabled.IsNull() && !plan.TriggerMetadata.MentionRaidProtectionEnabled.IsUnknown() {
			v := plan.TriggerMetadata.MentionRaidProtectionEnabled.ValueBool()
			meta.MentionRaidProtectionEnabled = &v
		}

		apiRule.TriggerMetadata = meta
	}

	actions := make([]discordgo.AutoModerationAction, len(plan.Actions))
	for i, a := range plan.Actions {
		action := discordgo.AutoModerationAction{
			Type: discordgo.AutoModerationActionType(a.Type.ValueInt64()),
		}
		if a.Metadata != nil {
			actionMeta := &discordgo.AutoModerationActionMetadata{}
			if !a.Metadata.ChannelID.IsNull() && !a.Metadata.ChannelID.IsUnknown() {
				actionMeta.ChannelID = a.Metadata.ChannelID.ValueString()
			}
			if !a.Metadata.DurationSeconds.IsNull() && !a.Metadata.DurationSeconds.IsUnknown() {
				actionMeta.Duration = int(a.Metadata.DurationSeconds.ValueInt64())
			}
			if !a.Metadata.CustomMessage.IsNull() && !a.Metadata.CustomMessage.IsUnknown() {
				actionMeta.CustomMessage = a.Metadata.CustomMessage.ValueString()
			}
			action.Metadata = actionMeta
		}
		actions[i] = action
	}
	apiRule.Actions = actions

	if !plan.ExemptRoles.IsNull() && !plan.ExemptRoles.IsUnknown() {
		var roles []string
		diags.Append(plan.ExemptRoles.ElementsAs(ctx, &roles, false)...)
		apiRule.ExemptRoles = &roles
	}

	if !plan.ExemptChannels.IsNull() && !plan.ExemptChannels.IsUnknown() {
		var channels []string
		diags.Append(plan.ExemptChannels.ElementsAs(ctx, &channels, false)...)
		apiRule.ExemptChannels = &channels
	}

	return apiRule
}

// refreshState maps a discordgox AutoModRule to the Terraform state model.
func (r *AutoModerationRuleResource) refreshState(_ context.Context, rule *discordgox.AutoModRule, state *autoModRuleModel) {
	state.ID = types.StringValue(rule.ID)
	state.Name = types.StringValue(rule.Name)
	state.EventType = types.Int64Value(int64(rule.EventType))
	state.TriggerType = types.Int64Value(int64(rule.TriggerType))

	if rule.Enabled != nil {
		state.Enabled = types.BoolValue(*rule.Enabled)
	} else {
		state.Enabled = types.BoolValue(true)
	}

	if rule.TriggerMetadata != nil {
		meta := &autoModTriggerMetadataModel{}

		if len(rule.TriggerMetadata.KeywordFilter) > 0 {
			vals := make([]attr.Value, len(rule.TriggerMetadata.KeywordFilter))
			for i, v := range rule.TriggerMetadata.KeywordFilter {
				vals[i] = types.StringValue(v)
			}
			meta.KeywordFilter, _ = types.ListValue(types.StringType, vals)
		} else {
			meta.KeywordFilter = types.ListNull(types.StringType)
		}

		if len(rule.TriggerMetadata.RegexPatterns) > 0 {
			vals := make([]attr.Value, len(rule.TriggerMetadata.RegexPatterns))
			for i, v := range rule.TriggerMetadata.RegexPatterns {
				vals[i] = types.StringValue(v)
			}
			meta.RegexPatterns, _ = types.ListValue(types.StringType, vals)
		} else {
			meta.RegexPatterns = types.ListNull(types.StringType)
		}

		if len(rule.TriggerMetadata.Presets) > 0 {
			vals := make([]attr.Value, len(rule.TriggerMetadata.Presets))
			for i, v := range rule.TriggerMetadata.Presets {
				vals[i] = types.Int64Value(int64(v))
			}
			meta.Presets, _ = types.ListValue(types.Int64Type, vals)
		} else {
			meta.Presets = types.ListNull(types.Int64Type)
		}

		if rule.TriggerMetadata.AllowList != nil && len(*rule.TriggerMetadata.AllowList) > 0 {
			vals := make([]attr.Value, len(*rule.TriggerMetadata.AllowList))
			for i, v := range *rule.TriggerMetadata.AllowList {
				vals[i] = types.StringValue(v)
			}
			meta.AllowList, _ = types.ListValue(types.StringType, vals)
		} else {
			meta.AllowList = types.ListNull(types.StringType)
		}

		if rule.TriggerMetadata.MentionTotalLimit > 0 {
			meta.MentionTotalLimit = types.Int64Value(int64(rule.TriggerMetadata.MentionTotalLimit))
		} else {
			meta.MentionTotalLimit = types.Int64Null()
		}

		if rule.TriggerMetadata.MentionRaidProtectionEnabled != nil {
			meta.MentionRaidProtectionEnabled = types.BoolValue(*rule.TriggerMetadata.MentionRaidProtectionEnabled)
		} else {
			meta.MentionRaidProtectionEnabled = types.BoolNull()
		}

		// Collapse Discord's empty metadata object to null to match config that omits it.
		if meta.KeywordFilter.IsNull() && meta.RegexPatterns.IsNull() &&
			meta.Presets.IsNull() && meta.AllowList.IsNull() &&
			meta.MentionTotalLimit.IsNull() && meta.MentionRaidProtectionEnabled.IsNull() {
			state.TriggerMetadata = nil
		} else {
			state.TriggerMetadata = meta
		}
	} else {
		state.TriggerMetadata = nil
	}

	state.Actions = make([]autoModActionModel, len(rule.Actions))
	for i, a := range rule.Actions {
		am := autoModActionModel{
			Type: types.Int64Value(int64(a.Type)),
		}
		if a.Metadata != nil && (a.Metadata.ChannelID != "" || a.Metadata.Duration != 0 || a.Metadata.CustomMessage != "") {
			meta := &autoModActionMetadataModel{
				ChannelID:       types.StringNull(),
				DurationSeconds: types.Int64Null(),
				CustomMessage:   types.StringNull(),
			}
			if a.Metadata.ChannelID != "" {
				meta.ChannelID = types.StringValue(a.Metadata.ChannelID)
			}
			if a.Metadata.Duration != 0 {
				meta.DurationSeconds = types.Int64Value(int64(a.Metadata.Duration))
			}
			if a.Metadata.CustomMessage != "" {
				meta.CustomMessage = types.StringValue(a.Metadata.CustomMessage)
			}
			am.Metadata = meta
		}
		state.Actions[i] = am
	}

	if rule.ExemptRoles != nil && len(*rule.ExemptRoles) > 0 {
		vals := make([]attr.Value, len(*rule.ExemptRoles))
		for i, v := range *rule.ExemptRoles {
			vals[i] = types.StringValue(v)
		}
		state.ExemptRoles, _ = types.ListValue(types.StringType, vals)
	} else {
		state.ExemptRoles = types.ListNull(types.StringType)
	}

	if rule.ExemptChannels != nil && len(*rule.ExemptChannels) > 0 {
		vals := make([]attr.Value, len(*rule.ExemptChannels))
		for i, v := range *rule.ExemptChannels {
			vals[i] = types.StringValue(v)
		}
		state.ExemptChannels, _ = types.ListValue(types.StringType, vals)
	} else {
		state.ExemptChannels = types.ListNull(types.StringType)
	}
}
