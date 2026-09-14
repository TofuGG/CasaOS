package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	http2 "github.com/IceWhaleTech/CasaOS-Common/utils/http"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-Common/utils/port"
	"github.com/IceWhaleTech/CasaOS/common"
	"github.com/IceWhaleTech/CasaOS/model"
	"github.com/IceWhaleTech/CasaOS/pkg/config"
	"github.com/IceWhaleTech/CasaOS/pkg/hwstatus"
	"github.com/IceWhaleTech/CasaOS/pkg/utils"
	"github.com/IceWhaleTech/CasaOS/pkg/utils/common_err"
	"github.com/IceWhaleTech/CasaOS/pkg/utils/version"
	"github.com/IceWhaleTech/CasaOS/service"
	model2 "github.com/IceWhaleTech/CasaOS/service/model"
	"github.com/IceWhaleTech/CasaOS/types"
	"github.com/labstack/echo/v4"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// @Summary check version
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/version/check [get]
func GetSystemCheckVersion(ctx echo.Context) error {
	need, version := version.IsNeedUpdate(service.MyService.Casa().GetCasaosVersion())
	if need {
		installLog := model2.AppNotify{}
		installLog.State = 0
		installLog.Message = "New version " + version.Version + " is ready, ready to upgrade"
		installLog.Type = types.NOTIFY_TYPE_NEED_CONFIRM
		installLog.CreatedAt = strconv.FormatInt(time.Now().Unix(), 10)
		installLog.UpdatedAt = strconv.FormatInt(time.Now().Unix(), 10)
		installLog.Name = "CasaOS System"
		service.MyService.Notify().AddLog(installLog)
	}
	data := make(map[string]interface{}, 3)
	data["need_update"] = need
	data["version"] = version
	data["current_version"] = common.VERSION
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

// @Summary 系统信息
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/update [post]
func SystemUpdate(ctx echo.Context) error {
	need, version := version.IsNeedUpdate(service.MyService.Casa().GetCasaosVersion())
	if need {
		service.MyService.System().UpdateSystemVersion(version.Version)
	}
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

// @Summary  get logs
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/error/logs [get]
func GetCasaOSErrorLogs(ctx echo.Context) error {
	line, _ := strconv.Atoi(utils.DefaultQuery(ctx, "line", "100"))
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: service.MyService.System().GetCasaOSLogs(line)})
}

// 系统配置
func GetSystemConfigDebug(ctx echo.Context) error {
	array := service.MyService.System().GetSystemConfigDebug()
	disk := service.MyService.System().GetDiskInfo()
	sys := service.MyService.System().GetSysInfo()
	version := service.MyService.Casa().GetCasaosVersion()
	var bugContent string = fmt.Sprintf(`
	 - OS: %s
	 - CasaOS Version: %s
	 - Disk Total: %v 
	 - Disk Used: %v 
	 - System Info: %s
	 - Remote Version: %s
	 - Browser: $Browser$ 
	 - Version: $Version$
`, sys.OS, common.VERSION, disk.Total>>20, disk.Used>>20, array, version.Version)

	//	array = append(array, fmt.Sprintf("disk,total:%v,used:%v,UsedPercent:%v", disk.Total>>20, disk.Used>>20, disk.UsedPercent))

	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: bugContent})
}

// @Summary get casaos server port
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/port [get]
func GetCasaOSPort(ctx echo.Context) error {
	return ctx.JSON(common_err.SUCCESS,
		model.Result{
			Success: common_err.SUCCESS,
			Message: common_err.GetMsg(common_err.SUCCESS),
			Data:    config.ServerInfo.HttpPort,
		})
}

// @Summary edit casaos server port
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Param port json string true "port"
// @Success 200 {string} string "ok"
// @Router /sys/port [put]
func PutCasaOSPort(ctx echo.Context) error {
	json := make(map[string]string)
	ctx.Bind(&json)
	portStr := json["port"]
	portNumber, err := strconv.Atoi(portStr)
	if err != nil {
		return ctx.JSON(common_err.SERVICE_ERROR,
			model.Result{
				Success: common_err.SERVICE_ERROR,
				Message: common_err.GetMsg(common_err.INVALID_PARAMS),
			})
	}

	isAvailable := port.IsPortAvailable(portNumber, "tcp")
	if !isAvailable {
		return ctx.JSON(common_err.SERVICE_ERROR,
			model.Result{
				Success: common_err.PORT_IS_OCCUPIED,
				Message: common_err.GetMsg(common_err.PORT_IS_OCCUPIED),
			})
	}
	service.MyService.System().UpSystemPort(strconv.Itoa(portNumber))
	return ctx.JSON(common_err.SUCCESS,
		model.Result{
			Success: common_err.SUCCESS,
			Message: common_err.GetMsg(common_err.SUCCESS),
		})
}

// @Summary active killing casaos
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/restart [post]
func PostKillCasaOS(ctx echo.Context) error {
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

// @Summary get system hardware info
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/hardware/info [get]
func GetSystemHardwareInfo(ctx echo.Context) error {
	data := make(map[string]string, 1)
	data["drive_model"] = service.MyService.System().GetDeviceTree()
	data["arch"] = runtime.GOARCH

	if cpu := service.MyService.System().GetCpuInfo(); len(cpu) > 0 {
		return ctx.JSON(common_err.SUCCESS,
			model.Result{
				Success: common_err.SUCCESS,
				Message: common_err.GetMsg(common_err.SUCCESS),
				Data:    data,
			})
	}
	return nil
}

// @Summary system utilization
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/utilization [get]
func GetSystemUtilization(ctx echo.Context) error {
	data := make(map[string]interface{})
	cpu := service.MyService.System().GetCpuPercent()
	num := service.MyService.System().GetCpuCoreNum()
	cpuModel := "arm"
	if cpu := service.MyService.System().GetCpuInfo(); len(cpu) > 0 {
		if strings.Count(strings.ToLower(strings.TrimSpace(cpu[0].ModelName)), "intel") > 0 {
			cpuModel = "intel"
		} else if strings.Count(strings.ToLower(strings.TrimSpace(cpu[0].ModelName)), "amd") > 0 {
			cpuModel = "amd"
		}
	}
	cpuData := make(map[string]interface{})
	cpuData["percent"] = cpu
	cpuData["num"] = num
	cpuData["temperature"] = service.MyService.System().GetCPUTemperature()
	cpuData["power"] = service.MyService.System().GetCPUPower()
	cpuData["model"] = cpuModel

	data["cpu"] = cpuData
	data["mem"] = service.MyService.System().GetMemInfo()

	// 拼装网络信息
	netList := service.MyService.System().GetNetInfo()
	newNet := []model.IOCountersStat{}
	nets := service.MyService.System().GetNet(true)
	for _, n := range netList {
		for _, netCardName := range nets {
			if n.Name == netCardName {
				item := *(*model.IOCountersStat)(unsafe.Pointer(&n))
				item.State = strings.TrimSpace(service.MyService.System().GetNetState(n.Name))
				item.Time = time.Now().UnixMilli()
				newNet = append(newNet, item)
				break
			}
		}
	}

	data["net"] = newNet

	// Provide disk + USB info on the HTTP endpoint as well. Upstream delivers
	// these only via the MessageBus socket push ("casaos:system:utilization"),
	// so without a MessageBus the Disks widget had nothing to render — and
	// mounted() crashed on the undefined value. The widget reads sys_disk as an
	// object here (the socket variant arrives as a JSON string which the UI
	// JSON.parses itself), so both transports are covered.
	if diskInfo := service.MyService.System().GetDiskInfo(); diskInfo != nil {
		data["sys_disk"] = map[string]interface{}{
			"size":   diskInfo.Total,
			"used":   diskInfo.Used,
			"avail":  diskInfo.Free,
			"health": "Healthy",
		}
	}
	data["sys_usb"] = []interface{}{}
	systemMap := service.MyService.Notify().GetSystemTempMap()
	systemMap.Range(func(key, value interface{}) bool {
		data[key.(string)] = value
		return true
	})
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

// GetSystemUtilizationInterval returns the current dashboard hardware-status
// push interval in milliseconds.
func GetSystemUtilizationInterval(ctx echo.Context) error {
	return ctx.JSON(common_err.SUCCESS, model.Result{
		Success: common_err.SUCCESS,
		Message: common_err.GetMsg(common_err.SUCCESS),
		Data:    map[string]int{"interval_ms": hwstatus.GetInterval()},
	})
}

// PutSystemUtilizationInterval updates the dashboard hardware-status push
// interval. The value is clamped to [250, 5000] ms (0 → 5000) and persisted
// to the CasaOS config file.
func PutSystemUtilizationInterval(ctx echo.Context) error {
	payload := &struct {
		IntervalMs int `json:"interval_ms"`
	}{}
	if err := ctx.Bind(payload); err != nil {
		return ctx.JSON(http.StatusBadRequest, model.Result{
			Success: common_err.SERVICE_ERROR,
			Message: common_err.GetMsg(common_err.INVALID_PARAMS),
		})
	}
	if payload.IntervalMs < 0 {
		return ctx.JSON(http.StatusBadRequest, model.Result{
			Success: common_err.SERVICE_ERROR,
			Message: common_err.GetMsg(common_err.INVALID_PARAMS),
		})
	}

	ms := hwstatus.SetInterval(payload.IntervalMs)
	service.MyService.System().UpHardwareStatusInterval(ms)

	return ctx.JSON(common_err.SUCCESS, model.Result{
		Success: common_err.SUCCESS,
		Message: common_err.GetMsg(common_err.SUCCESS),
		Data:    map[string]int{"interval_ms": ms},
	})
}

// @Summary get cpu info
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/cpu [get]
func GetSystemCupInfo(ctx echo.Context) error {
	cpu := service.MyService.System().GetCpuPercent()
	num := service.MyService.System().GetCpuCoreNum()
	data := make(map[string]interface{})
	data["percent"] = cpu
	data["num"] = num
	return ctx.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

// @Summary get mem info
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/mem [get]
func GetSystemMemInfo(ctx echo.Context) error {
	mem := service.MyService.System().GetMemInfo()
	return ctx.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: mem})
}

// @Summary get disk info
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/disk [get]
func GetSystemDiskInfo(ctx echo.Context) error {
	disk := service.MyService.System().GetDiskInfo()
	return ctx.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: disk})
}

// @Summary get Net info
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/net [get]
func GetSystemNetInfo(ctx echo.Context) error {
	netList := service.MyService.System().GetNetInfo()
	newNet := []model.IOCountersStat{}
	for _, n := range netList {
		for _, netCardName := range service.MyService.System().GetNet(true) {
			if n.Name == netCardName {
				item := *(*model.IOCountersStat)(unsafe.Pointer(&n))
				item.State = strings.TrimSpace(service.MyService.System().GetNetState(n.Name))
				item.Time = time.Now().Unix()
				newNet = append(newNet, item)
				break
			}
		}
	}

	return ctx.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: newNet})
}

// isPrivateOrReservedIP checks if an IP address is in a private, reserved, or link-local range.
// Blocks: loopback (127.0.0.0/8, ::1), RFC1918 (10/8, 172.16/12, 192.168/16),
// link-local (169.254/16, fe80::/10), cloud metadata (169.254.169.254), and IPv6 ULA (fc00::/7).
func isPrivateOrReservedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	// Cloud metadata endpoint
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return true
	}
	// RFC1918
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 10 {
			return true
		}
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		return false
	}
	// IPv6 ULA (fc00::/7)
	if len(ip) == net.IPv6len && ip[0]&0xfe == 0xfc {
		return true
	}
	return false
}

// isURLSafeToProxy checks if a URL targets a safe (non-private) host.
// Resolves the hostname and blocks private/reserved IPs to prevent SSRF attacks.
func isURLSafeToProxy(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http/https URLs are allowed")
	}
	hostname := u.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}
	// Quick block for obviously private hostnames
	switch strings.ToLower(hostname) {
	case "localhost", "0.0.0.0", "metadata.google.internal":
		return fmt.Errorf("URL targets a blocked host")
	}
	// Resolve hostname and check all resulting IPs
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("could not resolve hostname")
	}
	if len(ips) == 0 {
		return fmt.Errorf("hostname resolved to no addresses")
	}
	for _, ip := range ips {
		if isPrivateOrReservedIP(ip) {
			return fmt.Errorf("URL targets a private/reserved network address")
		}
	}
	return nil
}

func GetSystemProxy(ctx echo.Context) error {
	rawURL := ctx.QueryParam("url")
	if err := isURLSafeToProxy(rawURL); err != nil {
		return ctx.JSON(http.StatusBadRequest, model.Result{
			Success: common_err.INVALID_PARAMS,
			Message: "Prohibited target: " + err.Error(),
		})
	}
	resp, err := http2.Get(rawURL, 30*time.Second)
	if err != nil {
		return ctx.JSON(http.StatusBadGateway, model.Result{
			Success: common_err.SERVICE_ERROR,
			Message: common_err.GetMsg(common_err.SERVICE_ERROR),
		})
	}
	defer resp.Body.Close()
	// Only copy safe headers from response
	for k, v := range resp.Header {
		if len(v) > 0 {
			ctx.Response().Header().Set(k, v[0])
		}
	}
	rda, _ := io.ReadAll(resp.Body)
	ctx.Response().WriteHeader(resp.StatusCode)
	ctx.Response().Write(rda)
	return nil
}

func PutSystemState(ctx echo.Context) error {
	state := ctx.Param("state")
	var err error
	switch strings.ToLower(state) {
	case "off":
		err = service.MyService.System().SystemShutdown()
	case "restart":
		err = service.MyService.System().SystemReboot()
	}
	if err != nil {
		// Log the real error server-side; do not expose internal detail
		// (e.g. missing binaries/paths) to API clients.
		logger.Error("power action failed", zap.Error(err), zap.String("state", state))
		return ctx.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: "power action failed"})
	}
	return ctx.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: "The operation will be completed shortly."})
}

// @Summary 获取一个可用端口
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Param  type query string true "端口类型 udp/tcp"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/getport [get]
func GetPort(ctx echo.Context) error {
	t := utils.DefaultQuery(ctx, "type", "tcp")
	var p int
	ok := true
	for ok {
		p, _ = port.GetAvailablePort(t)
		ok = !port.IsPortAvailable(p, t)
	}
	// @tiger 这里最好封装成 {'port': ...} 的形式，来体现出参的上下文
	return ctx.JSON(common_err.SUCCESS, &model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: p})
}

// @Summary 检查端口是否可用
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Param  port path int true "端口号"
// @Param  type query string true "端口类型 udp/tcp"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/check/{port} [get]
func PortCheck(ctx echo.Context) error {
	p, _ := strconv.Atoi(ctx.Param("port"))
	t := utils.DefaultQuery(ctx, "type", "tcp")
	return ctx.JSON(common_err.SUCCESS, &model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: port.IsPortAvailable(p, t)})
}

func GetSystemEntry(ctx echo.Context) error {
	entry := service.MyService.System().GetSystemEntry()
	str := json.RawMessage(entry)
	if !gjson.ValidBytes(str) {
		return ctx.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: entry, Data: json.RawMessage("[]")})
	}
	return ctx.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: str})
}
