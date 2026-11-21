package provider

// Overrides layer hand-written behaviour on top of a generated resource where
// the cmdlets do not fit the generic New/Get/Set/Remove mapping. The generated
// resource keeps doing the writes; an override only adds:
//
//   - Read keeps the prior state value when the read-back is semantically the
//     same (JSON formatting and key order, case, server-added defaults,
//     identity objects vs the configured address), so a refresh does not turn
//     a cosmetic difference into a plan diff. See semantic.go.
//   - Update applies DeltaLists (lists only New-* takes) through Set-*
//     -Add<X>/-Remove<X> instead of replacing the object.
//   - Create applies SetOnly parameters (only Set-* takes them) right after
//     New-*.
//
// Which attributes are DeltaLists/SetOnly comes from the generator
// (zz_generated_overrides.go); the per-resource bindings are in
// overrides_resources.go.

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/terraprovider/go-exoscc/adminapi"
	"github.com/terraprovider/go-msadmin/consistency"
	"github.com/terraprovider/tf-msadmin/resourcex"
)

// overrideMeta is what the generator emits per overridden noun.
type overrideMeta struct {
	DeltaLists []string // New-only lists changed through Set -Add<X>/-Remove<X>
	SetOnly    []string // parameters only Set-* takes, applied right after New-*
}

// generatedResource is the method set every generated resource has.
type generatedResource interface {
	resource.ResourceWithConfigure
	resource.ResourceWithImportState
}

// override binds a generated resource (model M) to the shared behaviour.
type override[M any] struct {
	generatedResource
	noun string
	// get runs Get-<Noun> -Identity id.
	get func(ctx context.Context, id string) (*adminapi.Result, error)
	// set runs Set-<Noun> -Identity id with the fields fill assigns on the
	// params struct (passed as a pointer).
	set func(ctx context.Context, id string, fill func(params any) error) error
	// The generated pieces.
	read       func(context.Context, map[string]any, *M)
	identityOf func(M) string
	refresh    func(ctx context.Context, id string, m *M, diags *diag.Diagnostics, reflected func(map[string]any) bool) bool
	reconcile  func(cfg, read *M)
	// semantic overrides the default comparison (see keepEquivalent) per API field.
	semantic map[string]equalFunc
}

func (o *override[M]) meta() overrideMeta { return generatedOverrides[o.noun] }

// Schema documents the override behaviour on the generated schema.
func (o *override[M]) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	o.generatedResource.Schema(ctx, req, resp)
	var notes []string
	m := o.meta()
	if len(m.DeltaLists) > 0 {
		notes = append(notes, fmt.Sprintf("List attributes that only New-%s takes (%s) are updated in place through Set-%s -Add<X>/-Remove<X>.",
			o.noun, strings.Join(tfsdkNames(o, m.DeltaLists), ", "), o.noun))
	}
	if len(m.SetOnly) > 0 {
		notes = append(notes, fmt.Sprintf("%s can only be set by Set-%s; on create they are applied right after New-%s.",
			strings.Join(tfsdkNames(o, m.SetOnly), ", "), o.noun, o.noun))
	}
	notes = append(notes, "A refresh keeps the configured value when the service returns an equivalent one: "+
		"JSON values compare by content, ignoring formatting, key order and (except advanced_rule) keys the service adds; "+
		"lists and enumerations case-insensitively, locations by any of their identities.")
	resp.Schema.Description += " " + strings.Join(notes, " ")
}

// tfsdkNames maps model field names to Terraform attribute names.
func tfsdkNames[M any](_ *override[M], fields []string) []string {
	var m M
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = tfsdkName(&m, f)
	}
	return out
}

func (o *override[M]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state M
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := o.identityOf(state)
	get := func(ctx context.Context) (map[string]any, bool, error) {
		res, err := o.get(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		obj := firstObject(res.Value)
		return obj, obj != nil, nil
	}
	obj, present, err := resourcex.LoadUntil(ctx, consistency.Config{}, get, nil)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Get-%s failed", o.noun), err.Error())
		return
	}
	if !present {
		resp.State.RemoveResource(ctx)
		return
	}
	prior := state
	o.read(ctx, obj, &state)
	keepEquivalent(ctx, obj, &prior, &state, o.semantic)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (o *override[M]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	o.generatedResource.Create(ctx, req, resp)
	setOnly := o.meta().SetOnly
	if resp.Diagnostics.HasError() || len(setOnly) == 0 {
		return
	}
	var config, created M
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(resp.State.Get(ctx, &created)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var configured []string
	for _, f := range setOnly {
		if v := modelValue(&config, f); !v.IsNull() && !v.IsUnknown() {
			configured = append(configured, f)
		}
	}
	if len(configured) == 0 {
		return
	}
	err := o.set(ctx, o.identityOf(created), func(p any) error {
		for _, f := range configured {
			if err := assignParam(ctx, p, f, modelValue(&config, f)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		// The object exists; failing here would taint (and later replace) it.
		// The next refresh reads the real values and plans the Set again.
		resp.Diagnostics.AddWarning(fmt.Sprintf("Set-%s after New-%s failed", o.noun, o.noun),
			fmt.Sprintf("%s can only be set by Set-%s; it was not applied and will show as a change on the next plan: %s",
				strings.Join(configured, ", "), o.noun, err))
	}
}

func (o *override[M]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := o.identityOf(state)
	deltas := o.meta().DeltaLists

	if changedExcept(&plan, &state, deltas) {
		o.generatedResource.Update(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		// Only delta lists changed: skip the generated (empty) Set-* call.
		cur := state
		if !o.refresh(ctx, id, &cur, &resp.Diagnostics, nil) && resp.Diagnostics.HasError() {
			return
		}
		cfg := plan
		o.reconcile(&cfg, &cur)
		setModelValue(&cur, "ID", modelValue(&state, "ID"))
		resp.Diagnostics.Append(resp.State.Set(ctx, &cur)...)
	}

	var changes []listDelta
	for _, f := range deltas {
		pv, sv := modelValue(&plan, f), modelValue(&state, f)
		ps, ok1 := pv.(types.Set)
		ss, ok2 := sv.(types.Set)
		if !ok1 || !ok2 || ps.IsNull() || ps.IsUnknown() || ps.Equal(ss) {
			continue
		}
		add, remove := setDelta(setStrings(ss), setStrings(ps))
		if len(add) > 0 || len(remove) > 0 {
			changes = append(changes, listDelta{f, add, remove})
		}
	}
	if len(changes) == 0 {
		return
	}
	err := o.set(ctx, id, func(p any) error {
		for _, c := range changes {
			if len(c.add) > 0 {
				if err := assignField(p, "Add"+c.field, c.add); err != nil {
					return err
				}
			}
			if len(c.remove) > 0 {
				if err := assignField(p, "Remove"+c.field, c.remove); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		// Keep the prior values for the lists that were not applied.
		for _, c := range changes {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(tfsdkName(&state, c.field)), modelValue(&state, c.field))...)
		}
		resp.Diagnostics.AddError(fmt.Sprintf("Set-%s failed", o.noun), err.Error())
	}
}

// listDelta is one DeltaLists change: values for -Add<field> and -Remove<field>.
type listDelta struct {
	field       string
	add, remove []string
}

// ---- reflection over generated models and params ----

// modelValue returns field name of the model *M as an attr.Value.
func modelValue(m any, name string) attr.Value {
	f := reflect.ValueOf(m).Elem().FieldByName(name)
	if !f.IsValid() {
		panic(fmt.Sprintf("override: model %T has no field %s", m, name))
	}
	return f.Interface().(attr.Value)
}

func setModelValue(m any, name string, v attr.Value) {
	reflect.ValueOf(m).Elem().FieldByName(name).Set(reflect.ValueOf(v))
}

// tfsdkName returns the Terraform attribute name of model field name.
func tfsdkName(m any, name string) string {
	f, ok := reflect.TypeOf(m).Elem().FieldByName(name)
	if !ok {
		panic(fmt.Sprintf("override: model %T has no field %s", m, name))
	}
	return f.Tag.Get("tfsdk")
}

// changedExcept reports whether any model field other than skip differs
// between plan and state.
func changedExcept(plan, state any, skip []string) bool {
	sk := map[string]bool{}
	for _, s := range skip {
		sk[s] = true
	}
	pv, sv := reflect.ValueOf(plan).Elem(), reflect.ValueOf(state).Elem()
	t := pv.Type()
	for i := 0; i < t.NumField(); i++ {
		if sk[t.Field(i).Name] {
			continue
		}
		a, ok1 := pv.Field(i).Interface().(attr.Value)
		b, ok2 := sv.Field(i).Interface().(attr.Value)
		if !ok1 || !ok2 || !a.Equal(b) {
			return true
		}
	}
	return false
}

// assignParam writes a model value into params field name, converting to the
// binding's field type ([]string, *bool, *int64, string or any).
func assignParam(ctx context.Context, p any, name string, v attr.Value) error {
	switch t := v.(type) {
	case types.String:
		f, err := paramField(p, name)
		if err != nil {
			return err
		}
		if f.Kind() == reflect.Interface {
			if t.ValueString() == "" {
				return nil
			}
			f.Set(reflect.ValueOf(objectParam(t.ValueString())))
			return nil
		}
		return assignField(p, name, t.ValueString())
	case types.Bool:
		return assignField(p, name, t.ValueBoolPointer())
	case types.Int64:
		return assignField(p, name, t.ValueInt64Pointer())
	case types.Set:
		return assignField(p, name, setStrings(t))
	}
	return fmt.Errorf("override: unsupported value %T for %s", v, name)
}

func paramField(p any, name string) (reflect.Value, error) {
	f := reflect.ValueOf(p).Elem().FieldByName(name)
	if !f.IsValid() || !f.CanSet() {
		return reflect.Value{}, fmt.Errorf("override: params %T has no field %s", p, name)
	}
	return f, nil
}

// assignField sets params field name to v, which must be assignable (or the
// field an interface).
func assignField(p any, name string, v any) error {
	f, err := paramField(p, name)
	if err != nil {
		return err
	}
	rv := reflect.ValueOf(v)
	if !rv.Type().AssignableTo(f.Type()) {
		return fmt.Errorf("override: cannot assign %T to %s (%s)", v, name, f.Type())
	}
	f.Set(rv)
	return nil
}

// setStrings returns a known set's string elements (nil for null/unknown).
func setStrings(s types.Set) []string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	out := make([]string, 0, len(s.Elements()))
	for _, e := range s.Elements() {
		if v, ok := e.(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			out = append(out, v.ValueString())
		}
	}
	return out
}

// setDelta returns what to add to and remove from have to get want. Values
// compare case-insensitively (Purview treats identities that way), so a case
// change alone is no delta.
func setDelta(have, want []string) (add, remove []string) {
	in := func(v string, list []string) bool {
		for _, s := range list {
			if strings.EqualFold(s, v) {
				return true
			}
		}
		return false
	}
	for _, w := range want {
		if !in(w, have) && !in(w, add) {
			add = append(add, w)
		}
	}
	for _, h := range have {
		if !in(h, want) && !in(h, remove) {
			remove = append(remove, h)
		}
	}
	return add, remove
}
