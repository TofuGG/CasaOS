package v1

import (
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS/model"
	"github.com/IceWhaleTech/CasaOS/pkg/utils/common_err"
	"github.com/IceWhaleTech/CasaOS/service"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func GetSearchResult(ctx echo.Context) error {
	json := make(map[string]string)
	ctx.Bind(&json)
	url := json["url"]

	if url == "" {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: "key is empty"})
	}
	// data, err := service.MyService.Other().Search(key)
	data, err := service.MyService.Other().AgentSearch(url)
	if err != nil {
		logger.Error("agent search failed", zap.Error(err), zap.String("url", url))
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR)})
	}

	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}
