package discordgox

import (
	"encoding/json"

	"github.com/bwmarrin/discordgo"
)

// AutoModTriggerMetadata mirrors discordgo.AutoModerationTriggerMetadata but adds
// mention_raid_protection_enabled, which discordgo (v0.29.0) does not model.
type AutoModTriggerMetadata struct {
	KeywordFilter                []string                                `json:"keyword_filter,omitempty"`
	RegexPatterns                []string                                `json:"regex_patterns,omitempty"`
	Presets                      []discordgo.AutoModerationKeywordPreset `json:"presets,omitempty"`
	AllowList                    *[]string                               `json:"allow_list,omitempty"`
	MentionTotalLimit            int                                     `json:"mention_total_limit,omitempty"`
	MentionRaidProtectionEnabled *bool                                   `json:"mention_raid_protection_enabled,omitempty"`
}

// AutoModRule mirrors discordgo.AutoModerationRule but uses the extended trigger
// metadata above. Only the fields the provider sends/reads are included.
type AutoModRule struct {
	ID              string                                  `json:"id,omitempty"`
	Name            string                                  `json:"name,omitempty"`
	EventType       discordgo.AutoModerationRuleEventType   `json:"event_type,omitempty"`
	TriggerType     discordgo.AutoModerationRuleTriggerType `json:"trigger_type,omitempty"`
	TriggerMetadata *AutoModTriggerMetadata                 `json:"trigger_metadata,omitempty"`
	Actions         []discordgo.AutoModerationAction        `json:"actions,omitempty"`
	Enabled         *bool                                   `json:"enabled,omitempty"`
	ExemptRoles     *[]string                               `json:"exempt_roles,omitempty"`
	ExemptChannels  *[]string                               `json:"exempt_channels,omitempty"`
}

// AutoModerationRuleCreate creates a rule, sending mention_raid_protection_enabled.
func (c *Client) AutoModerationRuleCreate(guildID string, rule AutoModRule) (*AutoModRule, error) {
	endpoint := discordgo.EndpointGuildAutoModerationRules(guildID)
	return c.autoModRequest("POST", endpoint, rule)
}

// AutoModerationRuleEdit edits a rule, sending mention_raid_protection_enabled.
func (c *Client) AutoModerationRuleEdit(guildID, ruleID string, rule AutoModRule) (*AutoModRule, error) {
	endpoint := discordgo.EndpointGuildAutoModerationRule(guildID, ruleID)
	return c.autoModRequest("PATCH", endpoint, rule)
}

// AutoModerationRule fetches a single rule, including the extended metadata.
func (c *Client) AutoModerationRule(guildID, ruleID string) (*AutoModRule, error) {
	endpoint := discordgo.EndpointGuildAutoModerationRule(guildID, ruleID)
	return c.autoModRequest("GET", endpoint, nil)
}

func (c *Client) autoModRequest(method, endpoint string, body interface{}) (*AutoModRule, error) {
	resp, err := c.RequestWithBucketID(method, endpoint, body, endpoint)
	if err != nil {
		return nil, err
	}
	var rule AutoModRule
	if err := json.Unmarshal(resp, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}
