package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/terraprovider/go-exoscc/adminapi"
	"github.com/terraprovider/go-exoscc/purview"
	"github.com/terraprovider/terraform-provider-scc/internal/clients"
)

// TestOverridesCoverGeneratedMetadata checks the generator/override contract:
// every overridden noun has an override, and every DeltaLists/SetOnly field
// exists on the model and on the Set-* params with the expected type.
func TestOverridesCoverGeneratedMetadata(t *testing.T) {
	setParams := map[string]any{
		"Label":                     purview.SetLabelParams{},
		"LabelPolicy":               purview.SetLabelPolicyParams{},
		"DlpCompliancePolicy":       purview.SetDlpCompliancePolicyParams{},
		"DlpComplianceRule":         purview.SetDlpComplianceRuleParams{},
		"RetentionCompliancePolicy": purview.SetRetentionCompliancePolicyParams{},
		"RetentionComplianceRule":   purview.SetRetentionComplianceRuleParams{},
	}
	byName := map[string]resource.Resource{}
	for _, c := range generatedResources() {
		byName[resourceTypeName(c())] = c()
	}
	if len(generatedOverrides) != len(resourceOverrides) {
		t.Errorf("%d generated overrides, %d hand-written", len(generatedOverrides), len(resourceOverrides))
	}
	for noun, meta := range generatedOverrides {
		tf := "scc_" + pascalSnake(noun)
		ctor, ok := resourceOverrides[tf]
		if !ok {
			t.Errorf("%s: no override for %s", noun, tf)
			continue
		}
		if _, ok := byName[tf]; !ok {
			t.Errorf("%s: no generated resource %s", noun, tf)
		}
		p := reflect.TypeOf(setParams[noun])
		model := reflect.ValueOf(ctor()).Elem().FieldByName("read").Type().In(2).Elem()
		for _, f := range meta.DeltaLists {
			if _, ok := model.FieldByName(f); !ok {
				t.Errorf("%s: model has no %s", noun, f)
			}
			for _, pf := range []string{"Add" + f, "Remove" + f} {
				sf, ok := p.FieldByName(pf)
				if !ok || sf.Type != reflect.TypeOf([]string{}) {
					t.Errorf("%s: Set params field %s missing or not []string", noun, pf)
				}
			}
		}
		for _, f := range meta.SetOnly {
			if _, ok := model.FieldByName(f); !ok {
				t.Errorf("%s: model has no %s", noun, f)
			}
			if _, ok := p.FieldByName(f); !ok {
				t.Errorf("%s: Set params have no %s", noun, f)
			}
		}
	}
}

func pascalSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

func resourceSchema(t *testing.T, name string) schema.Schema {
	t.Helper()
	for _, c := range withOverrides(generatedResources()) {
		r := c()
		if resourceTypeName(r) != name {
			continue
		}
		var resp resource.SchemaResponse
		r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
		return resp.Schema
	}
	t.Fatalf("no resource %s", name)
	return schema.Schema{}
}

// requiresReplace reports whether the attribute carries a RequiresReplace plan
// modifier.
func requiresReplace(a schema.Attribute) bool {
	var mods []any
	switch t := a.(type) {
	case schema.StringAttribute:
		for _, m := range t.PlanModifiers {
			mods = append(mods, m)
		}
	case schema.SetAttribute:
		for _, m := range t.PlanModifiers {
			mods = append(mods, m)
		}
	case schema.BoolAttribute:
		for _, m := range t.PlanModifiers {
			mods = append(mods, m)
		}
	case schema.Int64Attribute:
		for _, m := range t.PlanModifiers {
			mods = append(mods, m)
		}
	}
	for _, m := range mods {
		if d, ok := m.(interface{ Description(context.Context) string }); ok &&
			strings.Contains(d.Description(context.Background()), "destroy and recreate") {
			return true
		}
	}
	return false
}

func TestSchemaUpdatableInPlace(t *testing.T) {
	inPlace := map[string][]string{
		"scc_label":                       {"display_name", "tooltip", "comment", "content_type", "priority", "advanced_settings", "apply_content_marking_header_text", "encryption_enabled", "encryption_rights_definitions"},
		"scc_label_policy":                {"labels", "exchange_location", "exchange_location_exception", "modern_group_location", "share_point_location", "one_drive_location", "advanced_settings"},
		"scc_dlp_compliance_policy":       {"exchange_location", "share_point_location", "one_drive_location", "teams_location", "endpoint_dlp_location", "mode", "priority"},
		"scc_dlp_compliance_rule":         {"advanced_rule", "content_contains_sensitive_information", "notify_user", "generate_alert", "except_if_recipient_domain_is", "disabled", "block_access"},
		"scc_retention_compliance_policy": {"exchange_location", "share_point_location", "one_drive_location", "modern_group_location", "teams_channel_location", "teams_chat_location", "adaptive_scope_location", "enabled"},
		"scc_retention_compliance_rule":   {"retention_duration", "retention_compliance_action", "apply_compliance_tag", "content_match_query"},
		"scc_protection_alert":            {"notify_user", "severity", "disabled", "filter", "threshold"},
	}
	sets := map[string][]string{
		"scc_label_policy":                {"labels", "exchange_location", "share_point_location"},
		"scc_dlp_compliance_policy":       {"exchange_location", "share_point_location", "teams_location"},
		"scc_dlp_compliance_rule":         {"notify_user", "generate_alert", "except_if_recipient_domain_is"},
		"scc_retention_compliance_policy": {"exchange_location", "share_point_location", "adaptive_scope_location"},
		"scc_protection_alert":            {"notify_user", "operation"},
	}
	for name, attrs := range inPlace {
		s := resourceSchema(t, name)
		if s.Version != schemaVersion {
			t.Errorf("%s: schema version %d", name, s.Version)
		}
		for _, a := range attrs {
			at, ok := s.Attributes[a]
			if !ok {
				t.Errorf("%s: no attribute %s", name, a)
				continue
			}
			if requiresReplace(at) {
				t.Errorf("%s.%s forces replacement", name, a)
			}
		}
		for _, a := range sets[name] {
			if _, ok := s.Attributes[a].(schema.SetAttribute); !ok {
				t.Errorf("%s.%s is not a set", name, a)
			}
		}
		for a := range s.Attributes {
			if (strings.HasPrefix(a, "add_") || strings.HasPrefix(a, "remove_")) && strings.HasSuffix(name, "_policy") {
				t.Errorf("%s exposes %s (covered by the list attribute)", name, a)
			}
		}
	}
	// Required only in one parameter set of New-*: optional.
	rule := resourceSchema(t, "scc_retention_compliance_rule")
	for _, a := range []string{"apply_compliance_tag", "publish_compliance_tag", "retention_duration"} {
		if sa := rule.Attributes[a].(schema.StringAttribute); sa.Required {
			t.Errorf("scc_retention_compliance_rule.%s is required", a)
		}
	}
	if sa := resourceSchema(t, "scc_retention_compliance_policy").Attributes["adaptive_scope_location"].(schema.SetAttribute); sa.Required {
		t.Error("adaptive_scope_location is required")
	}
	// Still create-time only.
	if !requiresReplace(resourceSchema(t, "scc_label").Attributes["name"]) {
		t.Error("scc_label.name should force replacement")
	}
}

func TestSetDelta(t *testing.T) {
	add, remove := setDelta([]string{"All", "a@contoso.com"}, []string{"all", "b@contoso.com"})
	if !reflect.DeepEqual(add, []string{"b@contoso.com"}) || !reflect.DeepEqual(remove, []string{"a@contoso.com"}) {
		t.Errorf("setDelta = %v, %v", add, remove)
	}
	if a, r := setDelta(nil, nil); a != nil || r != nil {
		t.Error("empty delta")
	}
}

func TestAssignParam(t *testing.T) {
	ctx := context.Background()
	var p purview.SetLabelParams
	if err := assignParam(ctx, &p, "Priority", types.Int64Value(4)); err != nil || p.Priority == nil || *p.Priority != 4 {
		t.Errorf("Priority: %v %v", err, p.Priority)
	}
	if err := assignParam(ctx, &p, "NextLabel", types.StringValue("Secret")); err != nil || p.NextLabel != "Secret" {
		t.Errorf("NextLabel (any): %v %v", err, p.NextLabel)
	}
	if err := assignParam(ctx, &p, "AdvancedSettings", types.StringValue(`{"color":"#000000"}`)); err != nil {
		t.Fatal(err)
	} else if m, ok := p.AdvancedSettings.(map[string]any); !ok || m["color"] != "#000000" {
		t.Errorf("AdvancedSettings = %#v", p.AdvancedSettings)
	}
	if err := assignParam(ctx, &p, "Comment", types.StringValue("x")); err != nil || p.Comment != "x" {
		t.Errorf("Comment: %v", err)
	}
	if err := assignParam(ctx, &p, "NoSuchField", types.StringValue("x")); err == nil {
		t.Error("unknown field accepted")
	}
	if err := assignField(&p, "Comment", 3); err == nil {
		t.Error("wrong type accepted")
	}
}

// ---- fake Admin API ----

type call struct {
	Cmdlet string
	Params map[string]any
}

// fakeAdmin serves InvokeCommand from stored objects keyed by noun. Set-*
// merges scalar parameters into the object; New-* stores the object given by
// newObject.
type fakeAdmin struct {
	mu    sync.Mutex
	calls []call
	objs  map[string]map[string]any
}

func (f *fakeAdmin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CmdletInput struct {
			CmdletName string
			Parameters map[string]any
		}
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	f.mu.Lock()
	defer f.mu.Unlock()
	name := body.CmdletInput.CmdletName
	f.calls = append(f.calls, call{name, body.CmdletInput.Parameters})
	verb, noun, _ := strings.Cut(name, "-")
	obj := f.objs[noun]
	var out []any
	switch verb {
	case "Get":
		if obj != nil {
			out = []any{obj}
		}
	case "New":
		out = []any{obj}
	case "Set":
		for k, v := range body.CmdletInput.Parameters {
			if k != "Identity" && !strings.HasPrefix(k, "Add") && !strings.HasPrefix(k, "Remove") {
				obj[k] = v
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"value": out})
}

func (f *fakeAdmin) writes() []call {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []call
	for _, c := range f.calls {
		if !strings.HasPrefix(c.Cmdlet, "Get-") {
			out = append(out, c)
		}
	}
	return out
}

func newFakeClient(t *testing.T, f *fakeAdmin) *clients.Client {
	t.Helper()
	srv := httptest.NewTLSServer(f)
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	admin, err := adminapi.New(adminapi.Options{
		Cloud:      adminapi.Cloud{Name: "SCC", BaseHost: u.Host, Resource: "https://ps.compliance.protection.outlook.com"},
		TenantID:   "00000000-0000-0000-0000-000000000000",
		Tokens:     adminapi.StaticTokenProvider("token"),
		HTTPClient: &http.Client{Transport: newReadbackTransport(srv.Client().Transport)},
		NoRetry:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &clients.Client{Admin: admin, SCC: purview.New(admin)}
}

func fixtureObject(t *testing.T, name string) map[string]any {
	return firstFixtureObject(t, name, nil)
}

// nullSets gives every unset set field of a model a typed null, as the
// framework would.
func nullSets(m any) {
	v := reflect.ValueOf(m).Elem()
	for i := 0; i < v.NumField(); i++ {
		if s, ok := v.Field(i).Interface().(types.Set); ok && s.ElementType(context.Background()) == nil {
			v.Field(i).Set(reflect.ValueOf(types.SetNull(types.StringType)))
		}
	}
}

func configured(t *testing.T, r resource.Resource, c *clients.Client) (resource.Resource, schema.Schema) {
	t.Helper()
	ctx := context.Background()
	var cr resource.ConfigureResponse
	r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: c}, &cr)
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	return r, sr.Schema
}

func TestLabelPolicyUpdateAppliesLabelDeltas(t *testing.T) {
	ctx := context.Background()
	f := &fakeAdmin{objs: map[string]map[string]any{"LabelPolicy": fixtureObject(t, "get-label-policy.json")}}
	r, s := configured(t, &versioned{Resource: newLabelPolicyOverride()}, newFakeClient(t, f))

	var state labelPolicyModel
	readLabelPolicy(ctx, normalizedCopy(f.objs["LabelPolicy"], normalizeLabelPolicy), &state)
	nullSets(&state)
	plan := state
	plan.Labels = stringSetValue(ctx, []string{"Public", "Confidential", "Highly Confidential"})
	plan.ExchangeLocation = stringSetValue(ctx, []string{"All"})

	req := resource.UpdateRequest{Plan: tfsdk.Plan{Schema: s}, State: tfsdk.State{Schema: s}}
	mustSet(t, req.Plan.Set(ctx, &plan))
	mustSet(t, req.State.Set(ctx, &state))
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	w := f.writes()
	if len(w) != 1 || w[0].Cmdlet != "Set-LabelPolicy" {
		t.Fatalf("writes = %v, want one Set-LabelPolicy", w)
	}
	got := w[0].Params
	if !reflect.DeepEqual(got["AddLabels"], []any{"Highly Confidential"}) || !reflect.DeepEqual(got["RemoveLabels"], []any{"General"}) {
		t.Errorf("Set-LabelPolicy params = %v", got)
	}
	if len(got) != 3 { // Identity, AddLabels, RemoveLabels
		t.Errorf("unexpected parameters sent: %v", got)
	}
	var after labelPolicyModel
	mustSet(t, resp.State.Get(ctx, &after))
	if !after.Labels.Equal(plan.Labels) {
		t.Errorf("state labels = %v, want plan", after.Labels)
	}
}

func TestLabelPolicyUpdateOtherFieldsAndLocations(t *testing.T) {
	ctx := context.Background()
	f := &fakeAdmin{objs: map[string]map[string]any{"LabelPolicy": fixtureObject(t, "get-label-policy.json")}}
	r, s := configured(t, &versioned{Resource: newLabelPolicyOverride()}, newFakeClient(t, f))

	var state labelPolicyModel
	readLabelPolicy(ctx, normalizedCopy(f.objs["LabelPolicy"], normalizeLabelPolicy), &state)
	nullSets(&state)
	plan := state
	plan.Comment = types.StringValue("managed by terraform")
	plan.ExchangeLocation = stringSetValue(ctx, []string{"jdoe@contoso.com"})

	req := resource.UpdateRequest{Plan: tfsdk.Plan{Schema: s}, State: tfsdk.State{Schema: s}}
	mustSet(t, req.Plan.Set(ctx, &plan))
	mustSet(t, req.State.Set(ctx, &state))
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	w := f.writes()
	if len(w) != 2 {
		t.Fatalf("writes = %v, want the generated Set and the location delta", w)
	}
	if w[0].Params["Comment"] != "managed by terraform" || w[0].Params["ExchangeLocation"] != nil {
		t.Errorf("first Set = %v", w[0].Params)
	}
	if !reflect.DeepEqual(w[1].Params["AddExchangeLocation"], []any{"jdoe@contoso.com"}) || !reflect.DeepEqual(w[1].Params["RemoveExchangeLocation"], []any{"All"}) {
		t.Errorf("delta Set = %v", w[1].Params)
	}
}

func TestLabelCreateAppliesSetOnlyPriority(t *testing.T) {
	ctx := context.Background()
	f := &fakeAdmin{objs: map[string]map[string]any{"Label": fixtureObject(t, "get-label.json")}}
	r, s := configured(t, &versioned{Resource: newLabelOverride()}, newFakeClient(t, f))

	var plan labelModel
	nullSets(&plan)
	plan.Name = types.StringValue("Confidential")
	plan.DisplayName = types.StringValue("Confidential")
	plan.Tooltip = types.StringValue("Business data that must not be shared outside the company.")
	plan.Priority = types.Int64Value(5)
	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: s}, Config: tfsdk.Config{Schema: s}}
	mustSet(t, req.Plan.Set(ctx, &plan))
	cfg := tfsdk.State{Schema: s}
	mustSet(t, cfg.Set(ctx, &plan))
	req.Config.Raw = cfg.Raw
	resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	w := f.writes()
	if len(w) != 2 || w[0].Cmdlet != "New-Label" || w[1].Cmdlet != "Set-Label" {
		t.Fatalf("writes = %v", w)
	}
	if _, ok := w[0].Params["Priority"]; ok {
		t.Error("New-Label was sent -Priority")
	}
	if w[1].Params["Priority"] != float64(5) || len(w[1].Params) != 2 {
		t.Errorf("Set-Label params = %v", w[1].Params)
	}
	var after labelModel
	mustSet(t, resp.State.Get(ctx, &after))
	if after.ApplyContentMarkingHeaderText.ValueString() != "CONFIDENTIAL" {
		t.Errorf("computed header text = %q, want the flattened read-back", after.ApplyContentMarkingHeaderText.ValueString())
	}
}

func TestDlpRuleReadKeepsConfiguredJSON(t *testing.T) {
	ctx := context.Background()
	f := &fakeAdmin{objs: map[string]map[string]any{"DlpComplianceRule": fixtureObject(t, "get-dlp-compliance-rule.json")}}
	r, s := configured(t, &versioned{Resource: newDlpComplianceRuleOverride()}, newFakeClient(t, f))

	var state dlpComplianceRuleModel
	readDlpComplianceRule(ctx, f.objs["DlpComplianceRule"], &state)
	nullSets(&state)
	state.AdvancedRule = types.StringValue(compactAdvancedRule)
	req := resource.ReadRequest{State: tfsdk.State{Schema: s}}
	mustSet(t, req.State.Set(ctx, &state))
	resp := resource.ReadResponse{State: tfsdk.State{Schema: s}}
	r.Read(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var after dlpComplianceRuleModel
	mustSet(t, resp.State.Get(ctx, &after))
	if after.AdvancedRule.ValueString() != compactAdvancedRule {
		t.Errorf("advanced_rule replaced by the pretty-printed read-back")
	}
	notify := setStrings(after.NotifyUser)
	sort.Strings(notify)
	if !reflect.DeepEqual(notify, []string{"LastModifier", "Owner", "SiteAdmin"}) {
		t.Errorf("notify_user = %v", notify)
	}
}

func normalizedCopy(obj map[string]any, norm func(map[string]any)) map[string]any {
	b, _ := json.Marshal(obj)
	var c map[string]any
	_ = json.Unmarshal(b, &c)
	norm(c)
	return c
}

func mustSet(t *testing.T, d interface{ HasError() bool }) {
	t.Helper()
	if d.HasError() {
		t.Fatal(d)
	}
}
