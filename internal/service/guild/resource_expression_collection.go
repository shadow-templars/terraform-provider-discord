package guild

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
)

var (
	_ resource.Resource                   = (*ExpressionCollectionResource)(nil)
	_ resource.ResourceWithModifyPlan     = (*ExpressionCollectionResource)(nil)
	_ resource.ResourceWithValidateConfig = (*ExpressionCollectionResource)(nil)
	_ resource.ResourceWithImportState    = (*ExpressionCollectionResource)(nil)
)

// ExpressionCollectionResource owns only the IDs recorded in its state. Each
// collection performs its own requests synchronously, including verification.
// Independent collections and individual resources may still run concurrently.
type ExpressionCollectionResource struct {
	client *client.DiscordClient
	kind   string
}

type collectionEntry struct {
	Key         string
	Name        string
	Image       string
	Roles       []string
	File        string
	Description string
	Tags        string
	ContentHash string
	fileData    []byte
}

type collectionReader interface {
	GetAttribute(context.Context, path.Path, interface{}) diag.Diagnostics
}

func NewEmojiCollectionResource() resource.Resource {
	return &ExpressionCollectionResource{kind: "emoji"}
}

func NewStickerCollectionResource() resource.Resource {
	return &ExpressionCollectionResource{kind: "sticker"}
}

func (r *ExpressionCollectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.kind + "_collection"
}

func (r *ExpressionCollectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"key": schema.StringAttribute{
			Required:    true,
			Description: "Unique stable key used in the ids map. Changing it rebuilds the collection.",
		},
		"name": schema.StringAttribute{
			Required:    true,
			Description: "Discord expression name.",
		},
	}
	if r.kind == "emoji" {
		attributes["image"] = schema.StringAttribute{
			Required:    true,
			Description: "Base64 image data URI, for example discord_local_image.data_uri, up to 256 KiB decoded. Image changes rebuild the collection.",
		}
		attributes["role_ids"] = schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Roles allowed to use this emoji; omitted or empty allows everyone.",
		}
	} else {
		attributes["file"] = schema.StringAttribute{
			Required:    true,
			Description: "Path to a PNG, APNG, GIF, or Lottie JSON file, up to 512 KiB. File contents are fingerprinted at plan and apply.",
		}
		attributes["description"] = schema.StringAttribute{
			Optional:    true,
			Description: "Empty or 2-100 character sticker description.",
		}
		attributes["tags"] = schema.StringAttribute{
			Required:    true,
			Description: "Sticker suggestion tags, up to 200 characters.",
		}
	}
	resp.Schema = schema.Schema{
		Description: "Manages an ordered collection of guild " + r.kind + "s. Uploads and verifies each entry sequentially in block order. Only recorded IDs are deleted. Do not also manage those IDs with individual resources. Discord does not expose picker positions or remote image hashes. Independent collections can run concurrently; use one collection per expression type or explicit depends_on. Import format: server_id:key=id,key=id.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Collection identifier (server ID and expression type).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"server_id": schema.StringAttribute{
				Required:      true,
				Description:   "Guild ID. Changing it replaces the collection.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"rebuild_on_change": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Explicitly permit deleting ALL managed entries before sequential recreation when entries, order, artwork, or observed metadata change. Default false refuses such changes. IDs change, existing references may become invalid, and rate limits may make the collection unavailable for an extended period. Never use create_before_destroy.",
			},
			"ids": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Discord IDs indexed by stable entry key. Partial failures retain successful uploads for cleanup on the next apply.",
			},
			"fingerprint": schema.StringAttribute{
				Computed:    true,
				Description: "Hash of ordered configuration and local artwork. Cleared on observed drift or partial failure so a subsequent plan repairs the collection.",
			},
		},
		Blocks: map[string]schema.Block{
			r.kind: schema.ListNestedBlock{
				Description: "Entries in upload order. All existing managed entries are deleted before any replacement is uploaded.",
				NestedObject: schema.NestedBlockObject{
					Attributes: attributes,
				},
			},
		},
	}
}

func (r *ExpressionCollectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// entries validates the entire collection before any destructive requests. The
// returned fingerprint includes list order and file bytes, with roles sorted.
func (r *ExpressionCollectionResource) entries(ctx context.Context, source collectionReader, diags *diag.Diagnostics) ([]collectionEntry, string, bool) {
	var items types.List
	diags.Append(source.GetAttribute(ctx, path.Root(r.kind), &items)...)
	if diags.HasError() || items.IsUnknown() {
		return nil, "", false
	}
	entries := make([]collectionEntry, 0, len(items.Elements()))
	keys, names := map[string]bool{}, map[string]bool{}
	for i, raw := range items.Elements() {
		object := raw.(types.Object)
		if object.IsUnknown() {
			return nil, "", false
		}
		attrs := object.Attributes()
		for _, value := range attrs {
			if value.IsUnknown() {
				return nil, "", false
			}
			if set, ok := value.(types.Set); ok {
				for _, element := range set.Elements() {
					if element.IsUnknown() {
						return nil, "", false
					}
				}
			}
		}
		stringValue := func(name string) string {
			if value, ok := attrs[name].(types.String); ok {
				return value.ValueString()
			}
			return ""
		}
		entry := collectionEntry{Key: stringValue("key"), Name: stringValue("name"), Image: stringValue("image"), File: stringValue("file"), Description: stringValue("description"), Tags: stringValue("tags"), Roles: []string{}}
		itemPath := path.Root(r.kind).AtListIndex(i)
		if entry.Key == "" || strings.ContainsAny(entry.Key, ":=,") || keys[entry.Key] {
			diags.AddAttributeError(itemPath.AtName("key"), "Invalid Collection Key", "Keys must be nonempty, unique, and contain no colon, equals sign, or comma.")
		}
		if names[entry.Name] {
			diags.AddAttributeError(itemPath.AtName("name"), "Duplicate Expression Name", "Names must be unique within the collection.")
		}
		keys[entry.Key], names[entry.Name] = true, true
		if r.kind == "emoji" {
			if !emojiNamePattern.MatchString(entry.Name) {
				diags.AddAttributeError(itemPath.AtName("name"), "Invalid Emoji Name", "Emoji names must contain 2-32 ASCII letters, digits, or underscores.")
			}
			if err := validateEmojiImage(entry.Image); err != nil {
				diags.AddAttributeError(itemPath.AtName("image"), "Invalid Emoji Image", err.Error())
			}
			roles := attrs["role_ids"].(types.Set)
			diags.Append(roles.ElementsAs(ctx, &entry.Roles, false)...)
			if entry.Roles == nil {
				entry.Roles = []string{}
			}
			sort.Strings(entry.Roles)
		} else {
			n, d, tags := utf8.RuneCountInString(entry.Name), utf8.RuneCountInString(entry.Description), utf8.RuneCountInString(entry.Tags)
			if n < 2 || n > 30 {
				diags.AddAttributeError(itemPath.AtName("name"), "Invalid Sticker Name", "Sticker names must contain 2-30 characters.")
			}
			if d == 1 || d > 100 {
				diags.AddAttributeError(itemPath.AtName("description"), "Invalid Sticker Description", "Descriptions must be empty or contain 2-100 characters.")
			}
			if tags == 0 || tags > 200 {
				diags.AddAttributeError(itemPath.AtName("tags"), "Invalid Sticker Tags", "Tags must contain 1-200 characters.")
			}
			data, err := os.ReadFile(entry.File)
			if err != nil {
				diags.AddAttributeError(itemPath.AtName("file"), "Error Reading Sticker File", err.Error())
			} else if len(data) == 0 || len(data) > 512*1024 {
				diags.AddAttributeError(itemPath.AtName("file"), "Invalid Sticker File", "Sticker files must be nonempty and at most 512 KiB.")
			} else {
				entry.fileData = data
				hash := sha256.Sum256(data)
				entry.ContentHash = hex.EncodeToString(hash[:])
			}
		}
		entries = append(entries, entry)
	}
	encoded, err := json.Marshal(entries)
	if err != nil {
		diags.AddError("Error Fingerprinting Collection", err.Error())
		return nil, "", false
	}
	hash := sha256.Sum256(encoded)
	return entries, hex.EncodeToString(hash[:]), !diags.HasError()
}

func (r *ExpressionCollectionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	r.entries(ctx, req.Config, &resp.Diagnostics)
}

func (r *ExpressionCollectionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	_, hash, known := r.entries(ctx, req.Plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if known {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("fingerprint"), hash)...)
	} else {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("fingerprint"), types.StringUnknown())...)
	}
	if req.State.Raw.IsNull() {
		return
	}
	var previousServer, plannedServer types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("server_id"), &previousServer)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("server_id"), &plannedServer)...)
	if !previousServer.Equal(plannedServer) {
		// A guild change replaces the resource. Old IDs must not be promised
		// by the plan for the new guild, even when its entries are identical.
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("id"), types.StringUnknown())...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("ids"), types.MapUnknown(types.StringType))...)
		resp.Diagnostics.AddWarning("Entire Collection Will Be Replaced", "Changing server_id deletes all owned expressions in the old guild and sequentially creates them in the new guild. Every Discord ID changes; do not use create_before_destroy.")
		return
	}
	var previous types.String
	var enabled types.Bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("fingerprint"), &previous)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("rebuild_on_change"), &enabled)...)
	if !known {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("ids"), types.MapUnknown(types.StringType))...)
		if enabled.IsUnknown() || enabled.ValueBool() {
			resp.Diagnostics.AddWarning("Collection May Be Rebuilt", "Some entry values are unknown until apply. If they change, all managed entries will be deleted before sequential recreation, every Discord ID will change, and existing references may become invalid. Rebuilding still requires rebuild_on_change = true at apply.")
		}
		return
	}
	if previous.ValueString() == hash {
		var ids types.Map
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("ids"), &ids)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("ids"), ids)...)
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("ids"), types.MapUnknown(types.StringType))...)
	if enabled.IsUnknown() {
		resp.Diagnostics.AddWarning("Collection Rebuild Requires Opt-In", "The collection changed, but rebuild_on_change is unknown. Apply will refuse the change unless it resolves to true; if enabled, all managed expressions are deleted and recreated with new IDs.")
		return
	}
	if !enabled.ValueBool() {
		resp.Diagnostics.AddError("Collection Rebuild Disabled", "The ordered collection changed or drifted. Set rebuild_on_change = true to explicitly allow deleting all managed entries and recreating them sequentially.")
		return
	}
	resp.Diagnostics.AddWarning("Entire Collection Will Be Rebuilt", "All entries managed by this collection will be deleted before sequential uploads begin. Every Discord ID changes, existing references may become invalid, and rate limits can cause extended unavailability. Unmanaged expressions will not be deleted.")
}

func collectionIDs(ctx context.Context, source collectionReader, diags *diag.Diagnostics) map[string]string {
	var ids types.Map
	diags.Append(source.GetAttribute(ctx, path.Root("ids"), &ids)...)
	result := map[string]string{}
	if !ids.IsNull() && !ids.IsUnknown() {
		diags.Append(ids.ElementsAs(ctx, &result, false)...)
	}
	return result
}

func (r *ExpressionCollectionResource) save(ctx context.Context, state *tfsdk.State, server string, ids map[string]string, hash string, diags *diag.Diagnostics) {
	diags.Append(state.SetAttribute(ctx, path.Root("id"), server+":"+r.kind)...)
	diags.Append(state.SetAttribute(ctx, path.Root("ids"), ids)...)
	diags.Append(state.SetAttribute(ctx, path.Root("fingerprint"), hash)...)
}

func (r *ExpressionCollectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	entries, hash, known := r.entries(ctx, req.Plan, &resp.Diagnostics)
	if !known {
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError("Unknown Collection Entry", "All entry values must be known before uploading the collection.")
		}
		return
	}
	var server, planned types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("server_id"), &server)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("fingerprint"), &planned)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !planned.IsUnknown() && planned.ValueString() != "" && planned.ValueString() != hash {
		resp.Diagnostics.AddError("Collection Artwork Changed After Planning", "Sticker file contents changed after planning. Create a new plan; no expressions were uploaded.")
		return
	}
	resp.State.Raw = req.Plan.Raw
	ids := map[string]string{}
	r.upload(ctx, server.ValueString(), entries, ids, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		hash = ""
	}
	r.save(ctx, &resp.State, server.ValueString(), ids, hash, &resp.Diagnostics)
}

func (r *ExpressionCollectionResource) upload(ctx context.Context, server string, entries []collectionEntry, ids map[string]string, diags *diag.Diagnostics) {
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			diags.AddError("Collection Upload Interrupted", err.Error())
			return
		}
		var id string
		var err error
		if r.kind == "emoji" {
			var emoji *discordgo.Emoji
			emoji, err = r.client.Session.GuildEmojiCreate(server, &discordgo.EmojiParams{Name: entry.Name, Image: entry.Image, Roles: entry.Roles}, discordgo.WithContext(ctx))
			if err == nil && emoji != nil {
				id = emoji.ID
			}
		} else {
			var sticker *discordgo.Sticker
			sticker, err = r.client.Session.GuildStickerCreate(server, entry.Name, entry.Description, entry.Tags, entry.File, entry.fileData, discordgo.WithContext(ctx))
			if err == nil && sticker != nil {
				id = sticker.ID
			}
		}
		if err != nil {
			diags.AddError("Error Uploading Collection Entry", fmt.Sprintf("Entry %q: %s", entry.Key, err))
			return
		}
		if id == "" {
			diags.AddError("Missing Expression ID", fmt.Sprintf("Discord returned no ID for entry %q. No further uploads were attempted.", entry.Key))
			return
		}
		// Record ownership before verifying: a failed GET must never orphan an upload.
		ids[entry.Key] = id
		matches, err := r.matches(ctx, server, id, entry)
		if err != nil || !matches {
			diags.AddError("Error Verifying Collection Entry", fmt.Sprintf("Entry %q (ID %s) could not be verified: matches=%t, error=%v. No further uploads were attempted.", entry.Key, id, matches, err))
			return
		}
	}
	// Reconcile the completed collection as well as each individual upload.
	for _, entry := range entries {
		matches, err := r.matches(ctx, server, ids[entry.Key], entry)
		if err != nil || !matches {
			diags.AddError("Error Reconciling Collection", fmt.Sprintf("Entry %q could not be reconciled: matches=%t, error=%v.", entry.Key, matches, err))
			return
		}
	}
}

func (r *ExpressionCollectionResource) matches(ctx context.Context, server, id string, entry collectionEntry) (bool, error) {
	if r.kind == "emoji" {
		emoji, err := r.client.Session.GuildEmoji(server, id, discordgo.WithContext(ctx))
		if err != nil {
			return false, err
		}
		roles := append([]string{}, emoji.Roles...)
		sort.Strings(roles)
		return emoji.ID == id && emoji.Name == entry.Name && strings.Join(roles, ",") == strings.Join(entry.Roles, ","), nil
	}
	sticker, err := r.client.Session.GuildSticker(server, id, discordgo.WithContext(ctx))
	if err != nil {
		return false, err
	}
	return sticker.ID == id && sticker.Name == entry.Name && sticker.Description == entry.Description && sticker.Tags == entry.Tags, nil
}

func (r *ExpressionCollectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var server types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("server_id"), &server)...)
	ids := collectionIDs(ctx, req.State, &resp.Diagnostics)
	// Refresh must not depend on local sticker files still being present. Read
	// configuration metadata directly and compare only API-observable attributes.
	var items types.List
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(r.kind), &items)...)
	if resp.Diagnostics.HasError() {
		return
	}
	drift := len(ids) != len(items.Elements())
	for _, raw := range items.Elements() {
		attrs := raw.(types.Object).Attributes()
		entry := collectionEntry{Key: attrs["key"].(types.String).ValueString(), Name: attrs["name"].(types.String).ValueString(), Roles: []string{}}
		if r.kind == "emoji" {
			resp.Diagnostics.Append(attrs["role_ids"].(types.Set).ElementsAs(ctx, &entry.Roles, false)...)
			sort.Strings(entry.Roles)
		} else {
			entry.Description = attrs["description"].(types.String).ValueString()
			entry.Tags = attrs["tags"].(types.String).ValueString()
		}
		id, exists := ids[entry.Key]
		if !exists {
			drift = true
			continue
		}
		matches, err := r.matches(ctx, server.ValueString(), id, entry)
		if err != nil {
			if isNotFound(err) {
				delete(ids, entry.Key)
				drift = true
				continue
			}
			resp.Diagnostics.AddError("Error Reading Collection Entry", fmt.Sprintf("Entry %q: %s", entry.Key, err))
			return
		}
		drift = drift || !matches
	}
	if drift {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("fingerprint"), "")...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ids"), ids)...)
	}
}

func (r *ExpressionCollectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	entries, hash, known := r.entries(ctx, req.Plan, &resp.Diagnostics)
	if !known {
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError("Unknown Collection Entry", "All entry values must be known before rebuilding the collection; no expressions were deleted.")
		}
		return
	}
	var server, previous, planned types.String
	var enabled types.Bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("server_id"), &server)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("fingerprint"), &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("fingerprint"), &previous)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("rebuild_on_change"), &enabled)...)
	ids := collectionIDs(ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !planned.IsUnknown() && planned.ValueString() != hash {
		resp.Diagnostics.AddError("Collection Artwork Changed After Planning", "Sticker file contents changed after the plan was created. Create a new plan; no expressions were deleted.")
		return
	}
	if previous.ValueString() != hash {
		if !enabled.ValueBool() {
			resp.Diagnostics.AddError("Collection Rebuild Disabled", "Set rebuild_on_change = true before changing the ordered collection.")
			return
		}
		resp.State = req.State
		r.remove(ctx, server.ValueString(), ids, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			r.save(ctx, &resp.State, server.ValueString(), ids, "", &resp.Diagnostics)
			return
		}
		resp.State.Raw = req.Plan.Raw
		r.upload(ctx, server.ValueString(), entries, ids, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			hash = ""
		}
	} else {
		resp.State.Raw = req.Plan.Raw
	}
	r.save(ctx, &resp.State, server.ValueString(), ids, hash, &resp.Diagnostics)
}

func (r *ExpressionCollectionResource) remove(ctx context.Context, server string, ids map[string]string, diags *diag.Diagnostics) {
	keys := make([]string, 0, len(ids))
	for key := range ids {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			diags.AddError("Collection Deletion Interrupted", err.Error())
			return
		}
		var err error
		if r.kind == "emoji" {
			err = r.client.Session.GuildEmojiDelete(server, ids[key], discordgo.WithContext(ctx))
		} else {
			err = r.client.Session.GuildStickerDelete(server, ids[key], discordgo.WithContext(ctx))
		}
		if err != nil && !isNotFound(err) {
			diags.AddError("Error Deleting Collection Entry", fmt.Sprintf("Entry %q: %s", key, err))
			return
		}
		delete(ids, key)
	}
}

func (r *ExpressionCollectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var server types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("server_id"), &server)...)
	ids := collectionIDs(ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State = req.State
	r.remove(ctx, server.ValueString(), ids, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		r.save(ctx, &resp.State, server.ValueString(), ids, "", &resp.Diagnostics)
	}
}

func (r *ExpressionCollectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	server, pairs, ok := strings.Cut(req.ID, ":")
	ids := map[string]string{}
	seen := map[string]bool{}
	if !ok || server == "" || pairs == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Expected server_id:key=id,key=id with explicit IDs to adopt.")
		return
	}
	for _, pair := range strings.Split(pairs, ",") {
		key, id, valid := strings.Cut(pair, "=")
		if !valid || key == "" || id == "" || strings.ContainsAny(key+id, ":=") || ids[key] != "" || seen[id] {
			resp.Diagnostics.AddError("Invalid Import ID", "Keys and IDs must be nonempty and unique. Expected server_id:key=id,key=id.")
			return
		}
		ids[key], seen[id] = id, true
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), server)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rebuild_on_change"), false)...)
	// Artwork cannot be recovered from Discord. The first configured apply must
	// explicitly rebuild adopted IDs, after individual resources relinquish them.
	r.save(ctx, &resp.State, server, ids, "", &resp.Diagnostics)
}
