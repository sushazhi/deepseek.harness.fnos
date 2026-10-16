package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// DshAccountStatus DeepSeek 账号状态数据结构
type DshAccountStatus struct {
	Status        string `json:"status"` // signed-out 或 credential-stored
	Name          string `json:"name,omitempty"`
	Contact       string `json:"contact,omitempty"`
	AvatarURL     string `json:"avatar_url,omitempty"`
	Currency      string `json:"currency,omitempty"`
	Balance       string `json:"balance,omitempty"`
	BonusBalance  string `json:"bonus_balance,omitempty"`
	UsageURL      string `json:"usage_url,omitempty"`
	TopUpURL      string `json:"top_up_url,omitempty"`
	WaitingSignIn bool   `json:"waiting_sign_in"`
	AuthorizeURL  string `json:"authorize_url,omitempty"`
}

// defaultClientMetadata 默认客户端元数据
func defaultClientMetadata() map[string]any {
	return map[string]any{
		"version":               globalAppVer,
		"locale":                "zh",
		"timezoneOffsetSeconds": 28800,
	}
}

// generateUUID 生成标准格式的 UUID 字符串
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// callDshRpc 向 DSH 本地服务发起 JSON-RPC 调用
func callDshRpc(method string, payload any) (json.RawMessage, error) {
	serverPort := GetConfig().GetServerPort()
	apiPath := method
	if !strings.HasPrefix(apiPath, "/") {
		apiPath = "/" + apiPath
	}
	targetURL := fmt.Sprintf("http://127.0.0.1:%d/api%s", serverPort, apiPath)

	// DSH 要求 payload 必须且仅包含一个 plain-object args 字段
	var payloadMap map[string]any
	if m, ok := payload.(map[string]any); ok {
		if innerArgs, hasArgs := m["args"]; hasArgs && len(m) == 1 {
			if _, isMap := innerArgs.(map[string]any); isMap {
				payloadMap = m
			}
		}
	}
	if payloadMap == nil {
		if payload == nil {
			payloadMap = map[string]any{"args": map[string]any{}}
		} else {
			payloadMap = map[string]any{"args": payload}
		}
	}

	reqEnvelope := map[string]any{
		"type":    "client-request",
		"rpcId":   generateUUID(),
		"method":  method,
		"payload": payloadMap,
	}
	bodyBytes, err := json.Marshal(reqEnvelope)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", fmt.Sprintf("127.0.0.1:%d", serverPort))
	req.Header.Set("Origin", fmt.Sprintf("http://127.0.0.1:%d", serverPort))
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	// 注入服务端代持的 DSH 会话凭据
	if session := GetDshSessionCookie(); session != "" {
		req.Header.Set("Cookie", session)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 若未通过鉴权，刷新会话凭据重试一次
	if resp.StatusCode == http.StatusUnauthorized {
		InvalidateDshSession()
		if newSession := GetDshSessionCookie(); newSession != "" {
			reqRetry, _ := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(bodyBytes))
			reqRetry.Header = req.Header.Clone()
			reqRetry.Header.Set("Cookie", newSession)
			if retryResp, retryErr := client.Do(reqRetry); retryErr == nil {
				_ = resp.Body.Close()
				resp = retryResp
				defer resp.Body.Close()
			}
		}
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DSH 响应异常 (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}

	var resEnvelope struct {
		Type   string `json:"type"`
		Result struct {
			Ok    bool            `json:"ok"`
			Value json.RawMessage `json:"value"`
			Error *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBytes, &resEnvelope); err != nil {
		return nil, fmt.Errorf("解析 DSH 响应失败: %w", err)
	}

	if !resEnvelope.Result.Ok {
		msg := "DSH 调用失败"
		if resEnvelope.Result.Error != nil {
			msg = resEnvelope.Result.Error.Message
		}
		return nil, fmt.Errorf("%s", msg)
	}

	return resEnvelope.Result.Value, nil
}

// handleAccountStatus 查询当前 DeepSeek 账号状态与详情
func handleAccountStatus(c *gin.Context) {
	status := DshAccountStatus{
		Status: "signed-out",
	}

	// 1. 查询基础账号状态
	rawState, err := callDshRpc("account/getState", map[string]any{})
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取账号状态失败: "+err.Error())
		return
	}

	var stateData struct {
		Status string `json:"status"`
		Links  struct {
			UsageURL string `json:"usageUrl"`
			TopUpURL string `json:"topUpUrl"`
		} `json:"links"`
		Attempt *struct {
			ID           string `json:"id"`
			Phase        string `json:"phase"`
			AuthorizeURL string `json:"authorizeUrl"`
		} `json:"attempt"`
	}
	_ = json.Unmarshal(rawState, &stateData)

	status.Status = stateData.Status
	status.UsageURL = stateData.Links.UsageURL
	status.TopUpURL = stateData.Links.TopUpURL

	if stateData.Attempt != nil && (stateData.Attempt.Phase == "waiting-browser" || stateData.Attempt.Phase == "initializing") {
		status.WaitingSignIn = true
		status.AuthorizeURL = stateData.Attempt.AuthorizeURL
	}

	// 2. 若已登录，尝试查询昵称与余额
	if status.Status == "credential-stored" {
		clientMeta := defaultClientMetadata()

		// 查询用户个人资料
		if rawProfile, err := callDshRpc("account/getProfile", map[string]any{"client": clientMeta}); err == nil {
			var profileOutcome struct {
				Status string `json:"status"`
				Value  struct {
					Name      string `json:"name"`
					Contact   string `json:"contact"`
					AvatarURL string `json:"avatarUrl"`
				} `json:"value"`
			}
			if json.Unmarshal(rawProfile, &profileOutcome) == nil && profileOutcome.Status == "ready" {
				status.Name = profileOutcome.Value.Name
				status.Contact = profileOutcome.Value.Contact
				status.AvatarURL = profileOutcome.Value.AvatarURL
			}
		} else {
			LogWarning("[账号] 查询个人资料失败: %v", err)
		}

		// 查询用户余额详情
		if rawBalance, err := callDshRpc("account/getBalance", map[string]any{"client": clientMeta}); err == nil {
			var balanceOutcome struct {
				Status string `json:"status"`
				Value  []struct {
					Currency string `json:"currency"`
					Balance  string `json:"balance"`
				} `json:"value"`
				BonusWallets []struct {
					Currency string `json:"currency"`
					Balance  string `json:"balance"`
				} `json:"bonusWallets"`
			}
			if json.Unmarshal(rawBalance, &balanceOutcome) == nil && balanceOutcome.Status == "ready" {
				if len(balanceOutcome.Value) > 0 {
					status.Currency = balanceOutcome.Value[0].Currency
					status.Balance = balanceOutcome.Value[0].Balance
				}
				if len(balanceOutcome.BonusWallets) > 0 {
					status.BonusBalance = balanceOutcome.BonusWallets[0].Balance
				}
			}
		} else {
			LogWarning("[账号] 查询余额失败: %v", err)
		}
	}

	OK(c, status)
}

// handleAccountLoginStart 发起 DeepSeek 官方账号登录授权
func handleAccountLoginStart(c *gin.Context) {
	serverPort := GetConfig().GetServerPort()
	payload := map[string]any{
		"args": map[string]any{
			"client":         defaultClientMetadata(),
			"callbackOrigin": fmt.Sprintf("http://127.0.0.1:%d", serverPort),
			"loginSource":    "web",
		},
	}

	rawValue, err := callDshRpc("account/startSignIn", payload)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "发起登录失败: "+err.Error())
		return
	}

	var result struct {
		Status  string `json:"status"`
		Attempt *struct {
			ID           string `json:"id"`
			Phase        string `json:"phase"`
			AuthorizeURL string `json:"authorizeUrl"`
		} `json:"attempt"`
	}
	if err := json.Unmarshal(rawValue, &result); err != nil {
		Fail(c, http.StatusInternalServerError, "解析授权响应失败: "+err.Error())
		return
	}

	authURL := ""
	attemptID := ""
	phase := ""
	if result.Attempt != nil {
		authURL = result.Attempt.AuthorizeURL
		attemptID = result.Attempt.ID
		phase = result.Attempt.Phase
	}

	// DSH 创建 Attempt 后通常有短暂异步初始化（phase=initializing），此时 authorizeUrl 尚未填充
	// 在 3 秒内每隔 250ms 轮询 getState 等待官方授权链接就绪，避免前端首次调用落空
	if authURL == "" {
		for i := 0; i < 12; i++ {
			time.Sleep(250 * time.Millisecond)
			rawState, stateErr := callDshRpc("account/getState", map[string]any{})
			if stateErr == nil {
				var st struct {
					Attempt *struct {
						ID           string `json:"id"`
						Phase        string `json:"phase"`
						AuthorizeURL string `json:"authorizeUrl"`
					} `json:"attempt"`
				}
				if json.Unmarshal(rawState, &st) == nil && st.Attempt != nil && st.Attempt.AuthorizeURL != "" {
					authURL = st.Attempt.AuthorizeURL
					attemptID = st.Attempt.ID
					phase = st.Attempt.Phase
					break
				}
			}
		}
	}

	if authURL == "" {
		LogWarning("[账号] 未获取到授权链接: 原始响应=%s", string(rawValue))
		Fail(c, http.StatusInternalServerError, "未获取到授权链接")
		return
	}

	LogInfo("[账号] 获取授权链接成功: attempt_id=%s, phase=%s", attemptID, phase)
	OK(c, gin.H{
		"authorize_url": authURL,
		"attempt_id":    attemptID,
		"phase":         phase,
	})
}

// handleAccountLoginConfirm 接收回调地址并向 DSH 内部完成兑换
func handleAccountLoginConfirm(c *gin.Context) {
	var req struct {
		URL   string `json:"url"`
		Code  string `json:"code"`
		State string `json:"state"`
	}
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&req)
	}

	rawInput := strings.TrimSpace(req.URL)
	codeVal := strings.TrimSpace(req.Code)
	stateVal := strings.TrimSpace(req.State)

	// 若提供了 URL，从中解析 code 与 state
	if rawInput != "" {
		// 补齐 scheme 方便解析
		parsedURLString := rawInput
		if !strings.Contains(parsedURLString, "://") {
			parsedURLString = "http://" + strings.TrimPrefix(parsedURLString, "/")
		}
		if u, err := url.Parse(parsedURLString); err == nil {
			if cParam := u.Query().Get("code"); cParam != "" {
				codeVal = cParam
			}
			if sParam := u.Query().Get("state"); sParam != "" {
				stateVal = sParam
			}
		}
		// 若从 URL 解析未成功，尝试直接解析 query 串或 hash 后的参数片段
		if codeVal == "" || stateVal == "" {
			qStr := rawInput
			if idx := strings.Index(qStr, "?"); idx != -1 {
				qStr = qStr[idx+1:]
			} else if idx := strings.Index(qStr, "#"); idx != -1 {
				qStr = qStr[idx+1:]
			}
			if vals, err := url.ParseQuery(qStr); err == nil {
				if cParam := vals.Get("code"); cParam != "" && codeVal == "" {
					codeVal = cParam
				}
				if sParam := vals.Get("state"); sParam != "" && stateVal == "" {
					stateVal = sParam
				}
			}
		}
	}

	if codeVal == "" || stateVal == "" {
		Fail(c, http.StatusBadRequest, "未能从输入内容中解析出有效的授权码 (code) 与校验参数 (state)")
		return
	}

	// 向 DSH 发起本地回调兑换
	serverPort := GetConfig().GetServerPort()
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/oauth/callback?code=%s&state=%s",
		serverPort, url.QueryEscape(codeVal), url.QueryEscape(stateVal))

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 阻断 302 重定向
		},
	}

	reqGet, err := http.NewRequest(http.MethodGet, callbackURL, nil)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "创建回调请求失败: "+err.Error())
		return
	}
	reqGet.Header.Set("Host", fmt.Sprintf("127.0.0.1:%d", serverPort))
	if session := GetDshSessionCookie(); session != "" {
		reqGet.Header.Set("Cookie", session)
	}

	resp, err := client.Do(reqGet)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "连接 DSH 回调接口失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusSeeOther {
		LogInfo("[账号] DeepSeek 官方账号授权兑换成功")
		OK(c, gin.H{"message": "登录授权成功"})
		return
	}

	if resp.StatusCode == http.StatusGone {
		Fail(c, http.StatusGone, "登录会话已过期或已被使用，请重新发起登录")
		return
	}

	respBytes, _ := io.ReadAll(resp.Body)
	Fail(c, http.StatusBadRequest, fmt.Sprintf("授权兑换未成功 (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(respBytes))))
}

// handleAccountLogout 注销当前 DeepSeek 账号
func handleAccountLogout(c *gin.Context) {
	clientMeta := defaultClientMetadata()
	_, err := callDshRpc("account/signOut", map[string]any{"client": clientMeta})
	if err != nil {
		Fail(c, http.StatusInternalServerError, "退出登录失败: "+err.Error())
		return
	}

	LogInfo("[账号] DeepSeek 官方账号已退出登录")
	OK(c, gin.H{"message": "已成功退出登录"})
}
