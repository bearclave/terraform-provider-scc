package provider

// The testdata/get-*.json fixtures are synthetic Admin API responses built
// from the shapes published for these cmdlets (Microsoft Learn, Practical365
// label reports, the LabelActions JSON of New-/Set-Label scripts, and the DLP
// policy location objects captured in terraform-provider-scc PR #4). They are
// not captured from a tenant.

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func loadEnvelope(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func firstFixtureObject(t *testing.T, name string, norm func(map[string]any)) map[string]any {
	t.Helper()
	body := loadEnvelope(t, name)
	if norm != nil {
		out, ok := normalizeEnvelope(body, norm)
		if !ok {
			t.Fatalf("normalizeEnvelope(%s) failed", name)
		}
		body = out
	}
	var env struct {
		Value []map[string]any `json:"value"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Value) != 1 {
		t.Fatalf("%s: %d objects", name, len(env.Value))
	}
	return env.Value[0]
}

func TestNormalizeLabelFlattensLabelActions(t *testing.T) {
	obj := firstFixtureObject(t, "get-label.json", normalizeLabel)
	var m labelModel
	readLabel(context.Background(), obj, &m)

	str := map[string]types.String{
		"apply_content_marking_header_text":       m.ApplyContentMarkingHeaderText,
		"apply_content_marking_header_font_color": m.ApplyContentMarkingHeaderFontColor,
		"apply_content_marking_header_alignment":  m.ApplyContentMarkingHeaderAlignment,
		"apply_content_marking_footer_text":       m.ApplyContentMarkingFooterText,
		"apply_content_marking_footer_alignment":  m.ApplyContentMarkingFooterAlignment,
		"apply_water_marking_text":                m.ApplyWaterMarkingText,
		"apply_water_marking_layout":              m.ApplyWaterMarkingLayout,
		"encryption_protection_type":              m.EncryptionProtectionType,
		"encryption_content_expired":              m.EncryptionContentExpiredOnDateInDaysOrNever,
		"encryption_rights_definitions":           m.EncryptionRightsDefinitions,
		"content_type":                            m.ContentType,
	}
	want := map[string]string{
		"apply_content_marking_header_text":       "CONFIDENTIAL",
		"apply_content_marking_header_font_color": "#000000",
		"apply_content_marking_header_alignment":  "Center",
		"apply_content_marking_footer_text":       "Contoso - Confidential",
		"apply_content_marking_footer_alignment":  "Left",
		"apply_water_marking_text":                "DRAFT",
		"apply_water_marking_layout":              "Diagonal",
		"encryption_protection_type":              "Template", // read back as "template"
		"encryption_content_expired":              "Never",
		"encryption_rights_definitions":           "AllStaff@contoso.com:VIEW,VIEWRIGHTSDATA,DOCEDIT,EDIT,PRINT;contoso.com:VIEW",
		"content_type":                            "File, Email",
	}
	for k, w := range want {
		if got := str[k].ValueString(); got != w {
			t.Errorf("%s = %q, want %q", k, got, w)
		}
	}
	bools := map[string][2]bool{
		"apply_content_marking_header_enabled": {m.ApplyContentMarkingHeaderEnabled.ValueBool(), true},
		"apply_content_marking_footer_enabled": {m.ApplyContentMarkingFooterEnabled.ValueBool(), true},
		"apply_water_marking_enabled":          {m.ApplyWaterMarkingEnabled.ValueBool(), false}, // disabled=true
		"encryption_enabled":                   {m.EncryptionEnabled.ValueBool(), true},
	}
	for k, v := range bools {
		if v[0] != v[1] {
			t.Errorf("%s = %v, want %v", k, v[0], v[1])
		}
	}
	ints := map[string][2]int64{
		"apply_content_marking_header_font_size": {m.ApplyContentMarkingHeaderFontSize.ValueInt64(), 10},
		"apply_content_marking_footer_font_size": {m.ApplyContentMarkingFooterFontSize.ValueInt64(), 8},
		"apply_content_marking_header_margin":    {m.ApplyContentMarkingHeaderMargin.ValueInt64(), 5},
		"encryption_offline_access_days":         {m.EncryptionOfflineAccessDays.ValueInt64(), -1},
		"priority":                               {m.Priority.ValueInt64(), 3},
	}
	for k, v := range ints {
		if v[0] != v[1] {
			t.Errorf("%s = %d, want %d", k, v[0], v[1])
		}
	}
	if !advancedSettingsEqual(`{"color":"#40E0D0"}`, m.AdvancedSettings.ValueString()) {
		t.Errorf("advanced_settings = %s, want color #40e0d0", m.AdvancedSettings.ValueString())
	}
	if n := len(m.LabelActions.Elements()); n != 4 {
		t.Errorf("label_actions has %d elements, want 4", n)
	}
}

func TestNormalizeLabelKeepsReportedProperties(t *testing.T) {
	obj := map[string]any{
		"LabelActions":                  []any{`{"Type":"applycontentmarking","SubType":"header","Settings":[{"Key":"text","Value":"from actions"}]}`},
		"ApplyContentMarkingHeaderText": "reported",
		"AdvancedSettings":              map[string]any{"color": "#000000"},
		"Settings":                      []any{"[color, #ffffff]"},
	}
	normalizeLabel(obj)
	if obj["ApplyContentMarkingHeaderText"] != "reported" {
		t.Errorf("overwrote a reported property: %v", obj["ApplyContentMarkingHeaderText"])
	}
	if adv := obj["AdvancedSettings"].(map[string]any); adv["color"] != "#000000" {
		t.Errorf("overwrote AdvancedSettings: %v", adv)
	}
}

func TestNormalizeLabelIgnoresMalformedActions(t *testing.T) {
	obj := map[string]any{"LabelActions": []any{"not json", 42, `{"Type":"encrypt","Settings":[{"Key":"offlineaccessdays","Value":"x"},{"Key":"unknownkey","Value":"1"}]}`}}
	normalizeLabel(obj)
	if _, ok := obj["EncryptionOfflineAccessDays"]; ok {
		t.Error("mapped an unparsable integer")
	}
	if len(obj) != 1 {
		t.Errorf("added unexpected properties: %v", obj)
	}
}

func TestNormalizeLabelPolicySettings(t *testing.T) {
	obj := firstFixtureObject(t, "get-label-policy.json", normalizeLabelPolicy)
	var m labelPolicyModel
	readLabelPolicy(context.Background(), obj, &m)
	if !advancedSettingsEqual(`{"RequireDowngradeJustification":"True","OutlookDefaultLabel":"None"}`, m.AdvancedSettings.ValueString()) {
		t.Errorf("advanced_settings = %s", m.AdvancedSettings.ValueString())
	}
	if got := setStrings(m.ExchangeLocation); len(got) != 1 || got[0] != "All" {
		t.Errorf("exchange_location = %v, want [All]", got)
	}
	if got := setStrings(m.Labels); len(got) != 3 {
		t.Errorf("labels = %v, want 3 labels", got)
	}
	if !m.SharePointLocation.IsNull() && len(m.SharePointLocation.Elements()) != 0 {
		t.Errorf("share_point_location = %v, want empty", m.SharePointLocation)
	}
}

func TestParseSettings(t *testing.T) {
	got := parseSettings([]any{"[color, #40e0d0]", "[contenttype, File, Email]", "bogus", "[, x]", 3})
	if len(got) != 2 || got["color"] != "#40e0d0" || got["contenttype"] != "File, Email" {
		t.Errorf("parseSettings = %v", got)
	}
	if parseSettings([]any{}) != nil || parseSettings("x") != nil {
		t.Error("want nil for no settings")
	}
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestReadbackTransportRewritesGzipResponse(t *testing.T) {
	body := loadEnvelope(t, "get-label.json")
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, _ = zw.Write(body)
	_ = zw.Close()

	next := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		h := http.Header{}
		h.Set("Content-Encoding", "gzip")
		return &http.Response{StatusCode: 200, Header: h, Body: io.NopCloser(bytes.NewReader(gz.Bytes())), Request: r}, nil
	})
	tr := newReadbackTransport(next)

	req, _ := http.NewRequest(http.MethodPost, "https://example.invalid/adminapi/beta/t/InvokeCommand", nil)
	req.Header.Set("X-CmdletName", "Get-Label")
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get("Content-Encoding") != "" {
		t.Error("Content-Encoding still set on the rewritten body")
	}
	out, _ := io.ReadAll(resp.Body)
	var env struct {
		Context string           `json:"@odata.context"`
		Value   []map[string]any `json:"value"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatal(err)
	}
	if env.Context == "" {
		t.Error("envelope fields were dropped")
	}
	if env.Value[0]["ApplyContentMarkingHeaderText"] != "CONFIDENTIAL" {
		t.Errorf("not normalised: %v", env.Value[0]["ApplyContentMarkingHeaderText"])
	}

	// Other cmdlets pass through untouched (still compressed).
	req.Header.Set("X-CmdletName", "Get-DlpComplianceRule")
	resp, _ = tr.RoundTrip(req)
	raw, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(raw, gz.Bytes()) || resp.Header.Get("Content-Encoding") != "gzip" {
		t.Error("an unrelated cmdlet's response was modified")
	}
}

func TestReadbackTransportFailsOpen(t *testing.T) {
	for _, body := range []string{"not json", `{"value":"x"}`, `{"error":{"code":"x"}}`} {
		next := roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader([]byte(body)))}, nil
		})
		req, _ := http.NewRequest(http.MethodPost, "https://example.invalid", nil)
		req.Header.Set("X-CmdletName", "Get-Label")
		resp, err := newReadbackTransport(next).RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		out, _ := io.ReadAll(resp.Body)
		if string(out) != body {
			t.Errorf("body %q rewritten to %q", body, out)
		}
	}
}
