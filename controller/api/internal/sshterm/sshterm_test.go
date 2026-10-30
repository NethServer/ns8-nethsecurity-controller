/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 */

package sshterm

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

type fakeUnits struct {
	addr string
	err  error
}

func (f fakeUnits) Resolve(context.Context, string) (string, error) { return f.addr, f.err }

type fakeKeys struct {
	pem []byte
	err error
}

func (f fakeKeys) Load(string) ([]byte, error) { return f.pem, f.err }

type windowChange struct {
	Cols, Rows, Width, Height uint32
}

// startSSHServer accepts the given client key and root password as root and runs an echo shell: it
// writes back whatever it reads and exits on "exit". Window changes are sent to the returned channel.
func startSSHServer(t *testing.T, clientKey ssh.PublicKey, rootPassword string) (string, <-chan windowChange) {
	t.Helper()

	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatal(err)
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if meta.User() == "root" && bytes.Equal(key.Marshal(), clientKey.Marshal()) {
				return nil, nil
			}
			return nil, errors.New("unauthorized")
		},
		PasswordCallback: func(meta ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if meta.User() == "root" && string(password) == rootPassword {
				return nil, nil
			}
			return nil, errors.New("unauthorized")
		},
	}
	config.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	resizes := make(chan windowChange, 1)
	go func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				return
			}
			go serveConn(nc, config, resizes)
		}
	}()
	return ln.Addr().String(), resizes
}

func serveConn(nc net.Conn, config *ssh.ServerConfig, resizes chan<- windowChange) {
	_, chans, reqs, err := ssh.NewServerConn(nc, config)
	if err != nil {
		return
	}
	go ssh.DiscardRequests(reqs)
	for newChan := range chans {
		ch, reqs, err := newChan.Accept()
		if err != nil {
			return
		}
		go func() {
			for req := range reqs {
				if req.Type == "window-change" {
					var wc windowChange
					if err := ssh.Unmarshal(req.Payload, &wc); err == nil {
						resizes <- wc
					}
				}
				if req.WantReply {
					_ = req.Reply(true, nil)
				}
			}
		}()
		go func() {
			defer ch.Close()
			buf := make([]byte, 1024)
			for {
				n, err := ch.Read(buf)
				if err != nil {
					return
				}
				if string(buf[:n]) == "exit" {
					_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
					return
				}
				_, _ = ch.Write(buf[:n])
			}
		}()
	}
}

func newClientKey(t *testing.T, passphrase string) (ssh.PublicKey, []byte) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var block *pem.Block
	if passphrase == "" {
		block, err = ssh.MarshalPrivateKey(priv, "")
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte(passphrase))
	}
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return sshPub, pem.EncodeToMemory(block)
}

func connect(t *testing.T, h *Handler, init initMessage) *websocket.Conn {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.Serve(w, r, "admin", "unit")
	}))
	t.Cleanup(srv.Close)

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteJSON(init); err != nil {
		t.Fatal(err)
	}
	return conn
}

// readUntil collects binary output until it contains want.
func readUntil(t *testing.T, conn *websocket.Conn, want string) {
	t.Helper()
	var got []byte
	for !bytes.Contains(got, []byte(want)) {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read output %q: %v", got, err)
		}
		got = append(got, data...)
	}
}

func TestServe(t *testing.T) {
	plainPub, plainPEM := newClientKey(t, "")
	encPub, encPEM := newClientKey(t, "secret")

	tests := []struct {
		name       string
		serverKey  ssh.PublicKey
		keys       fakeKeys
		unitsErr   error
		passphrase string
		password   string
		wantErr    string
	}{
		{name: "unencrypted key", serverKey: plainPub, keys: fakeKeys{pem: plainPEM}},
		{name: "unencrypted key ignores passphrase", serverKey: plainPub, keys: fakeKeys{pem: plainPEM}, passphrase: "unused"},
		{name: "encrypted key", serverKey: encPub, keys: fakeKeys{pem: encPEM}, passphrase: "secret"},
		{name: "wrong passphrase", serverKey: encPub, keys: fakeKeys{pem: encPEM}, passphrase: "nope", wantErr: ErrWrongPassphrase.Error()},
		{name: "missing passphrase", serverKey: encPub, keys: fakeKeys{pem: encPEM}, wantErr: ErrPassphraseRequired.Error()},
		{name: "no key", serverKey: plainPub, keys: fakeKeys{err: ErrNoKey}, wantErr: ErrNoKey.Error()},
		{name: "key not authorized on unit", serverKey: encPub, keys: fakeKeys{pem: plainPEM}, wantErr: "unable to authenticate"},
		{name: "root password without key", serverKey: plainPub, keys: fakeKeys{err: ErrNoKey}, password: "rootpw"},
		{name: "wrong root password", serverKey: plainPub, keys: fakeKeys{pem: plainPEM}, password: "nope", wantErr: ErrWrongPassword.Error()},
		{name: "unit not connected", serverKey: plainPub, keys: fakeKeys{pem: plainPEM}, unitsErr: ErrUnitNotConnected, wantErr: ErrUnitNotConnected.Error()},
		{name: "ssh disabled", serverKey: plainPub, keys: fakeKeys{pem: plainPEM}, unitsErr: ErrSSHDisabled, wantErr: ErrSSHDisabled.Error()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, _ := startSSHServer(t, tt.serverKey, "rootpw")
			h := New(fakeUnits{addr: addr, err: tt.unitsErr}, tt.keys)
			conn := connect(t, h, initMessage{Passphrase: tt.passphrase, Password: tt.password, Cols: 80, Rows: 24})

			if tt.wantErr == "" {
				if err := conn.WriteMessage(websocket.BinaryMessage, []byte("hello")); err != nil {
					t.Fatal(err)
				}
				readUntil(t, conn, "hello")
				if err := conn.WriteMessage(websocket.BinaryMessage, []byte("exit")); err != nil {
					t.Fatal(err)
				}
			}

			var err error
			for err == nil {
				_, _, err = conn.ReadMessage()
			}
			closeErr, ok := errors.AsType[*websocket.CloseError](err)
			if !ok {
				t.Fatalf("expected close frame, got %v", err)
			}
			if tt.wantErr == "" {
				if closeErr.Code != websocket.CloseNormalClosure {
					t.Fatalf("expected normal closure, got %v", closeErr)
				}
				return
			}
			if closeErr.Code != websocket.CloseInternalServerErr || !strings.Contains(closeErr.Text, tt.wantErr) {
				t.Fatalf("expected 1011 with %q, got %v", tt.wantErr, closeErr)
			}
		})
	}
}

func TestServeResize(t *testing.T) {
	pub, key := newClientKey(t, "")
	addr, resizes := startSSHServer(t, pub, "rootpw")
	conn := connect(t, New(fakeUnits{addr: addr}, fakeKeys{pem: key}), initMessage{Cols: 80, Rows: 24})

	// the echo proves the shell is up before resizing
	if err := conn.WriteMessage(websocket.BinaryMessage, []byte("ready")); err != nil {
		t.Fatal(err)
	}
	readUntil(t, conn, "ready")
	if err := conn.WriteJSON(controlMessage{Type: "resize", Cols: 120, Rows: 40}); err != nil {
		t.Fatal(err)
	}

	select {
	case wc := <-resizes:
		if wc.Cols != 120 || wc.Rows != 40 {
			t.Fatalf("expected 120x40, got %dx%d", wc.Cols, wc.Rows)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no window-change received")
	}
}

func TestDirKeys(t *testing.T) {
	_, err := DirKeys{Dir: t.TempDir()}.Load("missing")
	if !errors.Is(err, ErrNoKey) {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
}
