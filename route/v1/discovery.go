package v1

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS/service"
	"github.com/labstack/echo/v4"
)

// GetSystemInfo serves the /v1/sys/info LAN-discovery probe used by Zima
// client apps to detect and identify a CasaOS device before login.
func GetSystemInfo(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, echo.Map{
		"success": http.StatusOK,
		"message": "",
		"data":    service.MyService.System().GetDeviceInfo(),
	})
}
