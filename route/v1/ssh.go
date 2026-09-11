package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	sshHelper "github.com/IceWhaleTech/CasaOS-Common/utils/ssh"
	"github.com/IceWhaleTech/CasaOS/pkg/utils"
	"github.com/labstack/echo/v4"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"

	modelCommon "github.com/IceWhaleTech/CasaOS-Common/model"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	CheckOrigin:      func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // non-browser clients
			}
			return strings.HasPrefix(origin, "http://127.0.0.1") ||
				strings.HasPrefix(origin, "http://localhost") ||
				strings.HasPrefix(origin, "https://127.0.0.1") ||
				strings.HasPrefix(origin, "https://localhost")
		},
	HandshakeTimeout: time.Duration(time.Second * 5),
}

func PostSshLogin(ctx echo.Context) error {
	j := make(map[string]string)
	ctx.Bind(&j)
	userName := j["username"]
	password := j["password"]
	port := j["port"]
	if userName == "" || password == "" || port == "" {
		return ctx.JSON(common_err.CLIENT_ERROR, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: "Username or password or port is empty"})
	}
	_, err := sshHelper.NewSshClient(userName, password, port)
	if err != nil {
		logger.Error("connect ssh error", zap.Any("error", err))
		return ctx.JSON(common_err.CLIENT_ERROR, modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: common_err.GetMsg(common_err.CLIENT_ERROR), Data: "Please check if the username and port are correct, and make sure that ssh server is installed."})
	}
	return ctx.JSON(common_err.SUCCESS, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

func WsSsh(ctx echo.Context) error {
	_, e := exec.LookPath("ssh")
	if e != nil {
		return ctx.JSON(common_err.SERVICE_ERROR, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: "ssh server not found"})
	}

	wsConn, err := upgrader.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
	if err != nil {
		logger.Error("websocket upgrade failed", zap.Error(err))
		return nil
	}
	defer wsConn.Close()

	// SECURITY: SSH credentials must never be sent in the URL query string —
	// they would land in access logs, proxies and browser history. Read them
	// from the first WebSocket frame instead (sent as JSON by the frontend).
	wsConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, credsData, err := wsConn.ReadMessage()
	if err != nil {
		wsConn.WriteMessage(websocket.TextMessage, []byte("failed to read credentials"))
		return nil
	}
	wsConn.SetReadDeadline(time.Time{})

	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Port     string `json:"port"`
	}
	if err := json.Unmarshal(credsData, &creds); err != nil || creds.Username == "" || creds.Password == "" || creds.Port == "" {
		wsConn.WriteMessage(websocket.TextMessage, []byte("username or password or port is empty"))
		return nil
	}

	logBuff := new(bytes.Buffer)

	quitChan := make(chan bool, 3)
	var loginAttempts int
	const maxLoginAttempts = 3
	cols, _ := strconv.Atoi(utils.DefaultQuery(ctx, "cols", "200"))
	rows, _ := strconv.Atoi(utils.DefaultQuery(ctx, "rows", "32"))
	var client *ssh.Client
	for loginAttempts < maxLoginAttempts {
		loginAttempts++
		var err error
		client, err = sshHelper.NewSshClient(creds.Username, creds.Password, creds.Port)

		if err != nil && client == nil {
			wsConn.WriteMessage(websocket.TextMessage, []byte("Connection failed"))
			wsConn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[0m"))
			if loginAttempts >= maxLoginAttempts {
				wsConn.WriteMessage(websocket.TextMessage, []byte("Max login attempts reached"))
				break
			}
		} else {
			break
		}

	}
	if client != nil {
		defer client.Close()
	}

	ssConn, _ := sshHelper.NewSshConn(cols, rows, client)
	defer ssConn.Close()

	go ssConn.ReceiveWsMsg(wsConn, logBuff, quitChan)
	go ssConn.SendComboOutput(wsConn, quitChan)
	go ssConn.SessionWait(quitChan)

	<-quitChan
	return nil
}
