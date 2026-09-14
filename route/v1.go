package route

import (
	"crypto/ecdsa"
	"net/http"
	"strconv"

	"github.com/IceWhaleTech/CasaOS-Common/external"
	"github.com/IceWhaleTech/CasaOS-Common/utils/jwt"
	"github.com/IceWhaleTech/CasaOS/common"
	"github.com/IceWhaleTech/CasaOS/pkg/config"
	v1 "github.com/IceWhaleTech/CasaOS/route/v1"
	echo_jwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	echo_middleware "github.com/labstack/echo/v4/middleware"
)

// forceAuthMiddleware requires a valid JWT even for localhost requests.
// This prevents local unprivileged processes from bypassing authentication
// on sensitive routes (file API, batch operations, image serving).
func forceAuthMiddleware() echo.MiddlewareFunc {
	return echo_jwt.WithConfig(echo_jwt.Config{
		Skipper: func(c echo.Context) bool {
			return false // Never skip — always require auth
		},
		ParseTokenFunc: func(c echo.Context, token string) (interface{}, error) {
			valid, claims, err := jwt.Validate(token, func() (*ecdsa.PublicKey, error) {
				return external.GetPublicKey(config.CommonInfo.RuntimePath)
			})
			if err != nil || !valid {
				return nil, echo.ErrUnauthorized
			}
			c.Request().Header.Set("user_id", strconv.Itoa(claims.ID))
			return claims, nil
		},
		TokenLookupFuncs: []echo_middleware.ValuesExtractor{
			tokenFromRequest,
		},
	})
}

// websocketPaths lists the browser WebSocket endpoints. A browser WebSocket
// cannot set the Authorization header, so these routes additionally accept the
// JWT as a short-lived `token` query parameter (which the frontend already
// uses). Every other route stays header-only.
func websocketPaths() map[string]bool {
	return map[string]bool{
		"/v1/sys/wsssh": true,
		"/v1/file/ws":   true,
	}
}

// tokenFromRequest extracts the JWT from the Authorization header, or — only
// for the WebSocket endpoints above — from the `token` query parameter.
func tokenFromRequest(c echo.Context) ([]string, error) {
	if len(c.Request().Header.Get(echo.HeaderAuthorization)) > 0 {
		return []string{c.Request().Header.Get(echo.HeaderAuthorization)}, nil
	}
	if websocketPaths()[c.Path()] {
		if t := c.QueryParam("token"); t != "" {
			return []string{t}, nil
		}
	}
	return []string{""}, nil
}

func InitV1Router() http.Handler {
	e := echo.New()

	// SECURITY: Never trust client-supplied X-Forwarded-For / X-Real-IP headers for
	// remote-address determination. Extract the IP directly from the socket, so
	// RealIP() cannot be spoofed (the JWT skipper previously relied on it).
	e.IPExtractor = echo.ExtractIPDirect()

	e.Use((echo_middleware.CORSWithConfig(echo_middleware.CORSConfig{
		AllowOrigins: []string{
			"http://127.0.0.1:*",
			"http://localhost:*",
			"https://127.0.0.1:*",
			"https://localhost:*",
		},
		AllowMethods:     []string{echo.POST, echo.GET, echo.OPTIONS, echo.PUT, echo.DELETE},
		AllowHeaders:     []string{echo.HeaderAuthorization, echo.HeaderContentLength, echo.HeaderXCSRFToken, echo.HeaderContentType, echo.HeaderAccessControlAllowOrigin, echo.HeaderAccessControlAllowHeaders, echo.HeaderAccessControlAllowMethods, echo.HeaderConnection, echo.HeaderOrigin, echo.HeaderXRequestedWith},
		ExposeHeaders:    []string{echo.HeaderContentLength, echo.HeaderAccessControlAllowOrigin, echo.HeaderAccessControlAllowHeaders},
		MaxAge:           172800,
		AllowCredentials: false,
	})))
	e.Use(echo_middleware.Gzip())
	e.Use(echo_middleware.Recover())
	e.Use(echo_middleware.Logger())

	// Debug endpoint moved inside v1SysGroup for JWT auth protection

	v1Group := e.Group("/v1")
	//	e.Any("/v1/test", v1.CheckNetwork)
	v1Group.Use(echo_jwt.WithConfig(echo_jwt.Config{
		Skipper: func(c echo.Context) bool {
			// /v1/sys/info is a LAN-discovery probe — it must be reachable
			// before login. Everything else always requires JWT.
			if c.Request().URL.Path == "/v1/sys/info" {
				return true
			}
			return false // SECURITY: Always require JWT authentication
		},
		ParseTokenFunc: func(c echo.Context, token string) (interface{}, error) {
			valid, claims, err := jwt.Validate(token, func() (*ecdsa.PublicKey, error) { return external.GetPublicKey(config.CommonInfo.RuntimePath) })
			if err != nil || !valid {
				return nil, echo.ErrUnauthorized
			}

			c.Request().Header.Set("user_id", strconv.Itoa(claims.ID))

			return claims, nil
		},
		TokenLookupFuncs: []echo_middleware.ValuesExtractor{
			tokenFromRequest,
		},
	}))
	{

		v1SysGroup := v1Group.Group("/sys")
		v1SysGroup.Use()
		{
			v1SysGroup.GET("/version", v1.GetSystemCheckVersion) // version/check

			v1SysGroup.GET("/info", v1.GetSystemInfo) // LAN-discovery probe (unauthenticated)

			v1SysGroup.POST("/update", v1.SystemUpdate)

			v1SysGroup.GET("/hardware", v1.GetSystemHardwareInfo) // hardware/info

			v1SysGroup.GET("/wsssh", v1.WsSsh)
			v1SysGroup.POST("/ssh-login", v1.PostSshLogin)
			// v1SysGroup.GET("/config", v1.GetSystemConfig) //delete
			// v1SysGroup.POST("/config", v1.PostSetSystemConfig)
			v1SysGroup.GET("/debug", v1.GetSystemConfigDebug)
			v1SysGroup.GET("/logs", v1.GetCasaOSErrorLogs) // error/logs
			// v1SysGroup.GET("/widget/config", v1.GetWidgetConfig)//delete
			// v1SysGroup.POST("/widget/config", v1.PostSetWidgetConfig)//delete

			v1SysGroup.POST("/stop", v1.PostKillCasaOS)

			v1SysGroup.GET("/utilization", v1.GetSystemUtilization)
			v1SysGroup.GET("/utilization/interval", v1.GetSystemUtilizationInterval)
			v1SysGroup.PUT("/utilization/interval", v1.PutSystemUtilizationInterval)
			// v1SysGroup.GET("/cpu", v1.GetSystemCupInfo)
			// v1SysGroup.GET("/mem", v1.GetSystemMemInfo)
			// v1SysGroup.GET("/disk", v1.GetSystemDiskInfo)
			// v1SysGroup.GET("/network", v1.GetSystemNetInfo)

			v1SysGroup.GET("/server-info", nil)
			v1SysGroup.PUT("/server-info", nil)
			// v1SysGroup.GET("/port", v1.GetCasaOSPort)
			// v1SysGroup.PUT("/port", v1.PutCasaOSPort)
			v1SysGroup.GET("/proxy", v1.GetSystemProxy)
			v1SysGroup.PUT("/state/:state", v1.PutSystemState)
			v1SysGroup.GET("/entry", v1.GetSystemEntry)
		}

		// Version endpoints (moved from unauthenticated group — now require auth)
		v1Group.GET("/version/check", v1.GetSystemCheckVersion)
		v1Group.GET("/version/current", func(ctx echo.Context) error {
			return ctx.String(200, common.VERSION)
		})
		// Ping and recover remain accessible without auth via the JWT skipper
		v1Group.GET("/ping", func(ctx echo.Context) error {
			return ctx.String(200, "pong")
		})
		v1Group.GET("/recover/:type", v1.GetRecoverStorage)
		v1PortGroup := v1Group.Group("/port")
		v1PortGroup.Use()
		{
			v1PortGroup.GET("/", v1.GetPort)              // app/port
			v1PortGroup.GET("/state/:port", v1.PortCheck) // app/check/:port
		}
		v1FileGroup := v1Group.Group("/file")
		v1FileGroup.Use(forceAuthMiddleware()) // Enforce auth even from localhost
		{
			v1FileGroup.GET("", v1.GetDownloadSingleFile) // download/:path
			v1FileGroup.POST("", v1.PostCreateFile)
			v1FileGroup.PUT("", v1.PutFileContent)
			v1FileGroup.PUT("/name", v1.RenamePath)
			// file/rename
			v1FileGroup.GET("/content", v1.GetFilerContent) // file/read

			// File uploads need to be handled separately, and will not be modified here
			// v1FileGroup.POST("/upload", v1.PostFileUpload)
			v1FileGroup.POST("/upload", v1.PostFileUpload)
			v1FileGroup.GET("/upload", v1.GetFileUpload)
			// v1FileGroup.GET("/download", v1.UserFileDownloadCommonService)
			v1FileGroup.GET("/ws", v1.ConnectWebSocket)
			v1FileGroup.GET("/peers", v1.GetPeers)
		}
		v1CloudGroup := v1Group.Group("/cloud")
		v1CloudGroup.Use()
		{
			v1CloudGroup.GET("", v1.ListStorages)
			v1CloudGroup.DELETE("", v1.UmountStorage)
		}
		v1DriverGroup := v1Group.Group("/driver")
		v1DriverGroup.Use()
		{
			v1DriverGroup.GET("", v1.ListDriverInfo)
		}

		v1FolderGroup := v1Group.Group("/folder")
		v1FolderGroup.Use(forceAuthMiddleware()) // Enforce auth even from localhost
		{
			v1FolderGroup.PUT("/name", v1.RenamePath)
			v1FolderGroup.GET("", v1.DirPath)   ///file/dirpath
			v1FolderGroup.POST("", v1.MkdirAll) ///file/mkdir
			v1FolderGroup.GET("/size", v1.GetSize)
			v1FolderGroup.GET("/count", v1.GetFileCount)
		}
		v1BatchGroup := v1Group.Group("/batch")
		v1BatchGroup.Use(forceAuthMiddleware()) // Enforce auth even from localhost
		{

			v1BatchGroup.DELETE("", v1.DeleteFile) // file/delete
			v1BatchGroup.DELETE("/:id/task", v1.DeleteOperateFileOrDir)
			v1BatchGroup.POST("/task", v1.PostOperateFileOrDir) // file/operate
			v1BatchGroup.GET("", v1.GetDownloadFile)
		}
		v1ImageGroup := v1Group.Group("/image")
		v1ImageGroup.Use(forceAuthMiddleware()) // Enforce auth even from localhost
		{
			v1ImageGroup.GET("", v1.GetFileImage)
		}
		v1SambaGroup := v1Group.Group("/samba")
		v1SambaGroup.Use()
		{
			v1ConnectionsGroup := v1SambaGroup.Group("/connections")
			v1ConnectionsGroup.Use()
			{
				v1ConnectionsGroup.GET("", v1.GetSambaConnectionsList)
				v1ConnectionsGroup.POST("", v1.PostSambaConnectionsCreate)
				v1ConnectionsGroup.DELETE("/:id", v1.DeleteSambaConnections)
			}
			v1SharesGroup := v1SambaGroup.Group("/shares")
			v1SharesGroup.Use()
			{
				v1SharesGroup.GET("", v1.GetSambaSharesList)
				v1SharesGroup.POST("", v1.PostSambaSharesCreate)
				v1SharesGroup.DELETE("/:id", v1.DeleteSambaShares)
				v1SharesGroup.PUT("/:id/pause", v1.PauseSambaShare)
				v1SharesGroup.PUT("/:id/resume", v1.ResumeSambaShare)
				v1SharesGroup.GET("/status", v1.GetSambaStatus)
			}
		}
		v1NotifyGroup := v1Group.Group("/notify")
		v1NotifyGroup.Use()
		{
			v1NotifyGroup.POST("/:path", v1.PostNotifyMessage)
			// merge to system
			v1NotifyGroup.POST("/system_status", v1.PostSystemStatusNotify)
		}

		v1OtherGroup := v1Group.Group("/other")
		v1OtherGroup.Use()
		{
			v1OtherGroup.GET("/search", v1.GetSearchResult)
		}
		v1ZerotierGroup := v1Group.Group("/zt")
		v1ZerotierGroup.Use()
		{
			v1ZerotierGroup.Any("/*url", v1.ZerotierProxy)
		}
	}

	return e
}
