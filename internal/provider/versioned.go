package provider

// Schema version 1 and the upgrade of state written by provider 0.2.x
// (schema version 0). Regenerating on go-exoscc's typed parameter kinds
// changed attribute types (string -> bool / number / set of strings, e.g. every
// policy *_location, DLP rule notify_user) and dropped some attributes (the
// add_*/remove_* location companions). State from 0.2.x would fail to decode,
// so every resource declares version 1 and converts version-0 state by shape:
// values that fit the new type are converted, the rest become null and are
// filled by the next refresh.

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const schemaVersion = 1

// versioned wraps a resource with schema version 1 and the v0 upgrade.
type versioned struct{ resource.Resource }

var (
	_ resource.ResourceWithConfigure    = &versioned{}
	_ resource.ResourceWithImportState  = &versioned{}
	_ resource.ResourceWithModifyPlan   = &versioned{}
	_ resource.ResourceWithUpgradeState = &versioned{}
)

// withOverrides swaps in the overrides and versions every resource.
func withOverrides(ctors []func() resource.Resource) []func() resource.Resource {
	out := make([]func() resource.Resource, 0, len(ctors))
	for _, c := range ctors {
		ctor := c
		if o, ok := resourceOverrides[resourceTypeName(c())]; ok {
			ctor = o
		}
		out = append(out, func() resource.Resource { return &versioned{Resource: ctor()} })
	}
	return out
}

func resourceTypeName(r resource.Resource) string {
	var resp resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "scc"}, &resp)
	return resp.TypeName
}

func (v *versioned) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	v.Resource.Schema(ctx, req, resp)
	resp.Schema.Version = schemaVersion
}

func (v *versioned) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if c, ok := v.Resource.(resource.ResourceWithConfigure); ok {
		c.Configure(ctx, req, resp)
	}
}

func (v *versioned) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if i, ok := v.Resource.(resource.ResourceWithImportState); ok {
		i.ImportState(ctx, req, resp)
		return
	}
	resp.Diagnostics.AddError("Import not supported", "this resource does not support import")
}

func (v *versioned) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if m, ok := v.Resource.(resource.ResourceWithModifyPlan); ok {
		m.ModifyPlan(ctx, req, resp)
	}
}

func (v *versioned) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
			if req.RawState == nil || req.RawState.JSON == nil {
				resp.Diagnostics.AddError("Unable to upgrade state", "the prior state has no JSON representation")
				return
			}
			var sr resource.SchemaResponse
			v.Resource.Schema(ctx, resource.SchemaRequest{}, &sr)
			out, err := upgradeStateV0(req.RawState.JSON, sr.Schema)
			if err != nil {
				resp.Diagnostics.AddError("Unable to upgrade state", err.Error())
				return
			}
			resp.DynamicValue = &tfprotov6.DynamicValue{JSON: out}
		}},
	}
}

// upgradeStateV0 converts version-0 state JSON to the current schema's shape.
func upgradeStateV0(raw []byte, s schema.Schema) ([]byte, error) {
	var old map[string]any
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&old); err != nil {
		return nil, err
	}
	out := make(map[string]any, len(s.Attributes))
	for name, a := range s.Attributes {
		out[name] = convertV0(old[name], a)
	}
	return json.Marshal(out)
}

func convertV0(v any, a schema.Attribute) any {
	if v == nil {
		return nil
	}
	switch a.(type) {
	case schema.StringAttribute:
		switch t := v.(type) {
		case string:
			return t
		case bool:
			return strconv.FormatBool(t)
		case json.Number:
			return t.String()
		}
	case schema.BoolAttribute:
		switch t := v.(type) {
		case bool:
			return t
		case string:
			if b, err := strconv.ParseBool(strings.TrimSpace(t)); err == nil {
				return b
			}
		}
	case schema.Int64Attribute:
		switch t := v.(type) {
		case json.Number:
			if n, err := t.Int64(); err == nil {
				return n
			}
		case string:
			if n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64); err == nil {
				return n
			}
		}
	case schema.SetAttribute:
		var elems []any
		switch t := v.(type) {
		case []any:
			elems = t
		case string:
			if t == "" {
				return nil
			}
			if strings.HasPrefix(strings.TrimSpace(t), "[") && json.Unmarshal([]byte(t), &elems) == nil {
				break
			}
			elems = []any{t}
		default:
			return nil
		}
		set := []string{}
		seen := map[string]bool{}
		for _, e := range elems {
			if s := elementString(e); s != "" && !seen[s] {
				seen[s] = true
				set = append(set, s)
			}
		}
		return set
	}
	return nil
}
