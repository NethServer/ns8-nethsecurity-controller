/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 */

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NethServer/nethsecurity-controller/api/internal/sshterm"
	"github.com/NethServer/nethsecurity-controller/api/storage"
)

func login(t *testing.T, router *gin.Engine, username, password string) string {
	t.Helper()
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "login as %s", username)

	var resp struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp.Token
}

func TestTerminalAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupRouter()

	adminToken := login(t, router, "admin", "admin")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/accounts", bytes.NewBufferString(`{"username": "terminaluser", "password": "terminal", "display_name": "Terminal", "admin": false}`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	limitedToken := login(t, router, "terminaluser", "terminal")

	tests := []struct {
		name   string
		setup  func(req *http.Request)
		status int
	}{
		{name: "no credentials", setup: func(*http.Request) {}, status: http.StatusUnauthorized},
		{name: "invalid cookie", setup: func(req *http.Request) {
			req.AddCookie(&http.Cookie{Name: "ns_jwt", Value: "invalid.token.signature"})
		}, status: http.StatusUnauthorized},
		{name: "bearer header is not enough", setup: func(req *http.Request) {
			req.Header.Set("Authorization", "Bearer "+adminToken)
		}, status: http.StatusUnauthorized},
		{name: "basic auth is not enough", setup: func(req *http.Request) {
			req.SetBasicAuth("admin", "admin")
		}, status: http.StatusUnauthorized},
		{name: "user without access to unit", setup: func(req *http.Request) {
			req.AddCookie(&http.Cookie{Name: "ns_jwt", Value: limitedToken})
		}, status: http.StatusForbidden},
		// auth passes, then the upgrade fails since this is not a websocket request
		{name: "admin cookie", setup: func(req *http.Request) {
			req.AddCookie(&http.Cookie{Name: "ns_jwt", Value: adminToken})
		}, status: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/units/some-unit/terminal", nil)
			tt.setup(req)
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.status, w.Code)
		})
	}
}

func TestPGUnitsResolve(t *testing.T) {
	setupRouter()
	pool, ctx := storage.ReportInstance()

	units := []struct {
		uuid, addr, info string
		connected        bool
	}{
		{uuid: "6b0a8a4e-0001-4000-8000-000000000001", addr: "172.21.0.5", info: `{"ssh_port": 22}`, connected: true},
		{uuid: "6b0a8a4e-0001-4000-8000-000000000002", addr: "172.21.0.6", info: `{"ssh_port": 22}`},
		{uuid: "6b0a8a4e-0001-4000-8000-000000000003", addr: "172.21.0.7", info: `{"ssh_port": 0}`, connected: true},
		{uuid: "6b0a8a4e-0001-4000-8000-000000000004", addr: "172.21.0.8", info: `{}`, connected: true},
	}
	for _, u := range units {
		_, err := pool.Exec(ctx, `
			INSERT INTO units (uuid, vpn_address, info, vpn_connected_since)
			VALUES ($1, $2, $3::jsonb, CASE WHEN $4 THEN NOW() END)
			ON CONFLICT (uuid) DO UPDATE SET vpn_address = $2, info = $3::jsonb, vpn_connected_since = CASE WHEN $4 THEN NOW() END
		`, u.uuid, u.addr, u.info, u.connected)
		require.NoError(t, err)
		t.Cleanup(func() { _, _ = pool.Exec(ctx, "DELETE FROM units WHERE uuid = $1", u.uuid) })
	}

	tests := []struct {
		name     string
		uuid     string
		wantAddr string
		wantErr  error
	}{
		{name: "connected", uuid: units[0].uuid, wantAddr: "172.21.0.5:22"},
		{name: "not connected", uuid: units[1].uuid, wantErr: sshterm.ErrUnitNotConnected},
		{name: "ssh port 0", uuid: units[2].uuid, wantErr: sshterm.ErrSSHDisabled},
		{name: "no info", uuid: units[3].uuid, wantErr: sshterm.ErrSSHDisabled},
		{name: "unknown unit", uuid: "6b0a8a4e-0001-4000-8000-0000000000ff", wantErr: sshterm.ErrUnitNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := sshterm.PGUnits{Pool: pool}.Resolve(ctx, tt.uuid)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantAddr, addr)
		})
	}
}
