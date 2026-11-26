package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/terraprovider/go-exoscc/adminapi"
)

func TestWaitGone(t *testing.T) {
	deleteDelay = time.Millisecond
	notFound := &adminapi.APIError{Status: 404}
	calls := 0
	if err := waitGone(context.Background(), func(context.Context) error {
		calls++
		if calls < 3 {
			return nil
		}
		return notFound
	}); err != nil || calls != 3 {
		t.Fatalf("gone after the third check: err %v, calls %d", err, calls)
	}
	if err := waitGone(context.Background(), func(context.Context) error { return nil }); err == nil {
		t.Fatal("an object that stays must fail the delete")
	}
	other := errors.New("boom")
	if err := waitGone(context.Background(), func(context.Context) error { return other }); !errors.Is(err, other) {
		t.Fatalf("another error is returned: %v", err)
	}
}

func TestProtectionAlertDeleteIsVerified(t *testing.T) {
	found := false
	for _, c := range withOverrides(generatedResources()) {
		v, ok := c().(*versioned)
		if ok && resourceTypeName(v.Resource) == "scc_protection_alert" {
			_, found = v.Resource.(*protectionAlertVerified)
		}
	}
	if !found {
		t.Fatal("scc_protection_alert must use the verified delete")
	}
}
