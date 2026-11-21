package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// TestProviderSchemaIsValid serves the provider and fetches every schema, so
// the framework validates the generated + overridden resources (duplicate type
// names, invalid attributes, schema versions).
func TestProviderSchemaIsValid(t *testing.T) {
	srv, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("%s: %s", d.Summary, d.Detail)
		}
	}
	for _, name := range []string{"scc_label", "scc_label_policy", "scc_dlp_compliance_policy", "scc_dlp_compliance_rule", "scc_retention_compliance_policy", "scc_retention_compliance_rule", "scc_protection_alert", "scc_audit_config"} {
		s, ok := resp.ResourceSchemas[name]
		if !ok {
			t.Errorf("no resource %s", name)
			continue
		}
		if s.Version != schemaVersion {
			t.Errorf("%s: version %d", name, s.Version)
		}
	}
}
