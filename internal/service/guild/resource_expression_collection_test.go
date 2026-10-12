package guild

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func collectionSchema(kind string) schema.Schema {
	var resp resource.SchemaResponse
	(&ExpressionCollectionResource{kind: kind}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp.Schema
}

func collectionItems(t *testing.T, kind string, names ...string) types.List {
	t.Helper()
	attributes := map[string]attr.Type{"key": types.StringType, "name": types.StringType}
	if kind == "emoji" {
		attributes["image"], attributes["role_ids"] = types.StringType, types.SetType{ElemType: types.StringType}
	} else {
		attributes["file"], attributes["description"], attributes["tags"] = types.StringType, types.StringType, types.StringType
	}
	values := []attr.Value{}
	for _, name := range names {
		fields := map[string]attr.Value{"key": types.StringValue(name), "name": types.StringValue(name)}
		if kind == "emoji" {
			fields["image"], fields["role_ids"] = emojiModel().Image, emojiRoles(t, "789")
		} else {
			filename := filepath.Join(t.TempDir(), name+".png")
			if err := os.WriteFile(filename, []byte("image"), 0600); err != nil {
				t.Fatal(err)
			}
			fields["file"], fields["description"], fields["tags"] = types.StringValue(filename), types.StringNull(), types.StringValue("wave")
		}
		object, diags := types.ObjectValue(attributes, fields)
		assertEmojiDiags(t, diags)
		values = append(values, object)
	}
	list, diags := types.ListValue(types.ObjectType{AttrTypes: attributes}, values)
	assertEmojiDiags(t, diags)
	return list
}

func collectionState(t *testing.T, kind string, items types.List, ids map[string]string, hash string, rebuild bool) tfsdk.State {
	t.Helper()
	s := collectionSchema(kind)
	idValues, diags := types.MapValueFrom(context.Background(), types.StringType, ids)
	assertEmojiDiags(t, diags)
	fields := map[string]attr.Value{
		"id": types.StringValue("123:" + kind), "server_id": types.StringValue("123"),
		"rebuild_on_change": types.BoolValue(rebuild), "ids": idValues, "fingerprint": types.StringValue(hash), kind: items,
	}
	value, diags := types.ObjectValue(s.Type().(types.ObjectType).AttributeTypes(), fields)
	assertEmojiDiags(t, diags)
	raw, err := value.ToTerraformValue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return tfsdk.State{Schema: s, Raw: raw}
}

func collectionPlan(state tfsdk.State) tfsdk.Plan {
	return tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
}

func collectionHash(t *testing.T, r *ExpressionCollectionResource, state tfsdk.State) string {
	t.Helper()
	var resp resource.ValidateConfigResponse
	_, hash, known := r.entries(context.Background(), state, &resp.Diagnostics)
	assertEmojiDiags(t, resp.Diagnostics)
	if !known {
		t.Fatal("fingerprint unexpectedly unknown")
	}
	return hash
}

func TestCollectionSequentialUpload(t *testing.T) {
	for _, kind := range []string{"emoji", "sticker"} {
		t.Run(kind, func(t *testing.T) {
			var calls []string
			uploads := 0
			c := emojiResource(t, func(req *http.Request) (int, string) {
				calls = append(calls, req.Method)
				if req.Method == http.MethodPost {
					uploads++
					if len(calls) != uploads*2-1 {
						t.Fatal("next upload began before verification")
					}
					if kind == "emoji" {
						payload := emojiPayload(t, req)
						if string(payload["roles"]) != `["789"]` {
							t.Fatal("role restrictions lost")
						}
					} else {
						if err := req.ParseMultipartForm(512 * 1024); err != nil {
							t.Fatal(err)
						}
						if req.FormValue("name") != fmt.Sprintf("item%d", uploads) || req.FormValue("tags") != "wave" {
							t.Fatal("invalid sticker payload")
						}
						defer req.MultipartForm.RemoveAll()
					}
				}
				index := uploads
				if req.Method == http.MethodGet {
					var id int
					if _, err := fmt.Sscanf(req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:], "%d", &id); err != nil {
						t.Fatal(err)
					}
					index = id - 100
				}
				return http.StatusOK, fmt.Sprintf(`{"id":"%d","name":"item%d","roles":["789"],"description":"","tags":"wave"}`, 100+index, index)
			})
			r := &ExpressionCollectionResource{kind: kind, client: c.client}
			state := collectionState(t, kind, collectionItems(t, kind, "item1", "item2", "item3"), map[string]string{}, "", true)
			hash := collectionHash(t, r, state)
			assertEmojiDiags(t, state.SetAttribute(context.Background(), path.Root("fingerprint"), hash))
			resp := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}
			r.Create(context.Background(), resource.CreateRequest{Plan: collectionPlan(state)}, &resp)
			assertEmojiDiags(t, resp.Diagnostics)
			ids := collectionIDs(context.Background(), resp.State, &resp.Diagnostics)
			if len(ids) != 3 || ids["item1"] != "101" || ids["item3"] != "103" || strings.Join(calls, ",") != "POST,GET,POST,GET,POST,GET,GET,GET,GET" {
				t.Fatalf("incorrect ordered upload: %v %v", calls, ids)
			}
		})
	}
}

func TestCollectionPartialUploadOwnership(t *testing.T) {
	for _, verificationFailure := range []bool{false, true} {
		t.Run(fmt.Sprint(verificationFailure), func(t *testing.T) {
			uploads := 0
			c := emojiResource(t, func(req *http.Request) (int, string) {
				if req.Method == http.MethodPost {
					uploads++
					if uploads == 2 && !verificationFailure {
						return http.StatusForbidden, `{"message":"Missing Permissions"}`
					}
					return http.StatusOK, fmt.Sprintf(`{"id":"%d","name":"item%d","roles":["789"]}`, 100+uploads, uploads)
				}
				if verificationFailure {
					return http.StatusForbidden, `{"message":"Missing Permissions"}`
				}
				return http.StatusOK, `{"id":"101","name":"item1","roles":["789"]}`
			})
			r := &ExpressionCollectionResource{kind: "emoji", client: c.client}
			state := collectionState(t, "emoji", collectionItems(t, "emoji", "item1", "item2", "item3"), map[string]string{}, "", true)
			resp := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}
			r.Create(context.Background(), resource.CreateRequest{Plan: collectionPlan(state)}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("upload failure ignored")
			}
			var clean resource.ReadResponse
			ids := collectionIDs(context.Background(), resp.State, &clean.Diagnostics)
			assertEmojiDiags(t, clean.Diagnostics)
			if len(ids) != 1 || ids["item1"] != "101" {
				t.Fatalf("lost uploaded ID: %v", ids)
			}
			var hash types.String
			assertEmojiDiags(t, resp.State.GetAttribute(context.Background(), path.Root("fingerprint"), &hash))
			if hash.ValueString() != "" {
				t.Fatal("partial upload recorded as complete")
			}
		})
	}
}

func TestCollectionRebuildDeletesBeforeUploads(t *testing.T) {
	calls := []string{}
	c := emojiResource(t, func(req *http.Request) (int, string) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		if req.Method == http.MethodDelete {
			return http.StatusNoContent, ""
		}
		return http.StatusOK, `{"id":"999","name":"new_name","roles":["789"]}`
	})
	r := &ExpressionCollectionResource{kind: "emoji", client: c.client}
	before := collectionState(t, "emoji", collectionItems(t, "emoji", "old_name"), map[string]string{"old_name": "456", "removed": "789"}, "old", true)
	plan := collectionState(t, "emoji", collectionItems(t, "emoji", "new_name"), map[string]string{}, "", true)
	hash := collectionHash(t, r, plan)
	assertEmojiDiags(t, plan.SetAttribute(context.Background(), path.Root("fingerprint"), hash))
	resp := resource.UpdateResponse{State: before}
	r.Update(context.Background(), resource.UpdateRequest{State: before, Plan: collectionPlan(plan)}, &resp)
	assertEmojiDiags(t, resp.Diagnostics)
	if len(calls) != 5 || !strings.HasPrefix(calls[0], "DELETE ") || !strings.HasPrefix(calls[1], "DELETE ") || !strings.HasPrefix(calls[2], "POST ") {
		t.Fatalf("incorrect deletion barrier: %v", calls)
	}
	ids := collectionIDs(context.Background(), resp.State, &resp.Diagnostics)
	if len(ids) != 1 || ids["new_name"] != "999" {
		t.Fatalf("incorrect rebuilt ownership: %v", ids)
	}
}

func TestCollectionDeletionFailurePreservesRemainingIDs(t *testing.T) {
	c := emojiResource(t, func(req *http.Request) (int, string) {
		if strings.HasSuffix(req.URL.Path, "/111") {
			return http.StatusNotFound, `{"message":"Unknown Emoji"}`
		}
		if !strings.HasSuffix(req.URL.Path, "/222") || req.Method != http.MethodDelete {
			t.Fatal("unexpected request or unowned ID deletion")
		}
		return http.StatusForbidden, `{"message":"Missing Permissions"}`
	})
	r := &ExpressionCollectionResource{kind: "emoji", client: c.client}
	state := collectionState(t, "emoji", collectionItems(t, "emoji", "item1", "item2"), map[string]string{"a": "111", "b": "222"}, "old", true)
	resp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("deletion failure ignored")
	}
	var clean resource.ReadResponse
	ids := collectionIDs(context.Background(), resp.State, &clean.Diagnostics)
	assertEmojiDiags(t, clean.Diagnostics)
	if len(ids) != 1 || ids["b"] != "222" {
		t.Fatalf("lost pending deletion: %v", ids)
	}
}

func TestCollectionPlanOptInAndOrder(t *testing.T) {
	r := &ExpressionCollectionResource{kind: "emoji"}
	before := collectionState(t, "emoji", collectionItems(t, "emoji", "item1", "item2"), map[string]string{"item1": "111", "item2": "222"}, "", true)
	originalHash := collectionHash(t, r, before)
	assertEmojiDiags(t, before.SetAttribute(context.Background(), path.Root("fingerprint"), originalHash))
	for _, rebuild := range []bool{false, true} {
		planned := collectionState(t, "emoji", collectionItems(t, "emoji", "item2", "item1"), map[string]string{}, "", rebuild)
		resp := resource.ModifyPlanResponse{Plan: collectionPlan(planned)}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{State: before, Plan: collectionPlan(planned)}, &resp)
		if resp.Diagnostics.HasError() != !rebuild {
			t.Fatalf("incorrect opt-in diagnostic: %v", resp.Diagnostics)
		}
		if rebuild && resp.Diagnostics.WarningsCount() != 1 {
			t.Fatal("missing destructive rebuild warning")
		}
		var hash types.String
		assertEmojiDiags(t, resp.Plan.GetAttribute(context.Background(), path.Root("fingerprint"), &hash))
		if hash.ValueString() == originalHash {
			t.Fatal("reorder did not change fingerprint")
		}
	}
	resp := resource.ModifyPlanResponse{Plan: collectionPlan(before)}
	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{State: before, Plan: collectionPlan(before)}, &resp)
	assertEmojiDiags(t, resp.Diagnostics)
	if resp.Diagnostics.WarningsCount() != 0 {
		t.Fatal("unchanged collection warned of deletion")
	}
	ids := collectionIDs(context.Background(), resp.Plan, &resp.Diagnostics)
	if len(ids) != 2 {
		t.Fatal("unchanged plan lost IDs")
	}
}

func TestCollectionReadDriftAndPermissionErrors(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusForbidden} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			c := emojiResource(t, func(req *http.Request) (int, string) {
				if status != http.StatusOK {
					return status, `{"message":"error"}`
				}
				return status, `{"id":"111","name":"renamed","roles":[]}`
			})
			r := &ExpressionCollectionResource{kind: "emoji", client: c.client}
			state := collectionState(t, "emoji", collectionItems(t, "emoji", "item1"), map[string]string{"item1": "111"}, "old", true)
			resp := resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() != (status == http.StatusForbidden) {
				t.Fatalf("incorrect read diagnostic: %v", resp.Diagnostics)
			}
			var hash types.String
			assertEmojiDiags(t, resp.State.GetAttribute(context.Background(), path.Root("fingerprint"), &hash))
			if status == http.StatusForbidden && hash.ValueString() != "old" {
				t.Fatal("permission failure changed state")
			}
			if status != http.StatusForbidden && hash.ValueString() != "" {
				t.Fatal("drift not detected")
			}
			var clean resource.ReadResponse
			ids := collectionIDs(context.Background(), resp.State, &clean.Diagnostics)
			if (len(ids) == 0) != (status == http.StatusNotFound) {
				t.Fatal("missing ID refresh incorrect")
			}
		})
	}
}

func TestCollectionStickerContentFingerprint(t *testing.T) {
	r := &ExpressionCollectionResource{kind: "sticker"}
	items := collectionItems(t, "sticker", "item1")
	state := collectionState(t, "sticker", items, map[string]string{}, "", true)
	original := collectionHash(t, r, state)
	filename := items.Elements()[0].(types.Object).Attributes()["file"].(types.String).ValueString()
	if err := os.WriteFile(filename, []byte("different artwork"), 0600); err != nil {
		t.Fatal(err)
	}
	changed := collectionHash(t, r, state)
	if changed == original {
		t.Fatal("same-path artwork change was not detected")
	}
	c := emojiResource(t, func(req *http.Request) (int, string) {
		t.Fatal("artwork changed after plan reached Discord")
		return 0, ""
	})
	r.client = c.client
	assertEmojiDiags(t, state.SetAttribute(context.Background(), path.Root("fingerprint"), original))
	resp := resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{State: state, Plan: collectionPlan(state)}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("changed artwork applied with stale plan")
	}
}

func TestCollectionImport(t *testing.T) {
	r := &ExpressionCollectionResource{kind: "emoji"}
	for _, id := range []string{"123:a=111,b=222", "123", "123:", "123:a=111,a=222", "123:a=111,b=111", "123:a="} {
		state := collectionState(t, "emoji", collectionItems(t, "emoji"), map[string]string{}, "", false)
		resp := resource.ImportStateResponse{State: state}
		r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)
		if resp.Diagnostics.HasError() != (id != "123:a=111,b=222") {
			t.Fatalf("incorrect import diagnostics for %q: %v", id, resp.Diagnostics)
		}
	}
}

func TestCollectionValidationAndUnknownValues(t *testing.T) {
	r := &ExpressionCollectionResource{kind: "emoji"}
	for _, names := range [][]string{{"item1", "item1"}, {"x"}, {"has space"}} {
		state := collectionState(t, "emoji", collectionItems(t, "emoji", names...), map[string]string{}, "", false)
		var resp resource.ValidateConfigResponse
		r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: state.Schema, Raw: state.Raw}}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatalf("invalid entries accepted: %v", names)
		}
	}
	items := collectionItems(t, "emoji", "item1")
	object := items.Elements()[0].(types.Object)
	fields := object.Attributes()
	fields["image"] = types.StringUnknown()
	unknown, diags := types.ObjectValue(object.AttributeTypes(context.Background()), fields)
	assertEmojiDiags(t, diags)
	list, diags := types.ListValue(items.ElementType(context.Background()), []attr.Value{unknown})
	assertEmojiDiags(t, diags)
	state := collectionState(t, "emoji", list, map[string]string{}, "", false)
	var resp resource.ValidateConfigResponse
	_, _, known := r.entries(context.Background(), state, &resp.Diagnostics)
	assertEmojiDiags(t, resp.Diagnostics)
	if known {
		t.Fatal("unknown image was fingerprinted as known")
	}
}

func TestCollectionRoleFingerprintNormalization(t *testing.T) {
	r := &ExpressionCollectionResource{kind: "emoji"}
	original := collectionItems(t, "emoji", "item1")
	object := original.Elements()[0].(types.Object)
	var hashes []string
	for _, roles := range []types.Set{emojiRoles(t, "789", "987"), emojiRoles(t, "987", "789")} {
		fields := object.Attributes()
		fields["role_ids"] = roles
		value, diags := types.ObjectValue(object.AttributeTypes(context.Background()), fields)
		assertEmojiDiags(t, diags)
		list, diags := types.ListValue(original.ElementType(context.Background()), []attr.Value{value})
		assertEmojiDiags(t, diags)
		hashes = append(hashes, collectionHash(t, r, collectionState(t, "emoji", list, map[string]string{}, "", false)))
	}
	if hashes[0] != hashes[1] {
		t.Fatal("unordered roles caused a destructive rebuild")
	}
	hashes = nil
	for _, roles := range []types.Set{types.SetNull(types.StringType), emojiRoles(t)} {
		fields := object.Attributes()
		fields["role_ids"] = roles
		value, diags := types.ObjectValue(object.AttributeTypes(context.Background()), fields)
		assertEmojiDiags(t, diags)
		list, diags := types.ListValue(original.ElementType(context.Background()), []attr.Value{value})
		assertEmojiDiags(t, diags)
		hashes = append(hashes, collectionHash(t, r, collectionState(t, "emoji", list, map[string]string{}, "", false)))
	}
	if hashes[0] != hashes[1] {
		t.Fatal("omitted and empty roles caused a destructive rebuild")
	}
}

func TestCollectionCancellationStopsUploads(t *testing.T) {
	c := emojiResource(t, func(req *http.Request) (int, string) { t.Fatal("canceled operation reached Discord"); return 0, "" })
	r := &ExpressionCollectionResource{kind: "emoji", client: c.client}
	state := collectionState(t, "emoji", collectionItems(t, "emoji", "item1"), map[string]string{}, "", true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resp := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: collectionPlan(state)}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("cancellation ignored")
	}
}

func TestCollectionConfigureAndMetadata(t *testing.T) {
	for _, constructor := range []func() resource.Resource{NewEmojiCollectionResource, NewStickerCollectionResource} {
		r := constructor().(*ExpressionCollectionResource)
		var metadata resource.MetadataResponse
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "discord"}, &metadata)
		if metadata.TypeName != "discord_"+r.kind+"_collection" {
			t.Fatal("incorrect resource name")
		}
		var empty resource.ConfigureResponse
		r.Configure(context.Background(), resource.ConfigureRequest{}, &empty)
		assertEmojiDiags(t, empty.Diagnostics)
		var invalid resource.ConfigureResponse
		r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "wrong"}, &invalid)
		if !invalid.Diagnostics.HasError() {
			t.Fatal("invalid client accepted")
		}
		c := emojiResource(t, func(req *http.Request) (int, string) { return http.StatusOK, "{}" }).client
		var valid resource.ConfigureResponse
		r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: c}, &valid)
		assertEmojiDiags(t, valid.Diagnostics)
		if r.client != c {
			t.Fatal("client not retained")
		}
	}
}

func TestCollectionGuildChangeDoesNotReuseIDs(t *testing.T) {
	r := &ExpressionCollectionResource{kind: "emoji"}
	before := collectionState(t, "emoji", collectionItems(t, "emoji", "item1"), map[string]string{"item1": "111"}, "", true)
	hash := collectionHash(t, r, before)
	assertEmojiDiags(t, before.SetAttribute(context.Background(), path.Root("fingerprint"), hash))
	planned := before
	assertEmojiDiags(t, planned.SetAttribute(context.Background(), path.Root("server_id"), "456"))
	resp := resource.ModifyPlanResponse{Plan: collectionPlan(planned)}
	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{State: before, Plan: collectionPlan(planned)}, &resp)
	assertEmojiDiags(t, resp.Diagnostics)
	var id types.String
	var ids types.Map
	assertEmojiDiags(t, resp.Plan.GetAttribute(context.Background(), path.Root("id"), &id))
	assertEmojiDiags(t, resp.Plan.GetAttribute(context.Background(), path.Root("ids"), &ids))
	if !id.IsUnknown() || !ids.IsUnknown() {
		t.Fatal("new guild inherited old collection IDs")
	}
	if resp.Diagnostics.WarningsCount() != 1 {
		t.Fatal("guild replacement lacked destructive warning")
	}
}

func TestCollectionUpdateRefusesWithoutOptIn(t *testing.T) {
	c := emojiResource(t, func(req *http.Request) (int, string) { t.Fatal("disabled rebuild reached Discord"); return 0, "" })
	r := &ExpressionCollectionResource{kind: "emoji", client: c.client}
	before := collectionState(t, "emoji", collectionItems(t, "emoji", "item1"), map[string]string{"item1": "111"}, "old", false)
	planned := collectionState(t, "emoji", collectionItems(t, "emoji", "item2"), map[string]string{}, "", false)
	hash := collectionHash(t, r, planned)
	assertEmojiDiags(t, planned.SetAttribute(context.Background(), path.Root("fingerprint"), hash))
	resp := resource.UpdateResponse{State: before}
	r.Update(context.Background(), resource.UpdateRequest{State: before, Plan: collectionPlan(planned)}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("disabled rebuild was allowed")
	}
}

func TestCollectionUpdateDeletionFailureStopsUploads(t *testing.T) {
	c := emojiResource(t, func(req *http.Request) (int, string) {
		if req.Method != http.MethodDelete {
			t.Fatal("upload started before all deletions succeeded")
		}
		if strings.HasSuffix(req.URL.Path, "/111") {
			return http.StatusNoContent, ""
		}
		return http.StatusForbidden, `{"message":"Missing Permissions"}`
	})
	r := &ExpressionCollectionResource{kind: "emoji", client: c.client}
	before := collectionState(t, "emoji", collectionItems(t, "emoji", "item1", "item2"), map[string]string{"a": "111", "b": "222"}, "old", true)
	planned := collectionState(t, "emoji", collectionItems(t, "emoji", "item3"), map[string]string{}, "", true)
	hash := collectionHash(t, r, planned)
	assertEmojiDiags(t, planned.SetAttribute(context.Background(), path.Root("fingerprint"), hash))
	resp := resource.UpdateResponse{State: before}
	r.Update(context.Background(), resource.UpdateRequest{State: before, Plan: collectionPlan(planned)}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("deletion failure ignored")
	}
	var clean resource.ReadResponse
	ids := collectionIDs(context.Background(), resp.State, &clean.Diagnostics)
	assertEmojiDiags(t, clean.Diagnostics)
	if len(ids) != 1 || ids["b"] != "222" {
		t.Fatalf("remaining ownership lost: %v", ids)
	}
}

func TestCollectionStickerRefreshWithoutLocalArtwork(t *testing.T) {
	items := collectionItems(t, "sticker", "item1")
	filename := items.Elements()[0].(types.Object).Attributes()["file"].(types.String).ValueString()
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusForbidden} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			c := emojiResource(t, func(req *http.Request) (int, string) {
				if req.Method != http.MethodGet || !strings.HasSuffix(req.URL.Path, "/stickers/111") {
					t.Fatal("incorrect sticker refresh request")
				}
				return status, `{"id":"111","name":"item1","description":"","tags":"wave"}`
			})
			r := &ExpressionCollectionResource{kind: "sticker", client: c.client}
			state := collectionState(t, "sticker", items, map[string]string{"item1": "111"}, "old", true)
			resp := resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() != (status == http.StatusForbidden) {
				t.Fatalf("incorrect refresh diagnostic: %v", resp.Diagnostics)
			}
			var hash types.String
			assertEmojiDiags(t, resp.State.GetAttribute(context.Background(), path.Root("fingerprint"), &hash))
			if status == http.StatusNotFound && hash.ValueString() != "" {
				t.Fatal("missing sticker did not trigger drift")
			}
			if status != http.StatusNotFound && hash.ValueString() != "old" {
				t.Fatal("refresh changed unrelated state")
			}
		})
	}
}

func TestCollectionStickerDeleteAndPreflight(t *testing.T) {
	c := emojiResource(t, func(req *http.Request) (int, string) {
		if req.Method != http.MethodDelete || !strings.HasSuffix(req.URL.Path, "/stickers/111") {
			t.Fatal("incorrect sticker delete request")
		}
		return http.StatusNoContent, ""
	})
	r := &ExpressionCollectionResource{kind: "sticker", client: c.client}
	state := collectionState(t, "sticker", collectionItems(t, "sticker", "item1"), map[string]string{"item1": "111"}, "old", true)
	resp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	assertEmojiDiags(t, resp.Diagnostics)
	for _, contents := range [][]byte{nil, make([]byte, 512*1024+1)} {
		items := collectionItems(t, "sticker", "item1")
		file := items.Elements()[0].(types.Object).Attributes()["file"].(types.String).ValueString()
		if err := os.WriteFile(file, contents, 0600); err != nil {
			t.Fatal(err)
		}
		config := collectionState(t, "sticker", items, map[string]string{}, "", true)
		var validation resource.ValidateConfigResponse
		r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw}}, &validation)
		if !validation.Diagnostics.HasError() {
			t.Fatal("invalid file size accepted")
		}
	}
}

func TestCollectionCompletedReconciliationDetectsMissingUpload(t *testing.T) {
	calls := 0
	c := emojiResource(t, func(req *http.Request) (int, string) {
		calls++
		if calls == 3 {
			return http.StatusNotFound, `{"message":"Unknown Emoji"}`
		}
		return http.StatusOK, `{"id":"111","name":"item1","roles":["789"]}`
	})
	r := &ExpressionCollectionResource{kind: "emoji", client: c.client}
	state := collectionState(t, "emoji", collectionItems(t, "emoji", "item1"), map[string]string{}, "", true)
	resp := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: collectionPlan(state)}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("missing upload was not caught by final reconciliation")
	}
	var clean resource.ReadResponse
	ids := collectionIDs(context.Background(), resp.State, &clean.Diagnostics)
	assertEmojiDiags(t, clean.Diagnostics)
	if ids["item1"] != "111" {
		t.Fatal("failed reconciliation orphaned recorded ID")
	}
}

func TestCollectionToggleDoesNotRebuild(t *testing.T) {
	c := emojiResource(t, func(req *http.Request) (int, string) { t.Fatal("flag toggle reached Discord"); return 0, "" })
	r := &ExpressionCollectionResource{kind: "emoji", client: c.client}
	before := collectionState(t, "emoji", collectionItems(t, "emoji", "item1"), map[string]string{"item1": "111"}, "", false)
	hash := collectionHash(t, r, before)
	assertEmojiDiags(t, before.SetAttribute(context.Background(), path.Root("fingerprint"), hash))
	planned := before
	assertEmojiDiags(t, planned.SetAttribute(context.Background(), path.Root("rebuild_on_change"), true))
	resp := resource.UpdateResponse{State: before}
	r.Update(context.Background(), resource.UpdateRequest{State: before, Plan: collectionPlan(planned)}, &resp)
	assertEmojiDiags(t, resp.Diagnostics)
	ids := collectionIDs(context.Background(), resp.State, &resp.Diagnostics)
	if ids["item1"] != "111" {
		t.Fatal("flag toggle lost existing ID")
	}
}

func TestCollectionInvalidArtworkNeverDeletesExistingEntries(t *testing.T) {
	c := emojiResource(t, func(req *http.Request) (int, string) { t.Fatal("invalid artwork reached Discord"); return 0, "" })
	r := &ExpressionCollectionResource{kind: "sticker", client: c.client}
	items := collectionItems(t, "sticker", "item1")
	state := collectionState(t, "sticker", items, map[string]string{"item1": "111"}, "old", true)
	filename := items.Elements()[0].(types.Object).Attributes()["file"].(types.String).ValueString()
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
	resp := resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{State: state, Plan: collectionPlan(state)}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("missing artwork was ignored")
	}
	var clean resource.ReadResponse
	ids := collectionIDs(context.Background(), resp.State, &clean.Diagnostics)
	assertEmojiDiags(t, clean.Diagnostics)
	if ids["item1"] != "111" {
		t.Fatal("preflight failure changed ownership")
	}
}

func TestCollectionUnknownInputsWarnBeforePossibleRebuild(t *testing.T) {
	r := &ExpressionCollectionResource{kind: "emoji"}
	items := collectionItems(t, "emoji", "item1")
	before := collectionState(t, "emoji", items, map[string]string{"item1": "111"}, "old", true)
	object := items.Elements()[0].(types.Object)
	fields := object.Attributes()
	fields["image"] = types.StringUnknown()
	value, diags := types.ObjectValue(object.AttributeTypes(context.Background()), fields)
	assertEmojiDiags(t, diags)
	unknownItems, diags := types.ListValue(items.ElementType(context.Background()), []attr.Value{value})
	assertEmojiDiags(t, diags)
	planned := collectionState(t, "emoji", unknownItems, map[string]string{"item1": "111"}, "old", true)
	resp := resource.ModifyPlanResponse{Plan: collectionPlan(planned)}
	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{State: before, Plan: collectionPlan(planned)}, &resp)
	assertEmojiDiags(t, resp.Diagnostics)
	if resp.Diagnostics.WarningsCount() != 1 {
		t.Fatal("possible destructive change was not warned")
	}
	var hash types.String
	var ids types.Map
	assertEmojiDiags(t, resp.Plan.GetAttribute(context.Background(), path.Root("fingerprint"), &hash))
	assertEmojiDiags(t, resp.Plan.GetAttribute(context.Background(), path.Root("ids"), &ids))
	if !hash.IsUnknown() || !ids.IsUnknown() {
		t.Fatal("unknown inputs reused known fingerprint or IDs")
	}
	c := emojiResource(t, func(req *http.Request) (int, string) { t.Fatal("unknown values reached Discord"); return 0, "" })
	r.client = c.client
	create := resource.CreateResponse{State: tfsdk.State{Schema: planned.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: collectionPlan(planned)}, &create)
	if !create.Diagnostics.HasError() {
		t.Fatal("unknown create values silently accepted")
	}
	update := resource.UpdateResponse{State: before}
	r.Update(context.Background(), resource.UpdateRequest{State: before, Plan: collectionPlan(planned)}, &update)
	if !update.Diagnostics.HasError() {
		t.Fatal("unknown update values silently accepted")
	}
}
