/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 */

package sshterm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGUnits resolves units from the units table.
type PGUnits struct {
	Pool *pgxpool.Pool
}

func (u PGUnits) Resolve(ctx context.Context, unitID string) (string, error) {
	var (
		addr      *string
		connected bool
		port      int
	)
	err := u.Pool.QueryRow(ctx, `
		SELECT vpn_address, vpn_connected_since IS NOT NULL, COALESCE((info->>'ssh_port')::int, 0)
		FROM units WHERE uuid = $1
	`, unitID).Scan(&addr, &connected, &port)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUnitNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query unit: %w", err)
	}
	if !connected || addr == nil || *addr == "" {
		return "", ErrUnitNotConnected
	}
	if port == 0 {
		return "", ErrSSHDisabled
	}
	return net.JoinHostPort(*addr, strconv.Itoa(port)), nil
}
