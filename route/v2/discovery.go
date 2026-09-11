package v2

import (
	"net/http"
	"strings"

	"github.com/IceWhaleTech/CasaOS/service"
	"github.com/labstack/echo/v4"
)

// discoveryPaths are the unauthenticated LAN-discovery / device-info endpoints.
// They are intentionally excluded from JWT auth and OpenAPI validation so Zima
// and CasaOS client apps can identify the device before the user signs in.
var discoveryPaths = map[string]bool{
	"/v2/zimaos/device/info": true,
	"/v2/casaos/device/info": true,
	"/v2/sys/info":           true,
}

// IsDiscoveryPath reports whether an absolute request path belongs to the
// unauthenticated discovery endpoints. Trailing slashes are tolerated.
func IsDiscoveryPath(requestPath string) bool {
	return discoveryPaths[strings.TrimRight(requestPath, "/")]
}

// GetDeviceInfo serves /v2/zimaos/device/info and /v2/casaos/device/info.
// The body carries the CasaOS {success, message, data} envelope AND mirrors the
// DeviceInfo fields at the top level, because the Zima clients (casazima) read
// `os_version` from the top level of the /v2/zimaos/device/info response.
func GetDeviceInfo(ctx echo.Context) error {
	info := service.MyService.System().GetDeviceInfo()
	return ctx.JSON(http.StatusOK, echo.Map{
		"success": http.StatusOK,
		"message": "",
		"data":    info,
		// top-level mirrors for Zima client compatibility:
		"lan_ipv4":     info.LanIpv4,
		"port":         info.Port,
		"device_name":  info.DeviceName,
		"device_model": info.DeviceModel,
		"device_sn":    info.DeviceSN,
		"initialized":  info.Initialized,
		"os_version":   info.OS_Version,
		"hash":         info.Hash,
	})
}

// GetSysInfo serves the /v2/sys/info (and /v1/sys/info) discovery probe.
func GetSysInfo(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, echo.Map{
		"success": http.StatusOK,
		"message": "",
		"data":    service.MyService.System().GetDeviceInfo(),
	})
}
