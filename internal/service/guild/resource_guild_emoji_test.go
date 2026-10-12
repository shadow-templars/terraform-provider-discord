package guild

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/shadow-templars/terraform-provider-discord/internal/client"
	"github.com/shadow-templars/terraform-provider-discord/internal/discordgox"
)

type emojiTransport func(*http.Request) (*http.Response, error)

func (f emojiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func emojiResource(t *testing.T, handler func(*http.Request) (int, string)) *GuildEmojiResource {
	t.Helper()
	session, err := discordgo.New("Bot test")
	if err != nil {
		t.Fatal(err)
	}
	session.MaxRestRetries = 0
	session.Client = &http.Client{Transport: emojiTransport(func(req *http.Request) (*http.Response, error) {
		status, body := handler(req)
		return &http.Response{
			StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req,
		}, nil
	})}
	return &GuildEmojiResource{client: &client.DiscordClient{Session: discordgox.New(session)}}
}

func emojiSchema() schema.Schema {
	var resp resource.SchemaResponse
	(&GuildEmojiResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp.Schema
}

func emojiModel() GuildEmojiResourceModel {
	return GuildEmojiResourceModel{
		ID: types.StringValue("456"), ServerID: types.StringValue("123"), Name: types.StringValue("star_yes"),
		Image:   types.StringValue("data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("image"))),
		RoleIDs: types.SetNull(types.StringType), Animated: types.BoolValue(false),
	}
}

func emojiState(t *testing.T, model GuildEmojiResourceModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: emojiSchema()}
	assertEmojiDiags(t, state.Set(context.Background(), &model))
	return state
}

func emojiPlan(t *testing.T, model GuildEmojiResourceModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: emojiSchema()}
	assertEmojiDiags(t, plan.Set(context.Background(), &model))
	return plan
}

func emojiRoles(t *testing.T, roles ...string) types.Set {
	t.Helper()
	if roles == nil {
		roles = []string{}
	}
	value, diags := types.SetValueFrom(context.Background(), types.StringType, roles)
	assertEmojiDiags(t, diags)
	return value
}

func assertEmojiDiags(t *testing.T, diags diag.Diagnostics) {
	t.Helper()
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func emojiPayload(t *testing.T, req *http.Request) map[string]json.RawMessage {
	t.Helper()
	var payload map[string]json.RawMessage
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestGuildEmojiCreate(t *testing.T) {
	for _, restricted := range []bool{false, true} {
		t.Run(fmt.Sprint(restricted), func(t *testing.T) {
			model := emojiModel()
			model.ID = types.StringUnknown()
			model.Animated = types.BoolUnknown()
			model.Image = types.StringValue("data:image/gif;base64,R0lGODlh")
			if restricted {
				model.RoleIDs = emojiRoles(t, "789")
			}
			calls := 0
			r := emojiResource(t, func(req *http.Request) (int, string) {
				calls++
				if req.Method != http.MethodPost || !strings.HasSuffix(req.URL.Path, "/guilds/123/emojis") {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				payload := emojiPayload(t, req)
				if string(payload["name"]) != `"star_yes"` || string(payload["image"]) != `"data:image/gif;base64,R0lGODlh"` {
					t.Fatalf("wrong payload: %s", payload)
				}
				if restricted && string(payload["roles"]) != `["789"]` {
					t.Fatalf("missing roles: %s", payload)
				}
				return http.StatusCreated, `{"id":"456","name":"star_yes","animated":true,"roles":[]}`
			})
			var resp resource.CreateResponse
			resp.State.Schema = emojiSchema()
			r.Create(context.Background(), resource.CreateRequest{Plan: emojiPlan(t, model)}, &resp)
			assertEmojiDiags(t, resp.Diagnostics)
			var got GuildEmojiResourceModel
			assertEmojiDiags(t, resp.State.Get(context.Background(), &got))
			if got.ID.ValueString() != "456" || !got.Animated.ValueBool() || !got.Image.Equal(model.Image) || !got.RoleIDs.Equal(model.RoleIDs) || calls != 1 {
				t.Fatalf("incorrect create state: %+v; calls=%d", got, calls)
			}
		})
	}
}

func TestGuildEmojiReadRoleState(t *testing.T) {
	cases := []struct {
		name   string
		before types.Set
		remote string
		want   types.Set
	}{
		{"omitted", types.SetNull(types.StringType), `[]`, types.SetNull(types.StringType)},
		{"explicit empty", emojiRoles(t), `[]`, emojiRoles(t)},
		{"nil API roles", emojiRoles(t), `null`, emojiRoles(t)},
		{"remote added restriction", types.SetNull(types.StringType), `["789"]`, emojiRoles(t, "789")},
		{"remote removed restriction", emojiRoles(t, "789"), `[]`, emojiRoles(t)},
		{"unordered roles", emojiRoles(t, "789", "987"), `["987","789"]`, emojiRoles(t, "789", "987")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			model := emojiModel()
			model.RoleIDs = c.before
			r := emojiResource(t, func(req *http.Request) (int, string) {
				if req.Method != http.MethodGet || !strings.HasSuffix(req.URL.Path, "/guilds/123/emojis/456") {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				return http.StatusOK, `{"id":"456","name":"renamed","animated":true,"roles":` + c.remote + `}`
			})
			var resp resource.ReadResponse
			resp.State = emojiState(t, model)
			r.Read(context.Background(), resource.ReadRequest{State: resp.State}, &resp)
			assertEmojiDiags(t, resp.Diagnostics)
			var got GuildEmojiResourceModel
			assertEmojiDiags(t, resp.State.Get(context.Background(), &got))
			if !got.RoleIDs.Equal(c.want) || !got.Image.Equal(model.Image) || got.Name.ValueString() != "renamed" || !got.Animated.ValueBool() {
				t.Fatalf("incorrect refreshed state: %+v", got)
			}
		})
	}
}

// An omitted or empty role set must send an explicit [] to remove restrictions.
func TestGuildEmojiUpdateClearsRestrictions(t *testing.T) {
	for _, roles := range []types.Set{types.SetNull(types.StringType), emojiRoles(t), emojiRoles(t, "987")} {
		t.Run(roles.String(), func(t *testing.T) {
			before := emojiModel()
			before.RoleIDs = emojiRoles(t, "789")
			plan := before
			plan.Name = types.StringValue("star_renamed")
			plan.RoleIDs = roles
			plan.Animated = types.BoolUnknown()
			r := emojiResource(t, func(req *http.Request) (int, string) {
				if req.Method != http.MethodPatch || !strings.HasSuffix(req.URL.Path, "/guilds/123/emojis/456") {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				payload := emojiPayload(t, req)
				want := `[]`
				if roles.Equal(emojiRoles(t, "987")) {
					want = `["987"]`
				}
				if string(payload["roles"]) != want || string(payload["name"]) != `"star_renamed"` {
					t.Fatalf("incorrect PATCH payload: %s", payload)
				}
				if _, ok := payload["image"]; ok {
					t.Fatal("image must not be sent when updating an emoji")
				}
				return http.StatusOK, `{"id":"456","name":"star_renamed","roles":` + want + `}`
			})
			var resp resource.UpdateResponse
			resp.State.Schema = emojiSchema()
			r.Update(context.Background(), resource.UpdateRequest{Plan: emojiPlan(t, plan), State: emojiState(t, before)}, &resp)
			assertEmojiDiags(t, resp.Diagnostics)
			var got GuildEmojiResourceModel
			assertEmojiDiags(t, resp.State.Get(context.Background(), &got))
			if !got.RoleIDs.Equal(roles) || !got.Image.Equal(before.Image) || got.Animated.IsUnknown() || got.ID.ValueString() != "456" {
				t.Fatalf("incorrect update state: %+v", got)
			}
		})
	}
}

func TestGuildEmojiReadAndDeleteNotFound(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			r := emojiResource(t, func(req *http.Request) (int, string) {
				return status, `{"code":10014,"message":"Unknown Emoji"}`
			})
			state := emojiState(t, emojiModel())
			read := resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, &read)
			if status == http.StatusNotFound {
				assertEmojiDiags(t, read.Diagnostics)
				if !read.State.Raw.IsNull() {
					t.Fatal("missing emoji should be removed from state")
				}
			} else if !read.Diagnostics.HasError() || read.State.Raw.IsNull() {
				t.Fatal("permission errors must preserve state and return diagnostics")
			}
			deleted := resource.DeleteResponse{State: state}
			r.Delete(context.Background(), resource.DeleteRequest{State: state}, &deleted)
			if deleted.Diagnostics.HasError() != (status != http.StatusNotFound) {
				t.Fatalf("unexpected delete diagnostics: %v", deleted.Diagnostics)
			}
		})
	}
}

func TestGuildEmojiDelete(t *testing.T) {
	calls := 0
	r := emojiResource(t, func(req *http.Request) (int, string) {
		calls++
		if req.Method != http.MethodDelete || !strings.HasSuffix(req.URL.Path, "/guilds/123/emojis/456") {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		return http.StatusNoContent, ""
	})
	var resp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: emojiState(t, emojiModel())}, &resp)
	assertEmojiDiags(t, resp.Diagnostics)
	if calls != 1 {
		t.Fatalf("delete calls: %d", calls)
	}
}

func TestGuildEmojiImport(t *testing.T) {
	for _, id := range []string{"123:456", "123", "", ":456", "123:", "123:456:789"} {
		t.Run(id, func(t *testing.T) {
			r := &GuildEmojiResource{}
			resp := resource.ImportStateResponse{State: emojiState(t, GuildEmojiResourceModel{RoleIDs: types.SetNull(types.StringType)})}
			r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)
			if id != "123:456" {
				if !resp.Diagnostics.HasError() {
					t.Fatal("invalid import ID was accepted")
				}
				return
			}
			assertEmojiDiags(t, resp.Diagnostics)
			var got GuildEmojiResourceModel
			assertEmojiDiags(t, resp.State.Get(context.Background(), &got))
			if got.ServerID.ValueString() != "123" || got.ID.ValueString() != "456" || !got.Image.IsNull() {
				t.Fatalf("incorrect imported state: %+v", got)
			}
			reader := emojiResource(t, func(req *http.Request) (int, string) {
				return http.StatusOK, `{"id":"456","name":"star_yes","roles":["789"]}`
			})
			read := resource.ReadResponse{State: resp.State}
			reader.Read(context.Background(), resource.ReadRequest{State: resp.State}, &read)
			assertEmojiDiags(t, read.Diagnostics)
			assertEmojiDiags(t, read.State.Get(context.Background(), &got))
			if !got.Image.IsNull() || got.Name.ValueString() != "star_yes" || !got.RoleIDs.Equal(emojiRoles(t, "789")) {
				t.Fatalf("incorrect import refresh: %+v", got)
			}
		})
	}
}

func TestGuildEmojiValidateConfig(t *testing.T) {
	cases := []struct {
		name    string
		image   types.String
		wantErr bool
	}{
		{"star_yes", emojiModel().Image, false},
		{"x", emojiModel().Image, true},
		{"has space", emojiModel().Image, true},
		{"has-hyphen", emojiModel().Image, true},
		{strings.Repeat("a", 32), emojiModel().Image, false},
		{strings.Repeat("a", 33), emojiModel().Image, true},
		{"star_yes", types.StringValue("invalid"), true},
		{"star_yes", types.StringUnknown(), false},
	}
	for _, c := range cases {
		t.Run(c.name+"/"+c.image.String(), func(t *testing.T) {
			model := emojiModel()
			model.Name = types.StringValue(c.name)
			model.Image = c.image
			state := emojiState(t, model)
			var resp resource.ValidateConfigResponse
			(&GuildEmojiResource{}).ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: state.Schema, Raw: state.Raw}}, &resp)
			if resp.Diagnostics.HasError() != c.wantErr {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
		})
	}
}

func TestValidateEmojiImage(t *testing.T) {
	for _, format := range []string{"png", "jpeg", "gif", "webp", "avif"} {
		if err := validateEmojiImage("data:image/" + format + ";base64,YQ=="); err != nil {
			t.Errorf("%s: %v", format, err)
		}
	}
	for _, invalid := range []string{"", "file.png", "data:image/png,YQ==", "data:image/svg+xml;base64,YQ==", "data:image/png;base64,", "data:image/png;base64,!!", "data:image/png;base64,YR=="} {
		if err := validateEmojiImage(invalid); err == nil {
			t.Errorf("accepted invalid image %q", invalid)
		}
	}
	for _, size := range []int{256 * 1024, 256*1024 + 1, 256*1024 + 4} {
		image := "data:image/png;base64," + base64.StdEncoding.EncodeToString(make([]byte, size))
		if err := validateEmojiImage(image); (err != nil) != (size > 256*1024) {
			t.Errorf("size %d: unexpected error %v", size, err)
		}
	}
}

func TestGuildEmojiReplacement(t *testing.T) {
	s := emojiSchema()
	before := emojiModel()
	state := emojiState(t, before)
	for _, attribute := range []string{"server_id", "image"} {
		for _, changed := range []bool{false, true} {
			value := before.Image
			if attribute == "server_id" {
				value = before.ServerID
			}
			newValue := value
			if changed {
				newValue = types.StringValue(value.ValueString() + "changed")
			}
			modifier := s.Attributes[attribute].(schema.StringAttribute).PlanModifiers[0]
			var resp planmodifier.StringResponse
			modifier.PlanModifyString(context.Background(), planmodifier.StringRequest{
				State: state, Plan: emojiPlan(t, before), StateValue: value, PlanValue: newValue, ConfigValue: newValue,
			}, &resp)
			assertEmojiDiags(t, resp.Diagnostics)
			if resp.RequiresReplace != changed {
				t.Errorf("%s changed=%t: replacement=%t", attribute, changed, resp.RequiresReplace)
			}
		}
	}
}

func TestGuildEmojiAPIErrors(t *testing.T) {
	r := emojiResource(t, func(req *http.Request) (int, string) {
		return http.StatusForbidden, `{"message":"Missing Permissions","code":50013}`
	})
	model := emojiModel()
	create := resource.CreateResponse{State: emojiState(t, model)}
	r.Create(context.Background(), resource.CreateRequest{Plan: emojiPlan(t, model)}, &create)
	if !create.Diagnostics.HasError() {
		t.Fatal("create API error was ignored")
	}
	update := resource.UpdateResponse{State: emojiState(t, model)}
	r.Update(context.Background(), resource.UpdateRequest{Plan: emojiPlan(t, model)}, &update)
	if !update.Diagnostics.HasError() {
		t.Fatal("update API error was ignored")
	}
	bad := model
	bad.Image = types.StringValue("invalid")
	r = emojiResource(t, func(req *http.Request) (int, string) {
		t.Fatal("invalid image reached Discord")
		return 0, ""
	})
	r.Create(context.Background(), resource.CreateRequest{Plan: emojiPlan(t, bad)}, &create)
	if !create.Diagnostics.HasError() {
		t.Fatal("invalid image was accepted during apply")
	}
}

func TestGuildEmojiConfigure(t *testing.T) {
	r := NewGuildEmojiResource().(*GuildEmojiResource)
	var metadata resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "discord"}, &metadata)
	if metadata.TypeName != "discord_guild_emoji" {
		t.Fatalf("incorrect resource name: %s", metadata.TypeName)
	}
	var nilResp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{}, &nilResp)
	assertEmojiDiags(t, nilResp.Diagnostics)
	var badResp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "wrong type"}, &badResp)
	if !badResp.Diagnostics.HasError() {
		t.Fatal("invalid provider data was accepted")
	}
	c := &client.DiscordClient{}
	var goodResp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: c}, &goodResp)
	assertEmojiDiags(t, goodResp.Diagnostics)
	if r.client != c {
		t.Fatal("configured client was not retained")
	}
}
