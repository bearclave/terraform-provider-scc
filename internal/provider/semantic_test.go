package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// compactAdvancedRule is the rule in the fixture as jsonencode() writes it:
// compact, keys sorted.
const compactAdvancedRule = `{"Condition":{"Operator":"And","SubConditions":[{"ConditionName":"ContentContainsSensitiveInformation","Value":[{"groups":[{"Operator":"Or","labels":[],"name":"Default","sensitivetypes":[{"confidencelevel":"High","maxcount":-1,"mincount":1,"name":"Credit Card Number"}]}],"operator":"And"}]}]},"Version":"1.0"}`

func TestJSONEqual(t *testing.T) {
	obj := firstFixtureObject(t, "get-dlp-compliance-rule.json", nil)
	pretty := obj["AdvancedRule"].(string)
	if !jsonEqual(compactAdvancedRule, pretty) {
		t.Error("compact and pretty-printed AdvancedRule differ")
	}
	changed := `{"Condition":{"Operator":"And","SubConditions":[{"ConditionName":"ContentContainsSensitiveInformation","Value":[{"groups":[{"Operator":"Or","labels":[],"name":"Default","sensitivetypes":[{"confidencelevel":"High","maxcount":-1,"mincount":5,"name":"Credit Card Number"}]}],"operator":"And"}]}]},"Version":"1.0"}`
	if jsonEqual(changed, pretty) {
		t.Error("a changed mincount compares equal")
	}
	if !jsonEqual(`{"a":1.0}`, `{"a":1}`) || jsonEqual(`{"a":1}`, `{"a":1,"b":2}`) || jsonEqual("x", "x") {
		t.Error("jsonEqual edge cases")
	}
}

func TestJSONContainsSensitiveInformation(t *testing.T) {
	obj := firstFixtureObject(t, "get-dlp-compliance-rule.json", nil)
	var m dlpComplianceRuleModel
	readDlpComplianceRule(context.Background(), obj, &m)
	read := m.ContentContainsSensitiveInformation.ValueString()

	cases := map[string]bool{
		// jsonencode([{ operator = "And", groups = [...] }]) as an operator writes it.
		`[{"groups":[{"name":"Default","operator":"Or","sensitivetypes":[{"confidencelevel":"High","mincount":"1","name":"Credit Card Number"}]}],"operator":"And"}]`: true,
		// Keys in another case, numbers instead of strings.
		`[{"Groups":[{"Name":"Default","Operator":"or","SensitiveTypes":[{"ConfidenceLevel":"high","MinCount":1,"Name":"credit card number"}]}],"Operator":"and"}]`: true,
		// A different threshold or type is a real change.
		`[{"groups":[{"name":"Default","operator":"Or","sensitivetypes":[{"mincount":"10","name":"Credit Card Number"}]}],"operator":"And"}]`: false,
		`[{"groups":[{"name":"Default","operator":"Or","sensitivetypes":[{"name":"U.S. Social Security Number (SSN)"}]}],"operator":"And"}]`:  false,
		// Two types where the service has one.
		`[{"groups":[{"name":"Default","operator":"Or","sensitivetypes":[{"name":"Credit Card Number"},{"name":"IBAN"}]}],"operator":"And"}]`: false,
	}
	for cfg, want := range cases {
		if got := jsonContains(cfg, read); got != want {
			t.Errorf("jsonContains(%s) = %v, want %v", cfg, got, want)
		}
	}
	// The flat (non-grouped) form, as New-DlpComplianceRule examples use it.
	flat := `[{"name":"Credit Card Number","mincount":"1","maxcount":"-1","id":"x","rulePackId":"y"}]`
	if !jsonContains(`[{"Name":"Credit Card Number","minCount":1}]`, flat) {
		t.Error("flat form with server defaults not contained")
	}
	if !jsonContains(`{"name":"Credit Card Number"}`, `{"name":"Credit Card Number","id":"x"}`) {
		t.Error("single object form")
	}
	if !jsonContains(`[{"name":"Credit Card Number"}]`, `{"name":"Credit Card Number"}`) {
		t.Error("array config against an unwrapped single object")
	}
}

func TestAdvancedSettingsEqual(t *testing.T) {
	read := `{"color":"#40e0d0","contenttype":"File, Email","isparent":"False"}`
	cases := map[string]bool{
		`{"color":"#40e0d0"}`:                          true,
		`{"Color":"#40E0D0"}`:                          true,
		`{"color":"#40e0d0","DefaultSharingScope":""}`: true, // "" = remove, and it is absent
		`{"contenttype":["Email","File"]}`:             true,
		`{"color":"#ff0000"}`:                          false,
		`{"DefaultSharingScope":"Anyone"}`:             false,
		`{"color":""}`:                                 false, // removal not yet applied
	}
	for cfg, want := range cases {
		if got := advancedSettingsEqual(cfg, read); got != want {
			t.Errorf("advancedSettingsEqual(%s) = %v, want %v", cfg, got, want)
		}
	}
	if advancedSettingsEqual("not json", read) {
		t.Error("non-JSON config compares equal")
	}
}

func TestRightsDefinitionsEqual(t *testing.T) {
	read := "AllStaff@contoso.com:VIEW,VIEWRIGHTSDATA,DOCEDIT,EDIT,PRINT;contoso.com:VIEW"
	if !rightsDefinitionsEqual("contoso.com:VIEW;allstaff@contoso.com:EDIT,PRINT,VIEW,DOCEDIT,VIEWRIGHTSDATA", read) {
		t.Error("reordered identities and rights differ")
	}
	if !rightsDefinitionsEqual(`[{"Identity":"contoso.com","Rights":"VIEW"}]`, "contoso.com:VIEW") {
		t.Error("JSON form differs from parameter form")
	}
	if rightsDefinitionsEqual("contoso.com:VIEW,EDIT", "contoso.com:VIEW") {
		t.Error("added right compares equal")
	}
}

func TestListFoldEqual(t *testing.T) {
	if !listFoldEqual("File, Email", "email,file") || listFoldEqual("File", "File, Email") {
		t.Error("listFoldEqual")
	}
}

func TestSameSetMatchesIdentityAliases(t *testing.T) {
	ctx := context.Background()
	raw := []any{
		map[string]any{"Name": "jdoe", "PrimarySmtpAddress": "JDoe@contoso.com", "DisplayName": "Jane Doe"},
		map[string]any{"Name": "https://contoso.sharepoint.com/sites/hr", "ImmutableIdentity": "c0ffee"},
	}
	read := stringSetValue(ctx, getStringSlice(map[string]any{"X": raw}, "X"))
	prior := stringSetValue(ctx, []string{"jdoe@contoso.com", "https://contoso.sharepoint.com/sites/HR"})
	if !sameSet(prior, read, raw) {
		t.Error("configured address/URL not matched to the identity objects")
	}
	if sameSet(stringSetValue(ctx, []string{"jdoe@contoso.com"}), read, raw) {
		t.Error("a missing element compares equal")
	}
	if sameSet(stringSetValue(ctx, []string{"jdoe@contoso.com", "other@contoso.com"}), read, raw) {
		t.Error("a different element compares equal")
	}
	scalars := []any{"SiteAdmin", "LastModifier"}
	if !sameSet(stringSetValue(ctx, []string{"lastmodifier", "siteadmin"}), stringSetValue(ctx, []string{"SiteAdmin", "LastModifier"}), scalars) {
		t.Error("case-only difference")
	}
	if sameSet(types.SetNull(types.StringType), read, raw) {
		t.Error("null prior must not be kept")
	}
}

func TestMatchAllNeedsDistinctPartners(t *testing.T) {
	// Both wanted items match only the first available one.
	if matchAll(2, 2, func(i, j int) bool { return j == 0 }) {
		t.Error("two items matched one partner")
	}
	// Greedy would pair 0-0 and fail 1; augmenting finds 0-1, 1-0.
	if !matchAll(2, 2, func(i, j int) bool { return i == 0 || j == 0 }) {
		t.Error("augmenting path not found")
	}
}

// TestKeepEquivalentDlpRule reads the fixture rule into a state that was
// written from configuration (compact JSON, configured spellings) and checks
// that the refresh keeps every configured value, i.e. the next plan is empty.
func TestKeepEquivalentDlpRule(t *testing.T) {
	ctx := context.Background()
	obj := firstFixtureObject(t, "get-dlp-compliance-rule.json", nil)

	var prior dlpComplianceRuleModel
	readDlpComplianceRule(ctx, obj, &prior) // computed values as read
	prior.AdvancedRule = types.StringValue(compactAdvancedRule)
	prior.ContentContainsSensitiveInformation = types.StringValue(`[{"groups":[{"name":"Default","operator":"Or","sensitivetypes":[{"name":"Credit Card Number","mincount":"1","confidencelevel":"High"}]}],"operator":"And"}]`)
	prior.NotifyUser = stringSetValue(ctx, []string{"owner", "siteadmin", "lastmodifier"})
	prior.ExceptIfRecipientDomainIs = stringSetValue(ctx, []string{"Fabrikam.com", "contoso.com"})
	prior.ReportSeverityLevel = types.StringValue("high")
	want := prior

	read := prior
	readDlpComplianceRule(ctx, obj, &read)
	keepEquivalent(ctx, obj, &prior, &read, dlpRuleSemantic)

	if !read.AdvancedRule.Equal(want.AdvancedRule) {
		t.Errorf("advanced_rule not kept: %s", read.AdvancedRule.ValueString())
	}
	if !read.ContentContainsSensitiveInformation.Equal(want.ContentContainsSensitiveInformation) {
		t.Errorf("content_contains_sensitive_information not kept: %s", read.ContentContainsSensitiveInformation.ValueString())
	}
	if !read.NotifyUser.Equal(want.NotifyUser) || !read.ExceptIfRecipientDomainIs.Equal(want.ExceptIfRecipientDomainIs) {
		t.Errorf("sets not kept: %v %v", read.NotifyUser, read.ExceptIfRecipientDomainIs)
	}
	if !read.ReportSeverityLevel.Equal(want.ReportSeverityLevel) {
		t.Errorf("report_severity_level not kept: %s", read.ReportSeverityLevel)
	}
	if got := setStrings(read.GenerateAlert); len(got) != 1 || got[0] != "dlp-alerts@contoso.com" {
		t.Errorf("generate_alert = %v", got)
	}

	// A real change on the service side wins.
	prior.NotifyUser = stringSetValue(ctx, []string{"owner"})
	read = prior
	readDlpComplianceRule(ctx, obj, &read)
	keepEquivalent(ctx, obj, &prior, &read, dlpRuleSemantic)
	if len(read.NotifyUser.Elements()) != 3 {
		t.Errorf("drift on notify_user hidden: %v", read.NotifyUser)
	}
}

func TestPlanEquivalentAfterImport(t *testing.T) {
	obj := firstFixtureObject(t, "get-dlp-compliance-rule.json", nil)
	var state dlpComplianceRuleModel
	readDlpComplianceRule(context.Background(), obj, &state)

	plan := state
	plan.AdvancedRule = types.StringValue(compactAdvancedRule)
	cmp := plan
	if !planEquivalent(&cmp, &state, dlpRuleSemantic) || changedExcept(&cmp, &state, nil) {
		t.Error("a reformatted AdvancedRule still counts as a change")
	}

	plan.Comment = types.StringValue("changed")
	cmp = plan
	planEquivalent(&cmp, &state, dlpRuleSemantic)
	if !changedExcept(&cmp, &state, nil) {
		t.Error("a changed comment is hidden")
	}

	// Containment does not apply at plan time: a key dropped from the
	// configuration is a change.
	plan = state
	plan.ContentContainsSensitiveInformation = types.StringValue(`[{"operator":"And"}]`)
	cmp = plan
	planEquivalent(&cmp, &state, dlpRuleSemantic)
	if !changedExcept(&cmp, &state, nil) {
		t.Error("a configured subset of the read-back JSON is hidden")
	}
}

func TestLocaleEqual(t *testing.T) {
	want := `{"localeKey":"DisplayName","Settings":[{"Key":"en-us","Value":"Confidential"},{"Key":"de-de","Value":"Vertraulich"}]}`
	read := `{"localeKey":"displayName","Settings":[{"Key":"de-DE","Value":"Vertraulich"},{"Key":"en-US","Value":"Confidential"}]}`
	if !localeEqual(want, read) {
		t.Error("case and order of keys must not matter")
	}
	for name, other := range map[string]string{
		"another localeKey": `{"localeKey":"Tooltip","Settings":[{"Key":"en-us","Value":"Confidential"},{"Key":"de-de","Value":"Vertraulich"}]}`,
		"another value":     `{"localeKey":"displayName","Settings":[{"Key":"en-us","Value":"confidential"},{"Key":"de-de","Value":"Vertraulich"}]}`,
		"a missing locale":  `{"localeKey":"displayName","Settings":[{"Key":"en-us","Value":"Confidential"}]}`,
		"not JSON":          `Confidential`,
		"no localeKey":      `{"Settings":[]}`,
	} {
		if localeEqual(want, other) {
			t.Errorf("%s compares equal", name)
		}
	}
}

// TestLabelLocaleSettingsKeepConfiguration reads the fixture label, whose
// LocaleSettings come back as "displayName", into a state configured with the
// spelling the cmdlet takes and checks that the refresh keeps it.
func TestLabelLocaleSettingsKeepConfiguration(t *testing.T) {
	ctx := context.Background()
	obj := firstFixtureObject(t, "get-label.json", normalizeLabel)
	var read labelModel
	readLabel(ctx, obj, &read)

	prior := read
	prior.LocaleSettings = stringSetValue(ctx, []string{`{"Settings":[{"Value":"Vertraulich","Key":"DE-DE"}],"localeKey":"DisplayName"}`})
	keepEquivalent(ctx, obj, &prior, &read, nil)
	if !read.LocaleSettings.Equal(prior.LocaleSettings) {
		t.Errorf("configured locale not kept: %v", read.LocaleSettings)
	}

	prior.LocaleSettings = stringSetValue(ctx, []string{`{"localeKey":"DisplayName","Settings":[{"Key":"en-us","Value":"Confidential"}]}`})
	read2 := read
	readLabel(ctx, obj, &read2)
	keepEquivalent(ctx, obj, &prior, &read2, nil)
	if read2.LocaleSettings.Equal(prior.LocaleSettings) {
		t.Error("a locale the service does not hold is reported as unchanged")
	}
}
