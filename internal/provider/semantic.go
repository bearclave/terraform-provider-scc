package provider

// Semantic comparison of a read-back value with the prior state value. When
// they mean the same thing, an override's Read keeps the prior (configured)
// spelling, so formatting the service applies does not become a plan diff:
//
//   - sets: elements match case-insensitively, by any identity key of an
//     identity object (a location read back as {"Name":..,"PrimarySmtpAddress":..}
//     matches the configured address), or as equivalent JSON;
//   - strings holding JSON (hashtable parameters such as
//     ContentContainsSensitiveInformation): the configured JSON is contained
//     in the read-back one (keys case-insensitive, scalars compared as text,
//     arrays order-insensitive), because the service adds defaults
//     (rule package id, max count, ...);
//   - per-field comparators (equalFunc) for the rest: exact JSON (AdvancedRule),
//     advanced settings, rights definitions, case-insensitive enums.
//
// The containment rule means that deleting a key from configured JSON does
// not show as a change while the service keeps it; set it explicitly (to ""
// for advanced settings) to change it.

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// equalFunc reports whether the prior state value and the read-back value are
// equivalent.
type equalFunc func(prior, read string) bool

// keepEquivalent replaces each field of read (a *model) with the prior state
// value when the two are equivalent. obj is the raw read-back object, used for
// the identity keys of set elements. ID and Identity always take the read.
func keepEquivalent(ctx context.Context, obj map[string]any, prior, read any, semantic map[string]equalFunc) {
	pv, rv := reflect.ValueOf(prior).Elem(), reflect.ValueOf(read).Elem()
	t := pv.Type()
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Name
		if name == "ID" || name == "Identity" {
			continue
		}
		switch p := pv.Field(i).Interface().(type) {
		case types.Set:
			r, ok := rv.Field(i).Interface().(types.Set)
			if ok && sameSet(p, r, obj[name]) {
				rv.Field(i).Set(pv.Field(i))
			}
		case types.String:
			r, ok := rv.Field(i).Interface().(types.String)
			if !ok || p.IsNull() || p.IsUnknown() || r.IsNull() || r.IsUnknown() || p.Equal(r) {
				continue
			}
			eq := semantic[name]
			if eq == nil {
				eq = jsonContains
			}
			if eq(p.ValueString(), r.ValueString()) {
				rv.Field(i).Set(pv.Field(i))
			}
		}
	}
}

// sameSet reports whether the prior set and the read-back set hold the same
// elements. raw is the read-back property, whose identity objects carry the
// aliases an element may be configured by.
func sameSet(prior, read types.Set, raw any) bool {
	if prior.IsNull() || prior.IsUnknown() || read.IsNull() || read.IsUnknown() {
		return false
	}
	if prior.Equal(read) {
		return false // nothing to keep
	}
	want := setStrings(prior)
	var have [][]string
	if arr, ok := raw.([]any); ok {
		for _, e := range arr {
			if a := elementAliases(e); len(a) > 0 {
				have = append(have, a)
			}
		}
	} else {
		for _, s := range setStrings(read) {
			have = append(have, []string{s})
		}
	}
	if len(have) != len(want) {
		return false
	}
	return matchAll(len(want), len(have), func(i, j int) bool {
		for _, alias := range have[j] {
			if strings.EqualFold(alias, want[i]) || jsonEqual(want[i], alias) {
				return true
			}
		}
		return false
	})
}

// matchAll reports whether every one of n wanted items matches a distinct one
// of m available items (simple augmenting-path bipartite matching).
func matchAll(n, m int, match func(i, j int) bool) bool {
	owner := make([]int, m)
	for j := range owner {
		owner[j] = -1
	}
	var try func(i int, seen []bool) bool
	try = func(i int, seen []bool) bool {
		for j := 0; j < m; j++ {
			if seen[j] || !match(i, j) {
				continue
			}
			seen[j] = true
			if owner[j] < 0 || try(owner[j], seen) {
				owner[j] = i
				return true
			}
		}
		return false
	}
	for i := 0; i < n; i++ {
		if !try(i, make([]bool, m)) {
			return false
		}
	}
	return true
}

func decodeJSON(s string) (any, bool) {
	t := strings.TrimSpace(s)
	if t == "" || (t[0] != '{' && t[0] != '[') {
		return nil, false
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(t)))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil || dec.More() {
		return nil, false
	}
	return v, true
}

// jsonEqual reports whether a and b are the same JSON document regardless of
// formatting and key order (numbers compare by value). Used for AdvancedRule,
// which the service returns pretty-printed.
func jsonEqual(a, b string) bool {
	av, ok1 := decodeJSON(a)
	bv, ok2 := decodeJSON(b)
	return ok1 && ok2 && jsonDeepEqual(av, bv)
}

func jsonDeepEqual(a, b any) bool {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, v := range x {
			w, ok := y[k]
			if !ok || !jsonDeepEqual(v, w) {
				return false
			}
		}
		return true
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !jsonDeepEqual(x[i], y[i]) {
				return false
			}
		}
		return true
	case json.Number:
		y, ok := b.(json.Number)
		if !ok {
			return false
		}
		return numbersEqual(x.String(), y.String())
	default:
		return a == b
	}
}

func numbersEqual(a, b string) bool {
	if a == b {
		return true
	}
	fa, err1 := strconv.ParseFloat(a, 64)
	fb, err2 := strconv.ParseFloat(b, 64)
	return err1 == nil && err2 == nil && fa == fb
}

// jsonContains reports whether the prior JSON is contained in the read-back
// JSON (see the package comment above). Non-JSON values never match.
func jsonContains(prior, read string) bool {
	pv, ok1 := decodeJSON(prior)
	rv, ok2 := decodeJSON(read)
	return ok1 && ok2 && contains(pv, rv)
}

func contains(want, have any) bool {
	switch w := want.(type) {
	case map[string]any:
		h, ok := have.(map[string]any)
		if !ok {
			return false
		}
		for k, v := range w {
			hv := fold(h, k)
			if isEmptyJSON(v) {
				if !isEmptyJSON(hv) {
					return false
				}
				continue
			}
			if !contains(v, hv) {
				return false
			}
		}
		return true
	case []any:
		h, ok := have.([]any)
		if !ok {
			if have == nil {
				return len(w) == 0
			}
			h = []any{have} // a single hashtable may come back unwrapped
		}
		if len(w) != len(h) {
			return false
		}
		return matchAll(len(w), len(h), func(i, j int) bool { return contains(w[i], h[j]) })
	default:
		return scalarEqual(want, have)
	}
}

func isEmptyJSON(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == ""
	}
	return false
}

// scalarEqual compares JSON scalars as text ("1" == 1, "True" == true).
func scalarEqual(a, b any) bool {
	as, bs := scalarText(a), scalarText(b)
	if strings.EqualFold(as, bs) {
		return true
	}
	return numbersEqual(as, bs)
}

func scalarText(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case json.Number:
		return t.String()
	case []any, map[string]any:
		b, _ := json.Marshal(t)
		return string(b)
	}
	return coerceString(v)
}

// foldEqual compares case-insensitively (enums the service reports in another
// case, colours).
func foldEqual(a, b string) bool { return strings.EqualFold(a, b) }

// listFoldEqual compares comma-separated values case- and order-insensitively
// ("File, Email" == "email,file").
func listFoldEqual(a, b string) bool { return normList(a) == normList(b) }

func normList(s string) string {
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = strings.ToLower(strings.TrimSpace(p))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

// advancedSettingsEqual compares configured advanced settings (a JSON object,
// as jsonencode() of the hashtable) with the read-back (derived from Settings,
// see readback.go): every configured key must have the same value, comparing
// keys and values case-insensitively and multi-values order-insensitively; a
// key configured as "" (the documented way to remove a setting) must be
// absent. Settings the configuration does not name are ignored: Settings also
// carries built-in keys (isparent, displayname, ...).
func advancedSettingsEqual(prior, read string) bool {
	pv, ok1 := decodeJSON(prior)
	rv, ok2 := decodeJSON(read)
	pm, ok3 := pv.(map[string]any)
	rm, ok4 := rv.(map[string]any)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return false
	}
	for k, v := range pm {
		want, have := settingText(v), settingText(fold(rm, k))
		if normList(want) != normList(have) {
			return false
		}
	}
	return true
}

func settingText(v any) string {
	if arr, ok := v.([]any); ok {
		parts := make([]string, 0, len(arr))
		for _, e := range arr {
			parts = append(parts, scalarText(e))
		}
		return strings.Join(parts, ",")
	}
	return scalarText(v)
}

// rightsDefinitionsEqual compares EncryptionRightsDefinitions values
// ("id:RIGHT,RIGHT;id:RIGHT" or the read-back JSON form) ignoring case and the
// order of identities and rights.
func rightsDefinitionsEqual(a, b string) bool {
	return reflect.DeepEqual(parseRights(a), parseRights(b))
}

func parseRights(s string) map[string]string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") {
		s = rightsDefinitionsParam(s)
	}
	out := map[string]string{}
	for _, def := range strings.Split(s, ";") {
		id, rights, ok := strings.Cut(def, ":")
		id = strings.ToLower(strings.TrimSpace(id))
		if !ok || id == "" {
			continue
		}
		out[id] = normList(rights)
	}
	return out
}
