// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build (linux && !android) || (darwin && !ios) || freebsd || openbsd || plan9

package tailssh

import (
	"io"
	"net"
	"testing"
	"time"
)

// connSetupBackend pauses connection construction after newConn's shutdown check.
type connSetupBackend struct {
	ipnLocalBackend
	varRoot string
	entered chan struct{}
	release chan struct{}
}

func (b *connSetupBackend) TailscaleVarRoot() string {
	close(b.entered)
	<-b.release
	return b.varRoot
}

func TestHandleSSHConnDuringShutdown(t *testing.T) {
	for _, duringSetup := range []bool{false, true} {
		name := "before-setup"
		if duringSetup {
			name = "during-setup"
		}
		t.Run(name, func(t *testing.T) {
			b := &connSetupBackend{
				varRoot: t.TempDir(),
				entered: make(chan struct{}),
				release: make(chan struct{}),
			}
			srv := &server{lb: b, logf: t.Logf}
			client, serverConn := net.Pipe()
			defer client.Close()
			defer serverConn.Close()
			if err := client.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Fatal(err)
			}
			if !duringSetup {
				srv.Shutdown()
			}
			done := make(chan error, 1)
			go func() { done <- srv.HandleSSHConn(serverConn) }()
			if duringSetup {
				<-b.entered
				srv.Shutdown()
			}
			close(b.release)
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("HandleSSHConn succeeded after shutdown")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("HandleSSHConn did not return after shutdown")
			}
			if got := srv.NumActiveConns(); got != 0 {
				t.Errorf("active connections = %d, want 0", got)
			}
			var buf [1]byte
			if _, err := client.Read(buf[:]); err != io.EOF {
				t.Errorf("client Read = %v, want EOF", err)
			}
		})
	}
}
