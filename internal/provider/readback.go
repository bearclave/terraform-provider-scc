package provider

// Read-back normalisation. Some Get-* cmdlets report a setting in a different
// shape than the New-/Set-* parameter that writes it:
//
//   - Get-Label returns content marking, watermark, encryption and container
//     protection as JSON strings in LabelActions, not as the
//     ApplyContentMarking*/ApplyWaterMarking*/Encryption*/SiteAndGroupProtection*
//     parameters New-/Set-Label take.
//   - Get-Label and Get-LabelPolicy return advanced settings as "[key, value]"
//     strings in Settings, not as the AdvancedSettings hashtable.
//
// readbackTransport rewrites those responses before the Admin API client
// decodes them, adding the parameter-named properties (only where the object
// does not already carry them). The generated read, the post-update reflection
// wait and the data sources then all see the values under the names they
// write. Rewriting fails open: any response it cannot parse is passed through
// unchanged.

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// readbackNormalizers maps a cmdlet name (the X-CmdletName request header) to
// the function that normalises each returned object in place.
var readbackNormalizers = map[string]func(map[string]any){
	"Get-Label":       normalizeLabel,
	"New-Label":       normalizeLabel,
	"Set-Label":       normalizeLabel,
	"Get-LabelPolicy": normalizeLabelPolicy,
	"New-LabelPolicy": normalizeLabelPolicy,
	"Set-LabelPolicy": normalizeLabelPolicy,
}

type readbackTransport struct{ next http.RoundTripper }

func newReadbackTransport(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &readbackTransport{next: next}
}

// readbackParams adds parameters to a Get-* of one object (a request with
// -Identity) whose read-back is incomplete without them: Get-RetentionCompliancePolicy
// returns empty locations unless asked for -DistributionDetail.
var readbackParams = map[string]map[string]any{
	"Get-RetentionCompliancePolicy": {"DistributionDetail": true},
}

func (t *readbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if extra := readbackParams[req.Header.Get("X-CmdletName")]; extra != nil && req.Body != nil {
		if err := addParams(req, extra); err != nil {
			return nil, err
		}
	}
	resp, err := t.next.RoundTrip(req)
	if err != nil || resp == nil {
		return resp, err
	}
	norm := readbackNormalizers[req.Header.Get("X-CmdletName")]
	if norm == nil || resp.StatusCode != http.StatusOK {
		return resp, nil
	}
	raw, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	body, encoded := decodeBody(raw, resp.Header.Get("Content-Encoding"))
	if out, ok := normalizeEnvelope(body, norm); ok {
		body, encoded = out, false
	}
	if encoded {
		body = raw // could not decode the encoding: hand back the original bytes
	} else {
		resp.Header.Del("Content-Encoding")
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	return resp, nil
}

// decodeBody undoes gzip/deflate the way adminapi's readBody does (sniffing,
// because some responses advertise gzip and aren't). encoded reports that the
// returned bytes are still encoded (e.g. brotli).
func decodeBody(raw []byte, enc string) (body []byte, encoded bool) {
	switch strings.ToLower(enc) {
	case "":
		return raw, false
	case "gzip":
		if len(raw) < 2 || raw[0] != 0x1f || raw[1] != 0x8b {
			return raw, false
		}
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return raw, true
		}
		out, err := io.ReadAll(zr)
		if err != nil {
			return raw, true
		}
		return out, false
	case "deflate":
		out, err := io.ReadAll(flate.NewReader(bytes.NewReader(raw)))
		if err != nil {
			return raw, true
		}
		return out, false
	}
	return raw, true
}

// normalizeEnvelope applies norm to every object in an OData envelope's value
// array, keeping all other envelope fields.
func normalizeEnvelope(body []byte, norm func(map[string]any)) ([]byte, bool) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var env map[string]any
	if dec.Decode(&env) != nil {
		return nil, false
	}
	vals, ok := env["value"].([]any)
	if !ok {
		return nil, false
	}
	for _, v := range vals {
		if obj, ok := v.(map[string]any); ok {
			norm(obj)
		}
	}
	out, err := json.Marshal(env)
	if err != nil {
		return nil, false
	}
	return out, true
}

// ---- Label ----

// labelActionKind is the value type of a flattened label action setting.
type labelActionKind int

const (
	kindString labelActionKind = iota
	kindBool
	kindInt
	kindNotBool // the inverse of the setting ("disabled" -> ...Enabled)
)

type labelActionField struct {
	Param string
	Kind  labelActionKind
	Enum  []string // canonical spellings of an enum value (the read reports lower case)
}

var (
	alignments     = []string{"Left", "Center", "Right"}
	contentMarking = func(prefix string) map[string]labelActionField {
		return map[string]labelActionField{
			"disabled":  {prefix + "Enabled", kindNotBool, nil},
			"text":      {prefix + "Text", kindString, nil},
			"fontsize":  {prefix + "FontSize", kindInt, nil},
			"fontcolor": {prefix + "FontColor", kindString, nil},
			"fontname":  {prefix + "FontName", kindString, nil},
			"margin":    {prefix + "Margin", kindInt, nil},
			"alignment": {prefix + "Alignment", kindString, alignments},
		}
	}
	// labelActionFields maps a label action ("<type>" or "<type>/<subtype>",
	// lower case) and setting key to the New-/Set-Label parameter. Keys seen in
	// published Get-Label output: content marking and watermark (fontsize,
	// placement, text, fontcolor, margin, alignment, layout, disabled) and
	// encryption (disabled, protectiontype, templateid, templatearchived,
	// contentexpiredondateindaysornever, offlineaccessdays,
	// rightsdefinitions). The remaining keys follow the parameter names and are
	// unverified against a tenant; an unknown key is simply not mapped.
	labelActionFields = map[string]map[string]labelActionField{
		"applycontentmarking/header": contentMarking("ApplyContentMarkingHeader"),
		"applycontentmarking/footer": contentMarking("ApplyContentMarkingFooter"),
		"applywatermarking": {
			"disabled":  {"ApplyWaterMarkingEnabled", kindNotBool, nil},
			"text":      {"ApplyWaterMarkingText", kindString, nil},
			"fontsize":  {"ApplyWaterMarkingFontSize", kindInt, nil},
			"fontcolor": {"ApplyWaterMarkingFontColor", kindString, nil},
			"fontname":  {"ApplyWaterMarkingFontName", kindString, nil},
			"layout":    {"ApplyWaterMarkingLayout", kindString, []string{"Horizontal", "Diagonal"}},
		},
		"encrypt": {
			"disabled":                          {"EncryptionEnabled", kindNotBool, nil},
			"protectiontype":                    {"EncryptionProtectionType", kindString, []string{"Template", "RemoveProtection", "UserDefined"}},
			"templateid":                        {"EncryptionTemplateId", kindString, nil},
			"linkedtemplateid":                  {"EncryptionLinkedTemplateId", kindString, nil},
			"aiptemplatescopes":                 {"EncryptionAipTemplateScopes", kindString, nil},
			"contentexpiredondateindaysornever": {"EncryptionContentExpiredOnDateInDaysOrNever", kindString, nil},
			"offlineaccessdays":                 {"EncryptionOfflineAccessDays", kindInt, nil},
			"rightsdefinitions":                 {"EncryptionRightsDefinitions", kindString, nil},
			"promptuser":                        {"EncryptionPromptUser", kindBool, nil},
			"donotforward":                      {"EncryptionDoNotForward", kindBool, nil},
			"encryptonly":                       {"EncryptionEncryptOnly", kindBool, nil},
			"doublekeyencryptionurl":            {"EncryptionDoubleKeyEncryptionUrl", kindString, nil},
			"rightsurl":                         {"EncryptionRightsUrl", kindString, nil},
		},
		"protectgroup": {
			"disabled":                 {"SiteAndGroupProtectionEnabled", kindNotBool, nil},
			"privacy":                  {"SiteAndGroupProtectionPrivacy", kindString, []string{"Public", "Private", "Unspecified"}},
			"allowemailfromguestusers": {"SiteAndGroupProtectionAllowEmailFromGuestUsers", kindBool, nil},
			"allowaccesstoguestusers":  {"SiteAndGroupProtectionAllowAccessToGuestUsers", kindBool, nil},
		},
		"protectsite": {
			"allowfullaccess":            {"SiteAndGroupProtectionAllowFullAccess", kindBool, nil},
			"allowlimitedaccess":         {"SiteAndGroupProtectionAllowLimitedAccess", kindBool, nil},
			"blockaccess":                {"SiteAndGroupProtectionBlockAccess", kindBool, nil},
			"externalsharingcontroltype": {"SiteExternalSharingControlType", kindString, nil},
		},
	}
)

// normalizeLabel flattens LabelActions and Settings into the parameter-named
// properties New-/Set-Label take.
func normalizeLabel(obj map[string]any) {
	for _, a := range labelActions(obj["LabelActions"]) {
		key := strings.ToLower(a.Type)
		if a.SubType != "" {
			key += "/" + strings.ToLower(a.SubType)
		}
		fields := labelActionFields[key]
		for _, s := range a.Settings {
			f, ok := fields[strings.ToLower(s.Key)]
			if !ok {
				continue
			}
			if _, present := obj[f.Param]; present && obj[f.Param] != nil {
				continue // the service already reports it
			}
			if v, ok := actionValue(f, s.Value); ok {
				obj[f.Param] = v
			}
		}
	}
	addAdvancedSettings(obj)
}

func normalizeLabelPolicy(obj map[string]any) { addAdvancedSettings(obj) }

type labelAction struct {
	Type     string
	SubType  string
	Settings []labelActionSetting
}

type labelActionSetting struct {
	Key   string
	Value string
}

// labelActions decodes LabelActions: an array of JSON strings (or, defensively,
// of already-decoded objects), each {"Type":..,"SubType":..,"Settings":[{"Key":..,"Value":..}]}.
func labelActions(v any) []labelAction {
	arr, ok := v.([]any)
	if !ok {
		if s, isStr := v.(string); isStr && s != "" {
			arr = []any{s}
		} else {
			return nil
		}
	}
	var out []labelAction
	for _, e := range arr {
		var m map[string]any
		switch t := e.(type) {
		case string:
			if json.Unmarshal([]byte(t), &m) != nil {
				continue
			}
		case map[string]any:
			m = t
		default:
			continue
		}
		a := labelAction{Type: coerceString(fold(m, "Type")), SubType: coerceString(fold(m, "SubType"))}
		if sets, ok := fold(m, "Settings").([]any); ok {
			for _, se := range sets {
				if sm, ok := se.(map[string]any); ok {
					a.Settings = append(a.Settings, labelActionSetting{Key: coerceString(fold(sm, "Key")), Value: coerceString(fold(sm, "Value"))})
				}
			}
		}
		if a.Type != "" {
			out = append(out, a)
		}
	}
	return out
}

// fold returns m[key] matching the key case-insensitively.
func fold(m map[string]any, key string) any {
	if v, ok := m[key]; ok {
		return v
	}
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return nil
}

func actionValue(f labelActionField, v string) (any, bool) {
	v = strings.TrimSpace(v)
	switch f.Kind {
	case kindBool, kindNotBool:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, false
		}
		if f.Kind == kindNotBool {
			b = !b
		}
		return b, true
	case kindInt:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, false
		}
		return json.Number(strconv.FormatInt(n, 10)), true
	}
	if f.Param == "EncryptionRightsDefinitions" {
		return rightsDefinitionsParam(v), true
	}
	for _, e := range f.Enum {
		if strings.EqualFold(e, v) {
			return e, true
		}
	}
	return v, true
}

// rightsDefinitionsParam converts the read-back rights definitions
// ([{"Identity":"x","Rights":"VIEW,EDIT"}]) to the parameter syntax
// "x:VIEW,EDIT;y:VIEW". A value that is not that JSON is returned unchanged.
func rightsDefinitionsParam(v string) string {
	var defs []map[string]any
	if json.Unmarshal([]byte(v), &defs) != nil {
		return v
	}
	parts := make([]string, 0, len(defs))
	for _, d := range defs {
		id := coerceString(fold(d, "Identity"))
		if id == "" {
			continue
		}
		parts = append(parts, id+":"+strings.ReplaceAll(coerceString(fold(d, "Rights")), " ", ""))
	}
	return strings.Join(parts, ";")
}

// addAdvancedSettings derives AdvancedSettings from Settings when the object
// does not report it: Settings holds "[key, value]" strings ("[color, #40e0d0]",
// "[contenttype, File, Email]").
func addAdvancedSettings(obj map[string]any) {
	if v, ok := obj["AdvancedSettings"]; ok && v != nil {
		return
	}
	adv := parseSettings(obj["Settings"])
	if adv == nil {
		return
	}
	obj["AdvancedSettings"] = adv
}

// parseSettings reads "[key, value]" strings into a map. It returns nil when
// v holds none.
func parseSettings(v any) map[string]any {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	for _, e := range arr {
		s, ok := e.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
			continue
		}
		k, val, found := strings.Cut(s[1:len(s)-1], ",")
		k = strings.TrimSpace(k)
		if !found || k == "" {
			continue
		}
		out[k] = strings.TrimSpace(val)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// addParams merges extra into CmdletInput.Parameters of a request that targets
// one object (has Identity). Any other body is sent unchanged.
func addParams(req *http.Request, extra map[string]any) error {
	raw, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return err
	}
	body := raw
	var env map[string]any
	if json.Unmarshal(raw, &env) == nil {
		if in, ok := env["CmdletInput"].(map[string]any); ok {
			if params, ok := in["Parameters"].(map[string]any); ok && params["Identity"] != nil {
				for k, v := range extra {
					if _, set := params[k]; !set {
						params[k] = v
					}
				}
				if out, err := json.Marshal(env); err == nil {
					body = out
				}
			}
		}
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	return nil
}
