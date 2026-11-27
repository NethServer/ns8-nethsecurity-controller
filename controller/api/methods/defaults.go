/*
 * Copyright (C) 2024 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"

	"github.com/NethServer/nethsecurity-controller/api/configuration"
	"github.com/NethServer/nethsecurity-controller/api/logs"
	"github.com/NethServer/nethsecurity-controller/api/models"
	"github.com/NethServer/nethsecurity-controller/api/response"
	"github.com/fatih/structs"
	"github.com/gin-gonic/gin"
)

func GetDefaults(c *gin.Context) {
	// read and return defaults path
	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "success",
		Data: gin.H{
			"fqdn":               configuration.Config.FQDN,
			"grafana_path":       configuration.Config.GrafanaPath,
			"valid_subscription": configuration.Config.ValidSubscription,
		},
	}))
}

func GetPlatformInfo(c *gin.Context) {
	// read and return platform info
	data := structs.Map(configuration.Config.PlatformInfo)

	// add the storage usage, if the module already calculated it
	usage, err := readStorageUsage(configuration.Config.StorageUsageFile)
	if err == nil {
		for key, value := range structs.Map(usage) {
			data[key] = value
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		logs.Logs.Println("[ERR][PLATFORM] cannot read storage usage: " + err.Error())
	}

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "success",
		Data:    data,
	}))
}

func readStorageUsage(path string) (models.StorageUsage, error) {
	var usage models.StorageUsage
	content, err := os.ReadFile(path)
	if err != nil {
		return usage, err
	}
	err = json.Unmarshal(content, &usage)
	return usage, err
}
