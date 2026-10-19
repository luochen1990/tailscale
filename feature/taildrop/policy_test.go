// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package taildrop

import "testing"

func TestConsentApprovalAfterOptOut(t *testing.T) {
	m, _, _ := consentTestManager(t)
	m.opts.AllowExternalTaildrop = func() bool { return false }
	if err := m.resolveConsent("pending", true); err != ErrConsentNotAllowed {
		t.Fatalf("approve after opt-out: %v", err)
	}
}
