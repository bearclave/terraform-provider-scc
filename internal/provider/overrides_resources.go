package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/terraprovider/go-exoscc/adminapi"
	"github.com/terraprovider/go-exoscc/purview"
	"github.com/terraprovider/go-msadmin/consistency"
	"github.com/terraprovider/tf-msadmin/resourcex"
)

// resourceOverrides replaces a generated resource (by Terraform type name)
// with its override. Every noun in generatedOverrides must be here
// (TestOverridesCoverGeneratedMetadata).
var resourceOverrides = map[string]func() resource.Resource{
	"scc_label":                       newLabelOverride,
	"scc_label_policy":                newLabelPolicyOverride,
	"scc_dlp_compliance_policy":       newDlpCompliancePolicyOverride,
	"scc_dlp_compliance_rule":         newDlpComplianceRuleOverride,
	"scc_retention_compliance_policy": newRetentionCompliancePolicyOverride,
	"scc_retention_compliance_rule":   newRetentionComplianceRuleOverride,
}

// applySet runs a Set-* call with the generated write retry.
func applySet[P any](ctx context.Context, call func(context.Context, P) (*adminapi.Result, error), p P, fill func(any) error) error {
	if err := fill(&p); err != nil {
		return err
	}
	_, err := resourcex.RetryWriteCall(ctx, consistency.Config{}, call, p, isNotFound)
	return err
}

var (
	labelSemantic = map[string]equalFunc{
		"AdvancedSettings":                            advancedSettingsEqual,
		"EncryptionRightsDefinitions":                 rightsDefinitionsEqual,
		"ContentType":                                 listFoldEqual,
		"EncryptionProtectionType":                    foldEqual,
		"ApplyContentMarkingHeaderAlignment":          foldEqual,
		"ApplyContentMarkingFooterAlignment":          foldEqual,
		"ApplyContentMarkingHeaderFontColor":          foldEqual,
		"ApplyContentMarkingFooterFontColor":          foldEqual,
		"ApplyWaterMarkingFontColor":                  foldEqual,
		"ApplyWaterMarkingLayout":                     foldEqual,
		"SiteAndGroupProtectionPrivacy":               foldEqual,
		"SiteExternalSharingControlType":              foldEqual,
		"EncryptionContentExpiredOnDateInDaysOrNever": foldEqual,
	}
	labelPolicySemantic = map[string]equalFunc{
		"AdvancedSettings": advancedSettingsEqual,
	}
	dlpPolicySemantic = map[string]equalFunc{
		"Mode": foldEqual,
	}
	dlpRuleSemantic = map[string]equalFunc{
		// The service returns the rule pretty-printed with "Version" first; the
		// document itself must match exactly (no containment): it is the
		// complete condition tree.
		"AdvancedRule":        jsonEqual,
		"ReportSeverityLevel": foldEqual,
		"NotifyUserType":      foldEqual,
		"BlockAccessScope":    foldEqual,
	}
	retentionRuleSemantic = map[string]equalFunc{
		"RetentionDuration":            foldEqual,
		"RetentionDurationDisplayHint": foldEqual,
		"RetentionComplianceAction":    foldEqual,
		"ExpirationDateOption":         foldEqual,
	}
)

func newLabelOverride() resource.Resource {
	g := &labelResource{}
	return &override[labelModel]{
		generatedResource: g,
		noun:              "Label",
		get: func(ctx context.Context, id string) (*adminapi.Result, error) {
			return g.client.SCC.GetLabel(ctx, purview.GetLabelParams{Identity: id})
		},
		set: func(ctx context.Context, id string, fill func(any) error) error {
			return applySet(ctx, g.client.SCC.SetLabel, purview.SetLabelParams{Identity: id}, fill)
		},
		read: readLabel, identityOf: g.identityOf, refresh: g.refresh, reconcile: g.reconcileState,
		semantic: labelSemantic,
	}
}

func newLabelPolicyOverride() resource.Resource {
	g := &labelPolicyResource{}
	return &override[labelPolicyModel]{
		generatedResource: g,
		noun:              "LabelPolicy",
		get: func(ctx context.Context, id string) (*adminapi.Result, error) {
			return g.client.SCC.GetLabelPolicy(ctx, purview.GetLabelPolicyParams{Identity: id})
		},
		set: func(ctx context.Context, id string, fill func(any) error) error {
			return applySet(ctx, g.client.SCC.SetLabelPolicy, purview.SetLabelPolicyParams{Identity: id}, fill)
		},
		read: readLabelPolicy, identityOf: g.identityOf, refresh: g.refresh, reconcile: g.reconcileState,
		semantic: labelPolicySemantic,
	}
}

func newDlpCompliancePolicyOverride() resource.Resource {
	g := &dlpCompliancePolicyResource{}
	return &override[dlpCompliancePolicyModel]{
		generatedResource: g,
		noun:              "DlpCompliancePolicy",
		get: func(ctx context.Context, id string) (*adminapi.Result, error) {
			return g.client.SCC.GetDlpCompliancePolicy(ctx, purview.GetDlpCompliancePolicyParams{Identity: id})
		},
		set: func(ctx context.Context, id string, fill func(any) error) error {
			return applySet(ctx, g.client.SCC.SetDlpCompliancePolicy, purview.SetDlpCompliancePolicyParams{Identity: id}, fill)
		},
		read: readDlpCompliancePolicy, identityOf: g.identityOf, refresh: g.refresh, reconcile: g.reconcileState,
		semantic: dlpPolicySemantic,
	}
}

func newDlpComplianceRuleOverride() resource.Resource {
	g := &dlpComplianceRuleResource{}
	return &override[dlpComplianceRuleModel]{
		generatedResource: g,
		noun:              "DlpComplianceRule",
		get: func(ctx context.Context, id string) (*adminapi.Result, error) {
			return g.client.SCC.GetDlpComplianceRule(ctx, purview.GetDlpComplianceRuleParams{Identity: id})
		},
		set: func(ctx context.Context, id string, fill func(any) error) error {
			return applySet(ctx, g.client.SCC.SetDlpComplianceRule, purview.SetDlpComplianceRuleParams{Identity: id}, fill)
		},
		read: readDlpComplianceRule, identityOf: g.identityOf, refresh: g.refresh, reconcile: g.reconcileState,
		semantic: dlpRuleSemantic,
	}
}

func newRetentionCompliancePolicyOverride() resource.Resource {
	g := &retentionCompliancePolicyResource{}
	return &override[retentionCompliancePolicyModel]{
		generatedResource: g,
		noun:              "RetentionCompliancePolicy",
		get: func(ctx context.Context, id string) (*adminapi.Result, error) {
			return g.client.SCC.GetRetentionCompliancePolicy(ctx, purview.GetRetentionCompliancePolicyParams{Identity: id})
		},
		set: func(ctx context.Context, id string, fill func(any) error) error {
			return applySet(ctx, g.client.SCC.SetRetentionCompliancePolicy, purview.SetRetentionCompliancePolicyParams{Identity: id}, fill)
		},
		read: readRetentionCompliancePolicy, identityOf: g.identityOf, refresh: g.refresh, reconcile: g.reconcileState,
	}
}

func newRetentionComplianceRuleOverride() resource.Resource {
	g := &retentionComplianceRuleResource{}
	return &override[retentionComplianceRuleModel]{
		generatedResource: g,
		noun:              "RetentionComplianceRule",
		get: func(ctx context.Context, id string) (*adminapi.Result, error) {
			return g.client.SCC.GetRetentionComplianceRule(ctx, purview.GetRetentionComplianceRuleParams{Identity: id})
		},
		set: func(ctx context.Context, id string, fill func(any) error) error {
			return applySet(ctx, g.client.SCC.SetRetentionComplianceRule, purview.SetRetentionComplianceRuleParams{Identity: id}, fill)
		},
		read: readRetentionComplianceRule, identityOf: g.identityOf, refresh: g.refresh, reconcile: g.reconcileState,
		semantic: retentionRuleSemantic,
	}
}
