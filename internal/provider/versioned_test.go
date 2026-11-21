package provider

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// v0LabelPolicyState is scc_label_policy state as provider 0.2.0 wrote it:
// lists were strings, add_*/remove_* companions existed.
const v0LabelPolicyState = `{
 "id": "0b1e2c4d-1111-2222-3333-444455556666",
 "identity": "Global sensitivity label policy",
 "name": "Global sensitivity label policy",
 "comment": "",
 "labels": "",
 "exchange_location": "All",
 "share_point_location": "[\"https://contoso.sharepoint.com/sites/a\"]",
 "add_exchange_location": "",
 "retry_distribution": "",
 "settings": ""
}`

func TestUpgradeStateV0(t *testing.T) {
	s := resourceSchema(t, "scc_label_policy")
	out, err := upgradeStateV0([]byte(v0LabelPolicyState), s)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(out, &got)
	want := map[string]any{
		"id":                   "0b1e2c4d-1111-2222-3333-444455556666",
		"name":                 "Global sensitivity label policy",
		"labels":               nil,
		"exchange_location":    []any{"All"},
		"share_point_location": []any{"https://contoso.sharepoint.com/sites/a"},
		"settings":             "",
	}
	for k, w := range want {
		if !reflect.DeepEqual(got[k], w) {
			t.Errorf("%s = %#v, want %#v", k, got[k], w)
		}
	}
	for _, gone := range []string{"add_exchange_location", "retry_distribution"} {
		if _, ok := got[gone]; ok {
			t.Errorf("%s kept", gone)
		}
	}
	// The result decodes against the current schema.
	typ := s.Type().TerraformType(context.Background())
	if _, err := tftypes.ValueFromJSONWithOpts(out, typ, tftypes.ValueFromJSONOpts{}); err != nil {
		t.Errorf("upgraded state does not fit the schema: %v", err)
	}
}

func TestUpgradeStateV0Scalars(t *testing.T) {
	s := resourceSchema(t, "scc_label")
	out, err := upgradeStateV0([]byte(`{"name":"x","priority":"3","encryption_enabled":"true","apply_content_marking_header_enabled":"","conditions":"a"}`), s)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(out, &got)
	if got["priority"] != float64(3) || got["encryption_enabled"] != true || got["apply_content_marking_header_enabled"] != nil {
		t.Errorf("scalars = %v %v %v", got["priority"], got["encryption_enabled"], got["apply_content_marking_header_enabled"])
	}
	if !reflect.DeepEqual(got["conditions"], []any{"a"}) {
		t.Errorf("conditions = %v", got["conditions"])
	}
}

func TestVersionedUpgradeStateHook(t *testing.T) {
	ctx := context.Background()
	v := &versioned{Resource: newLabelPolicyOverride()}
	up := v.UpgradeState(ctx)[0]
	s := resourceSchema(t, "scc_label_policy")
	req := resource.UpgradeStateRequest{RawState: &tfprotov6.RawState{JSON: []byte(v0LabelPolicyState)}}
	resp := resource.UpgradeStateResponse{State: tfsdk.State{Schema: s}}
	up.StateUpgrader(ctx, req, &resp)
	if resp.Diagnostics.HasError() || resp.DynamicValue == nil {
		t.Fatalf("upgrade failed: %v", resp.Diagnostics)
	}
}
