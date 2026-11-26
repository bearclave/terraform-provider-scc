package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/terraprovider/go-exoscc/purview"
)

// deleteAttempts and deleteDelay bound the wait until a removed alert policy
// is gone: Remove-ProtectionAlert over the Admin API answered success while
// the policy stayed, so a delete is only done once Get no longer finds it.
var (
	deleteAttempts = 12
	deleteDelay    = 5 * time.Second
)

// verifiedDeletes replaces a generated resource whose Remove-* answered
// success without removing the object (outside the generator contract of
// resourceOverrides).
var verifiedDeletes = map[string]func() resource.Resource{
	"scc_protection_alert": newProtectionAlertVerified,
}

// protectionAlertVerified is scc_protection_alert with a delete that forces
// the removal and checks that the policy is gone.
type protectionAlertVerified struct {
	*protectionAlertResource
}

func newProtectionAlertVerified() resource.Resource {
	return &protectionAlertVerified{protectionAlertResource: &protectionAlertResource{}}
}

func (r *protectionAlertVerified) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state protectionAlertModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := r.identityOf(state)
	force := true
	if _, err := r.client.SCC.RemoveProtectionAlert(ctx, purview.RemoveProtectionAlertParams{Identity: id, ForceDeletion: &force}); err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Remove-ProtectionAlert failed", err.Error())
		return
	}
	if err := waitGone(ctx, func(ctx context.Context) error {
		_, err := r.client.SCC.GetProtectionAlert(ctx, purview.GetProtectionAlertParams{Identity: id})
		return err
	}); err != nil {
		resp.Diagnostics.AddError("Remove-ProtectionAlert did not remove the alert policy", fmt.Sprintf("%s: %s", id, err))
	}
}

// waitGone calls get until it reports the object as not found.
func waitGone(ctx context.Context, get func(context.Context) error) error {
	for i := 0; i < deleteAttempts; i++ {
		err := get(ctx)
		if err != nil && isNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(deleteDelay):
		}
	}
	return fmt.Errorf("still present after %d checks", deleteAttempts)
}
