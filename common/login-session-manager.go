package common

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type LoginSessionStatus string

const (
	SessionStatusPending LoginSessionStatus = "pending"
	SessionStatusScanned LoginSessionStatus = "scanned"
	SessionStatusSuccess LoginSessionStatus = "success"
	SessionStatusExpired LoginSessionStatus = "expired"
)

type WeChatUserInfo struct {
	OpenID string `json:"openid"`
}

type LoginSession struct {
	LoginToken string             `json:"login_token"` // 前端查询令牌
	SceneID    string             `json:"scene_id"`    // 微信场景值
	Status     LoginSessionStatus `json:"status"`      // 会话状态
	WeChatID   string             `json:"wechat_id"`   // 微信用户ID
	AuthCode   string             `json:"auth_code"`   // 授权给clinx的代码
	UserInfo   *WeChatUserInfo    `json:"user_info"`   // 用户详细信息
	CreatedAt  time.Time          `json:"created_at"`  // 创建时间
	ExpiredAt  time.Time          `json:"expired_at"`  // 过期时间
}

type LoginSessionManager struct {
	sessions map[string]*LoginSession // key: login_token
	scenes   map[string]string        // key: scene_id, value: login_token
	mutex    sync.RWMutex
	cleanup  *time.Ticker
}

var sessionManager = &LoginSessionManager{
	sessions: make(map[string]*LoginSession),
	scenes:   make(map[string]string),
	cleanup:  time.NewTicker(1 * time.Minute),
}

func init() {
	go sessionManager.startCleanup()
}

// 生成唯一登录令牌
func (m *LoginSessionManager) generateLoginToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// 生成场景值
func (m *LoginSessionManager) generateSceneID() string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("login_%d_%s", timestamp, randomStr)
}

// 创建登录会话
func (m *LoginSessionManager) CreateSession() *LoginSession {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	loginToken := m.generateLoginToken()
	sceneID := m.generateSceneID()
	authCode := m.generateLoginToken() // 生成授权码

	session := &LoginSession{
		LoginToken: loginToken,
		SceneID:    sceneID,
		Status:     SessionStatusPending,
		AuthCode:   authCode,
		CreatedAt:  time.Now(),
		ExpiredAt:  time.Now().Add(10 * time.Minute), // 10分钟过期
	}

	m.sessions[loginToken] = session
	m.scenes[sceneID] = loginToken

	SysLog(fmt.Sprintf("Created login session: token=%s, scene=%s", loginToken, sceneID))
	return session
}

// 通过登录令牌获取会话
func (m *LoginSessionManager) GetSession(loginToken string) *LoginSession {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, exists := m.sessions[loginToken]
	if !exists {
		return nil
	}

	if session.ExpiredAt.Before(time.Now()) {
		delete(m.sessions, loginToken)
		delete(m.scenes, session.SceneID)
		return nil
	}

	return session
}

// 通过场景值获取会话
func (m *LoginSessionManager) GetSessionByScene(sceneID string) *LoginSession {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	loginToken, exists := m.scenes[sceneID]
	if !exists {
		return nil
	}

	session, exists := m.sessions[loginToken]
	if !exists || session.ExpiredAt.Before(time.Now()) {
		delete(m.scenes, sceneID)
		if exists {
			delete(m.sessions, loginToken)
		}
		return nil
	}

	return session
}

// 更新会话状态（用户扫码后）
func (m *LoginSessionManager) UpdateSessionByScene(sceneID, wechatID string, userInfo *WeChatUserInfo) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	loginToken, exists := m.scenes[sceneID]
	if !exists {
		return false
	}

	session, sessionExists := m.sessions[loginToken]
	if !sessionExists {
		return false
	}

	session.WeChatID = wechatID
	session.UserInfo = userInfo
	session.Status = SessionStatusSuccess

	SysLog(fmt.Sprintf("Updated login session: wechat_id=%s, status=success", wechatID))
	return true
}

// 清理过期会话
func (m *LoginSessionManager) startCleanup() {
	for range m.cleanup.C {
		m.mutex.Lock()
		now := time.Now()

		for loginToken, session := range m.sessions {
			if session.ExpiredAt.Before(now) {
				session.Status = SessionStatusExpired
				delete(m.sessions, loginToken)
				delete(m.scenes, session.SceneID)
			}
		}

		SysLog(fmt.Sprintf("Cleanup completed, active sessions: %d", len(m.sessions)))
		m.mutex.Unlock()
	}
}

// 获取活跃会话数量
func (m *LoginSessionManager) GetActiveSessionCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.sessions)
}

// 通过授权码查找微信用户ID（全局函数）
func FindWeChatIDByAuthCode(authCode string) string {
	return sessionManager.findWeChatIDByAuthCode(authCode)
}

// 内部方法：通过授权码查找微信用户ID
func (m *LoginSessionManager) findWeChatIDByAuthCode(authCode string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, session := range m.sessions {
		if session.AuthCode == authCode && session.Status == SessionStatusSuccess {
			return session.WeChatID
		}
	}

	return ""
}

// 导出全局访问器
func GetSessionManager() *LoginSessionManager {
	return sessionManager
}
