package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"wechat-server/common"

	"github.com/gin-gonic/gin"
)

type CreateQRCodeResponse struct {
	Success bool `json:"success"`
	Data    struct {
		SceneID       string `json:"scene_id"`
		QRCodeURL     string `json:"qrcode_url"`
		LoginToken    string `json:"login_token"`
		ExpireSeconds int    `json:"expire_seconds"`
	} `json:"data"`
	Message string `json:"message"`
}

type WeChatQRCodeRequest struct {
	ExpireSeconds int    `json:"expire_seconds"`
	ActionName    string `json:"action_name"`
	ActionInfo    struct {
		Scene struct {
			SceneStr string `json:"scene_str"`
		} `json:"scene"`
	} `json:"action_info"`
}

type WeChatQRCodeResponse struct {
	Ticket        string `json:"ticket"`
	ExpireSeconds int    `json:"expire_seconds"`
	URL           string `json:"url"`
	ErrorCode     int    `json:"errcode,omitempty"`
	ErrorMessage  string `json:"errmsg,omitempty"`
}

func CreateLoginQRCode(c *gin.Context) {
	session := common.GetSessionManager().CreateSession()
	if session == nil {
		c.JSON(http.StatusInternalServerError, CreateQRCodeResponse{
			Success: false,
			Message: "创建登录会话失败",
		})
		return
	}
	accessToken := common.GetAccessToken()
	if accessToken == "" {
		c.JSON(http.StatusInternalServerError, CreateQRCodeResponse{
			Success: false,
			Message: "获取微信访问令牌失败",
		})
		return
	}
	qrReq := WeChatQRCodeRequest{
		ExpireSeconds: 600, // 10分钟
		ActionName:    "QR_STR_SCENE",
	}
	qrReq.ActionInfo.Scene.SceneStr = session.SceneID

	qrReqJSON, err := json.Marshal(qrReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, CreateQRCodeResponse{
			Success: false,
			Message: "构建二维码请求失败",
		})
		return
	}
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/qrcode/create?access_token=%s", accessToken)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(qrReqJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, CreateQRCodeResponse{
			Success: false,
			Message: "调用微信API失败: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	var qrResp WeChatQRCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&qrResp); err != nil {
		c.JSON(http.StatusInternalServerError, CreateQRCodeResponse{
			Success: false,
			Message: "解析微信响应失败: " + err.Error(),
		})
		return
	}
	if qrResp.ErrorCode != 0 {
		c.JSON(http.StatusInternalServerError, CreateQRCodeResponse{
			Success: false,
			Message: fmt.Sprintf("微信API错误: %s", qrResp.ErrorMessage),
		})
		return
	}
	qrcodeURL := fmt.Sprintf("https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=%s", qrResp.Ticket)
	response := CreateQRCodeResponse{
		Success: true,
		Message: "二维码创建成功",
	}
	response.Data.SceneID = session.SceneID
	response.Data.QRCodeURL = qrcodeURL
	response.Data.LoginToken = session.LoginToken
	response.Data.ExpireSeconds = qrResp.ExpireSeconds

	c.JSON(http.StatusOK, response)

	common.SysLog(fmt.Sprintf("Created login QRCode: scene=%s, token=%s, ticket=%s",
		session.SceneID, session.LoginToken, qrResp.Ticket))
}
