/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 */

// Package sshterm bridges a websocket to an SSH shell on a unit.
//
// The client sends a JSON init message {"passphrase","password","cols","rows"} as the first frame.
// After that, binary frames carry terminal input and output, and text frames carry
// control messages ({"type":"resize","cols","rows"}). The session ends with a close
// frame: 1000 when the shell exits, 1011 with the error as reason otherwise.
package sshterm

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

var (
	ErrUnitNotFound       = errors.New("unit not found")
	ErrUnitNotConnected   = errors.New("unit not connected")
	ErrSSHDisabled        = errors.New("ssh disabled on unit")
	ErrNoKey              = errors.New("no ssh key for user")
	ErrPassphraseRequired = errors.New("key passphrase required")
	ErrWrongPassphrase    = errors.New("wrong key passphrase")
	ErrWrongPassword      = errors.New("wrong root password")
)

const (
	sshUser     = "root"
	dialTimeout = 10 * time.Second
	initTimeout = 30 * time.Second
	closeWait   = time.Second
)

// UnitResolver returns the host:port of a unit's SSH server.
type UnitResolver interface {
	Resolve(ctx context.Context, unitID string) (string, error)
}

// KeyLoader returns the PEM encoded private key of a user.
type KeyLoader interface {
	Load(user string) ([]byte, error)
}

type Handler struct {
	units    UnitResolver
	keys     KeyLoader
	upgrader websocket.Upgrader
}

func New(units UnitResolver, keys KeyLoader) *Handler {
	return &Handler{units: units, keys: keys}
}

type initMessage struct {
	Passphrase string `json:"passphrase"`
	Password   string `json:"password"`
	Cols       int    `json:"cols"`
	Rows       int    `json:"rows"`
}

type controlMessage struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// Serve upgrades the request and runs a shell on unitID as root, authenticated with the root
// password from the init message or else with user's key.
// Callers must have already authorized user for unitID.
func (h *Handler) Serve(w http.ResponseWriter, r *http.Request, user, unitID string) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade already replied with an HTTP error
		return
	}
	defer conn.Close()

	closeWith(conn, h.run(r.Context(), conn, user, unitID))
}

func (h *Handler) run(ctx context.Context, conn *websocket.Conn, user, unitID string) error {
	// Init message: passphrase and initial terminal size
	var init initMessage
	if err := conn.SetReadDeadline(time.Now().Add(initTimeout)); err != nil {
		return err
	}
	if err := conn.ReadJSON(&init); err != nil {
		return fmt.Errorf("read init message: %w", err)
	}
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		return err
	}

	// Unit address and credentials
	addr, err := h.units.Resolve(ctx, unitID)
	if err != nil {
		return fmt.Errorf("resolve unit %s: %w", unitID, err)
	}
	auth, err := h.authMethod(user, init)
	if err != nil {
		return err
	}

	// SSH connection and shell on a pty
	client, err := dial(ctx, addr, auth)
	// x/crypto/ssh has no typed error for a refused login
	if err != nil && init.Password != "" && strings.Contains(err.Error(), "unable to authenticate") {
		return ErrWrongPassword
	}
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	defer session.Close()

	if err := session.RequestPty("xterm-256color", init.Rows, init.Cols, ssh.TerminalModes{}); err != nil {
		return fmt.Errorf("request pty: %w", err)
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("open stdin: %w", err)
	}
	out := &wsWriter{conn: conn}
	session.Stdout = out
	session.Stderr = out
	if err := session.Shell(); err != nil {
		return fmt.Errorf("start shell: %w", err)
	}

	// Bridge until the shell exits or the client goes away
	readDone := make(chan error, 1)
	go func() { readDone <- pump(conn, stdin, session) }()
	waitDone := make(chan error, 1)
	go func() { waitDone <- session.Wait() }()

	select {
	case err := <-waitDone:
		if _, ok := errors.AsType[*ssh.ExitError](err); ok {
			return nil
		}
		return err
	case err := <-readDone:
		return err
	}
}

// pump forwards client frames to the shell until the client goes away.
func pump(conn *websocket.Conn, stdin io.Writer, session *ssh.Session) error {
	for {
		kind, data, err := conn.ReadMessage()
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read websocket: %w", err)
		}

		switch kind {
		case websocket.BinaryMessage:
			if _, err := stdin.Write(data); err != nil {
				return fmt.Errorf("write stdin: %w", err)
			}
		case websocket.TextMessage:
			var msg controlMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				return fmt.Errorf("decode control message: %w", err)
			}
			if msg.Type != "resize" {
				return fmt.Errorf("unknown control message %q", msg.Type)
			}
			if err := session.WindowChange(msg.Rows, msg.Cols); err != nil {
				return fmt.Errorf("resize: %w", err)
			}
		}
	}
}

// authMethod returns the root password when given, user's key otherwise.
func (h *Handler) authMethod(user string, init initMessage) (ssh.AuthMethod, error) {
	if init.Password != "" {
		return ssh.Password(init.Password), nil
	}
	pemBytes, err := h.keys.Load(user)
	if err != nil {
		return nil, fmt.Errorf("load key of %s: %w", user, err)
	}
	signer, err := parseKey(pemBytes, init.Passphrase)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

func parseKey(pemBytes []byte, passphrase string) (ssh.Signer, error) {
	signer, err := ssh.ParsePrivateKey(pemBytes)
	if _, ok := errors.AsType[*ssh.PassphraseMissingError](err); ok {
		if passphrase == "" {
			return nil, ErrPassphraseRequired
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(passphrase))
	}
	if errors.Is(err, x509.IncorrectPasswordError) {
		return nil, ErrWrongPassphrase
	}
	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}
	return signer, nil
}

func dial(ctx context.Context, addr string, auth ssh.AuthMethod) (*ssh.Client, error) {
	dialer := net.Dialer{Timeout: dialTimeout}
	nc, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	config := &ssh.ClientConfig{
		User: sshUser,
		Auth: []ssh.AuthMethod{auth},
		// Units are already authenticated by their VPN certificate
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if err := nc.SetDeadline(time.Now().Add(dialTimeout)); err != nil {
		nc.Close()
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(nc, addr, config)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("ssh handshake with %s: %w", addr, err)
	}
	if err := nc.SetDeadline(time.Time{}); err != nil {
		c.Close()
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

// wsWriter sends each write as a binary frame; stdout and stderr share it.
type wsWriter struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.conn.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func closeWith(conn *websocket.Conn, err error) {
	code, reason := websocket.CloseNormalClosure, ""
	if err != nil {
		code, reason = websocket.CloseInternalServerErr, err.Error()
	}
	// Close frame payload is capped at 125 bytes, 2 of which hold the code
	if len(reason) > 123 {
		reason = reason[:123]
	}
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(closeWait))
}
