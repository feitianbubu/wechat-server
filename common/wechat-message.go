package common

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type WeChatMessageRequest struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgId        int64    `xml:"MsgId"`
	MsgDataId    int64    `xml:"MsgDataId"`
	Idx          int64    `xml:"Idx"`
	// 事件相关字段
	Event    string `xml:"Event,omitempty"`
	EventKey string `xml:"EventKey,omitempty"`
	Ticket   string `xml:"Ticket,omitempty"`
}

type WeChatMessageResponse struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
}

func ProcessWeChatMessage(req *WeChatMessageRequest, res *WeChatMessageResponse) {
	// 记录接收到的消息
	SysLog(fmt.Sprintf("Received WeChat message: type=%s, from=%s, event=%s, key=%s, content=%s",
		req.MsgType, req.FromUserName, req.Event, req.EventKey, req.Content))

	switch {
	case req.MsgType == "event" && (req.Event == "subscribe" || req.Event == "SCAN"):
		// 处理扫码事件（关注或已关注用户扫码）
		handleQRCodeScanEvent(req, res)

	case req.MsgType == "text":
		// 保留原有的验证码逻辑
		switch req.Content {
		case "验证码":
			handleVerificationCode(req, res)
		default:
			res.Content = "发送「验证码」获取登录验证码"
		}

	default:
		res.Content = "欢迎使用！发送「验证码」获取登录验证码"
	}
}

// 处理二维码扫码事件
func handleQRCodeScanEvent(req *WeChatMessageRequest, res *WeChatMessageResponse) {
	var sceneID string

	if req.Event == "subscribe" && strings.HasPrefix(req.EventKey, "qrscene_") {
		// 新用户扫码关注
		sceneID = strings.TrimPrefix(req.EventKey, "qrscene_")
		res.Content = "欢迎关注！登录成功，请返回网页继续操作"

	} else if req.Event == "SCAN" {
		// 已关注用户扫码
		sceneID = req.EventKey
		res.Content = "登录成功，请返回网页继续操作"

	} else {
		// 普通关注事件
		res.Content = "欢迎关注！发送「验证码」获取登录验证码，或使用扫码登录功能"
		return
	}

	// 查找对应的登录会话
	session := GetSessionManager().GetSessionByScene(sceneID)
	if session == nil {
		SysLog(fmt.Sprintf("No session found for scene: %s", sceneID))
		if req.Event == "subscribe" {
			res.Content = "欢迎关注！二维码可能已过期，请重新生成"
		}
		return
	}

	// 获取用户信息（简化版，实际可调用微信API获取）
	userInfo := &WeChatUserInfo{
		OpenID: req.FromUserName,
	}

	// 更新会话状态
	success := GetSessionManager().UpdateSessionByScene(sceneID, req.FromUserName, userInfo)
	if success {
		SysLog(fmt.Sprintf("Successfully updated login session: scene=%s, wechat=%s",
			sceneID, req.FromUserName))
	} else {
		SysLog(fmt.Sprintf("Failed to update login session: scene=%s", sceneID))
		res.Content = "登录失败，请重新扫码"
	}
}

func handleVerificationCode(req *WeChatMessageRequest, res *WeChatMessageResponse) {
	code := GenerateAllNumberVerificationCode(6)
	RegisterWeChatCodeAndID(code, req.FromUserName)
	res.Content = code
	SysLog(fmt.Sprintf("Generated verification code: %s for user: %s", code, req.FromUserName))
}
