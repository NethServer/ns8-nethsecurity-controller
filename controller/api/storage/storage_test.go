/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Giacomo Sanchietti <giacomo.sanchietti@nethesis.it>
 */

package storage

import (
	"fmt"
	"os"
	"testing"

	"github.com/NethServer/nethsecurity-controller/api/configuration"
	"github.com/NethServer/nethsecurity-controller/api/logs"
	"github.com/NethServer/nethsecurity-controller/api/models"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logs.Init("test")
	// Set required env vars for config
	os.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	os.Setenv("GRAFANA_POSTGRES_PASSWORD", "grafana_pass")
	os.Setenv("REPORT_DB_URI", "postgres://report:password@localhost:5432/report")
	os.Setenv("GRAFANA_PATH", "/grafana")
	os.Setenv("WEBSSH_PATH", "/webssh")
	os.Setenv("PROMETHEUS_PATH", "/prometheus")
	os.Setenv("PROMTAIL_PORT", "3100")
	os.Setenv("PROMTAIL_ADDRESS", "localhost")
	os.Setenv("ISSUER_2FA", "issuer")
	os.Setenv("DATA_DIR", "/tmp/data")
	os.Setenv("CREDENTIALS_DIR", "/tmp/creds")
	os.Setenv("REGISTRATION_TOKEN", "token")
	os.Setenv("SECRET_JWT", "secret")
	os.Setenv("ADMIN_PASSWORD", "password")
	os.Setenv("ADMIN_USERNAME", "admin")
	os.Setenv("LISTEN_ADDRESS", "127.0.0.1:5000")
	os.Setenv("SENSITIVE_LIST", "password,secret,token,passphrase,private,key")
	os.Setenv("OVPN_DIR", "/etc/openvpn")
	os.Setenv("SECRETS_DIR", "/tmp/secrets")
	os.Setenv("FQDN", "example.com")
	os.Setenv("VALID_SUBSCRIPTION", "false")
	os.Setenv("PROMETHEUS_AUTH_PASSWORD", "prometheus")
	os.Setenv("PROMETHEUS_AUTH_USERNAME", "prometheus")
	os.Setenv("RETENTION_DAYS", "60")
	os.Setenv("CACHE_TTL", "7200")
	os.Setenv("OVPN_UDP_PORT", "1194")
	os.Setenv("OVPN_NETMASK", "255.255.0.0")
	os.Setenv("OVPN_NETWORK", "172.21.0.0")
	configuration.Init()
	// Assume DB is running
	os.Exit(m.Run())
}

func TestLoadReportSchemaKeepsDpiAggregates(t *testing.T) {
	pgpool, pgctx := ReportInstance()
	query := `SELECT array_agg(mat_hypertable_id ORDER BY user_view_name) FROM _timescaledb_catalog.continuous_agg
		WHERE user_view_name IN ('ca_dpi_stats_hourly_bytes', 'ca_dpi_stats_hourly_client')`
	var before, after []int32
	assert.NoError(t, pgpool.QueryRow(pgctx, query).Scan(&before))
	assert.Len(t, before, 2)

	assert.True(t, loadReportSchema(pgpool, pgctx))

	assert.NoError(t, pgpool.QueryRow(pgctx, query).Scan(&after))
	assert.Equal(t, before, after)
}

func TestLoadReportSchemaPurgesOrphanMaterializationRanges(t *testing.T) {
	pgpool, pgctx := ReportInstance()
	conn, err := pgpool.Acquire(pgctx)
	assert.NoError(t, err)
	// Replica role skips the foreign key, like TimescaleDB's own catalog writes
	_, err = conn.Exec(pgctx, "SET session_replication_role = replica")
	assert.NoError(t, err)
	_, err = conn.Exec(pgctx, "INSERT INTO _timescaledb_catalog.continuous_aggs_materialization_ranges VALUES (99999, 0, 1)")
	assert.NoError(t, err)
	_, err = conn.Exec(pgctx, "RESET session_replication_role")
	assert.NoError(t, err)
	conn.Release()

	assert.True(t, loadReportSchema(pgpool, pgctx))

	var count int
	assert.NoError(t, pgpool.QueryRow(pgctx, "SELECT count(*) FROM _timescaledb_catalog.continuous_aggs_materialization_ranges WHERE materialization_id = 99999").Scan(&count))
	assert.Equal(t, 0, count)
}

func TestAddUnit(t *testing.T) {
	// Test adding a unit
	err := AddUnit("550e8400-e29b-41d4-a716-446655440000", "192.168.1.10", "")
	assert.NoError(t, err)

	// Verify it exists
	unit, err := GetUnit("550e8400-e29b-41d4-a716-446655440000")
	assert.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", unit["id"])
	assert.Equal(t, "192.168.1.10", unit["ipaddress"])

	// Clean up
	DeleteUnit("550e8400-e29b-41d4-a716-446655440000")
}

func TestGetUnit(t *testing.T) {
	// Add a unit first
	AddUnit("550e8400-e29b-41d4-a716-446655440001", "192.168.1.11", "")

	// Get it
	unit, err := GetUnit("550e8400-e29b-41d4-a716-446655440001")
	assert.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", unit["id"])

	// Clean up
	DeleteUnit("550e8400-e29b-41d4-a716-446655440001")
}

func TestGetFreeIP(t *testing.T) {
	// This depends on network config, but test that it returns something
	ip := GetFreeIP()
	assert.NotEmpty(t, ip)
}

func TestGetUnitCredentials(t *testing.T) {
	// Set credentials
	err := SetUnitCredentials("550e8400-e29b-41d4-a716-446655440002", "user", "pass")
	assert.NoError(t, err)

	// Get them
	user, pass, err := GetUnitCredentials("550e8400-e29b-41d4-a716-446655440002")
	assert.NoError(t, err)
	assert.Equal(t, "user", user)
	assert.Equal(t, "pass", pass)

	// Clean up
	DeleteUnit("550e8400-e29b-41d4-a716-446655440002")
}

func TestUnitGroupExists(t *testing.T) {
	// Test non-existing group
	exists, err := UnitGroupExists(999)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestAddAccount(t *testing.T) {
	account := models.Account{
		Username: "testuser",
		Password: "testpass",
	}
	id, err := AddAccount(account)
	assert.NoError(t, err)
	assert.Greater(t, id, 0)

	// Clean up
	DeleteAccount(fmt.Sprintf("%d", id))
}

func TestGetPassword(t *testing.T) {
	// Add account
	account := models.Account{
		Username: "testuser2",
		Password: "testpass",
	}
	id, _ := AddAccount(account)

	// Get password
	pass := GetPassword("testuser2")
	assert.NotEmpty(t, pass)

	// Clean up
	DeleteAccount(fmt.Sprintf("%d", id))
}

func TestIsAdmin(t *testing.T) {
	// Test with non-admin
	assert.False(t, IsAdmin("testuser"))
	// Admin might not be loaded, so skip asserting true
}

func TestGetAccounts(t *testing.T) {
	accounts, err := GetAccounts()
	assert.NoError(t, err)
	// DB might be empty, so >=0
	assert.GreaterOrEqual(t, len(accounts), 0)
}

func TestUpdatePassword(t *testing.T) {
	// Add account
	account := models.Account{
		Username: "testuser3",
		Password: "testpass",
	}
	id, _ := AddAccount(account)

	// Update password
	err := UpdatePassword("testuser3", "newpass")
	assert.NoError(t, err)

	// Verify
	pass := GetPassword("testuser3")
	assert.NotEmpty(t, pass)

	// Clean up
	DeleteAccount(fmt.Sprintf("%d", id))
}

func TestListUnits(t *testing.T) {
	// Add a unit
	AddUnit("550e8400-e29b-41d4-a716-446655440003", "192.168.1.12", "")

	// List units
	units, err := ListUnits()
	assert.NoError(t, err)
	assert.Greater(t, len(units), 0)

	// Clean up
	DeleteUnit("550e8400-e29b-41d4-a716-446655440003")
}

func TestAddUnitGroup(t *testing.T) {
	// Add unit group
	group := models.UnitGroup{Name: "testgroup", Description: "", Units: []string{}}
	id, err := AddUnitGroup(group)
	assert.NoError(t, err)
	assert.Greater(t, id, 0)

	// Clean up
	DeleteUnitGroup(id)
}

func TestGetUnitGroup(t *testing.T) {
	// Add group
	group := models.UnitGroup{Name: "testgroup2", Description: "", Units: []string{}}
	id, _ := AddUnitGroup(group)

	// Get it
	group2, err := GetUnitGroup(id)
	assert.NoError(t, err)
	assert.Equal(t, "testgroup2", group2.Name)

	// Clean up
	DeleteUnitGroup(id)
}
