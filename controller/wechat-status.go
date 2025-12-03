package controller

import (
	"fmt"
	"net/http"

	"wechat-server/common"

	"github.com/gin-gonic/gin"
)

type LoginStatusResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Status     string                 `json:"status"`
		WeChatUser *common.WeChatUserInfo `json:"wechat_user,omitempty"`
		AuthCode   string                 `json:"auth_code,omitempty"`
	} `json:"data"`
	Message string `json:"message"`
}

func GetLoginStatus(c *gin.Context) {
	loginToken := c.Query("login_token")
	if loginToken == "" {
		c.JSON(http.StatusBadRequest, LoginStatusResponse{
			Success: false,
			Message: "缺少login_token参数",
		})
		return
	}
	session := common.GetSessionManager().GetSession(loginToken)
	if session == nil {
		c.JSON(http.StatusOK, LoginStatusResponse{
			Success: false,
			Message: "登录令牌无效或已过期",
		})
		return
	}
	response := LoginStatusResponse{
		Success: true,
		Message: "查询成功",
	}

	response.Data.Status = string(session.Status)
	if session.Status == common.SessionStatusSuccess && session.UserInfo != nil {
		response.Data.WeChatUser = session.UserInfo
		response.Data.AuthCode = session.AuthCode
	}

	c.JSON(http.StatusOK, response)

	common.SysLog(fmt.Sprintf("Login status query: token=%s, status=%s", loginToken, session.Status))
}
