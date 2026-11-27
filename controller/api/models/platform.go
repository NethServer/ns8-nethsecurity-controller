/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Giacomo Sanchietti <giacomo.sanchietti@nethesis.it>
 */

package models

type PlatformInfo struct {
	VpnPort              string `json:"vpn_port" structs:"vpn_port"`
	VpnNetwork           string `json:"vpn_network" structs:"vpn_network"`
	ControllerVersion    string `json:"controller_version" structs:"controller_version"`
	MetricsRetentionDays int    `json:"metrics_retention_days" structs:"metrics_retention_days"`
	LogsRetentionDays    int    `json:"logs_retention_days" structs:"logs_retention_days"`
}

// StorageUsage is the disk space used by the data collected from the units,
// calculated periodically by the module
type StorageUsage struct {
	LogsStorageBytes           int64 `json:"logs_storage_bytes" structs:"logs_storage_bytes"`
	SystemMetricsStorageBytes  int64 `json:"system_metrics_storage_bytes" structs:"system_metrics_storage_bytes"`
	TrafficMetricsStorageBytes int64 `json:"traffic_metrics_storage_bytes" structs:"traffic_metrics_storage_bytes"`
	TotalStorageBytes          int64 `json:"total_storage_bytes" structs:"total_storage_bytes"`
	StorageUpdatedAt           int64 `json:"storage_updated_at" structs:"storage_updated_at"` // unix timestamp
}
