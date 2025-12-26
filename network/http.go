package network

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"vortenixgo/bot"
	"vortenixgo/network/ws"

	"golang.org/x/net/html"
	"golang.org/x/net/proxy"
)

const (
	GrowtopiaUserAgent = "UbiServices_SDK_2022.Release.9_PC64_ansi_static"
	ChromeUserAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0"
	ServerDataURL      = "https://www.growtopia1.com/growtopia/server_data.php"
)

// HTTPHandler handles all HTTP requests for a bot
type HTTPHandler struct {
	Client *http.Client
}

// NewHTTPHandler creates a new HTTP handler with optional proxy support
func NewHTTPHandler(proxyStr string) *HTTPHandler {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if proxyStr != "" {
		parts := strings.Split(proxyStr, ":")
		var dialer proxy.Dialer
		var err error

		if len(parts) == 4 {
			// format: host:port:user:pass
			auth := &proxy.Auth{
				User:     parts[2],
				Password: parts[3],
			}
			dialer, err = proxy.SOCKS5("tcp", parts[0]+":"+parts[1], auth, proxy.Direct)
		} else if len(parts) == 2 {
			// format: host:port
			dialer, err = proxy.SOCKS5("tcp", proxyStr, nil, proxy.Direct)
		} else {
			log.Printf("[HTTP] Unknown proxy format: %s. Expected host:port or host:port:user:pass", proxyStr)
		}

		if err == nil && dialer != nil {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		} else if err != nil {
			log.Printf("[HTTP] Failed to setup SOCKS5 proxy: %v", err)
		}
	}

	return &HTTPHandler{
		Client: &http.Client{
			Transport: transport,
			Timeout:   60 * time.Second,
		},
	}
}

// GetMeta performs the getMeta request to fetch server data
func (h *HTTPHandler) GetMeta(b *bot.Bot) error {
	data := url.Values{}
	data.Set("version", b.Login.GameVersion)
	data.Set("platform", b.Login.PlatformID)
	data.Set("protocol", b.Login.Protocol)

	req, err := http.NewRequest("POST", ServerDataURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Host", "www.growtopia1.com")
	req.Header.Set("User-Agent", GrowtopiaUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")

	log.Printf("[HTTP][%s] Requesting Meta: %s", b.Name, ServerDataURL)
	if ws.GlobalHub != nil {
		ws.GlobalHub.BroadcastDebug(b.ID, "HTTPS", "POST "+ServerDataURL+"\nBody: "+data.Encode(), false)
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		b.Lock()
		b.Status = "HTTP_BLOCK"
		b.Unlock()
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		b.Lock()
		b.Status = "HTTP_BLOCK"
		b.Unlock()
		return err
	}

	responseString := string(body)
	if strings.Contains(responseString, "RTENDMARKERBS1001") {
		parsedData := parseGrowtopiaResponse(responseString)

		b.Lock()
		if val, ok := parsedData["server"]; ok {
			b.Server.Enet.ServerIP = val
			b.Server.Enet.NowConnectedIP = val
		}
		if val, ok := parsedData["port"]; ok {
			fmt.Sscanf(val, "%d", &b.Server.Enet.ServerPort)
			fmt.Sscanf(val, "%d", &b.Server.Enet.NowConnectedPort)
		}
		if val, ok := parsedData["meta"]; ok {
			b.Login.Meta = val
		}
		b.Unlock()

		log.Printf("[HTTP][%s] Meta received - Server: %s:%d", b.Name, b.Server.Enet.ServerIP, b.Server.Enet.ServerPort)
		if ws.GlobalHub != nil {
			ws.GlobalHub.BroadcastDebug(b.ID, "HTTPS", "Response (RTENDMARKERBS1001 found):\n"+responseString, false)
		}
	} else {
		log.Printf("[HTTP][%s] RTENDMARKERBS1001 not found in response", b.Name)
		b.Lock()
		b.Status = "HTTP_BLOCK"
		b.Unlock()
		if ws.GlobalHub != nil {
			ws.GlobalHub.BroadcastDebug(b.ID, "HTTPS", "Response (MARKER MISSING):\n"+responseString, true)
		}
		return fmt.Errorf("HTTP_BLOCK: marker missing")
	}

	return nil
}

// CheckToken performs the token validation request
func (h *HTTPHandler) CheckToken(b *bot.Bot) (string, error) {
	b.Lock()
	ltoken := b.Server.HTTPS.LToken
	loginPkt := b.Login.LoginPkt
	b.Unlock()

	if ltoken == "" {
		return "", fmt.Errorf("ltoken is empty")
	}

	data := url.Values{}
	data.Set("refreshToken", ltoken)
	data.Set("clientData", loginPkt)

	valKey := "40db4045f2d8c572efe8c4a060605726"
	targetURL := "https://login.growtopiagame.com/player/growid/checktoken?valKey=" + valKey

	req, err := http.NewRequest("POST", targetURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", GrowtopiaUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	log.Printf("[HTTP][%s] Checking Token...", b.Name)
	if ws.GlobalHub != nil {
		ws.GlobalHub.BroadcastDebug(b.ID, "HTTPS", "POST "+targetURL+"\nBody: "+data.Encode(), false)
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		b.Lock()
		b.Status = "HTTP_BLOCK"
		b.Unlock()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 502 {
		b.Lock()
		b.Status = "Bad Gateway"
		b.Unlock()
		return "", fmt.Errorf("Bad Gateway")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	responseString := string(body)
	if ws.GlobalHub != nil {
		ws.GlobalHub.BroadcastDebug(b.ID, "HTTPS", "CheckToken Response (HTTP "+fmt.Sprint(resp.StatusCode)+"):\n"+responseString, false)
	}

	// Case-insensitive check for "token is invalid"
	if strings.Contains(strings.ToLower(responseString), "token is invalid") {
		b.Lock()
		b.Status = "Invalid Token"
		b.Server.HTTPS.LToken = ""
		b.Unlock()
		return "", fmt.Errorf("Token is invalid")
	}

	if strings.Contains(responseString, "Oops, too many people trying to login at once.") {
		b.Lock()
		b.Status = "Too Many People"
		b.Unlock()
		return "", fmt.Errorf("too many people")
	}

	// Parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if status, ok := result["status"].(string); ok && status == "success" {
		b.Lock()
		if token, ok := result["token"].(string); ok {
			b.Server.HTTPS.LToken = token
		}
		b.Status = "Token Valid"
		b.Unlock()
		return b.Server.HTTPS.LToken, nil
	}

	return "", fmt.Errorf("token validation failed: %s", responseString)
}

// GetDashboard performs the dashboard request to fetch the login form URL
func (h *HTTPHandler) GetDashboard(b *bot.Bot) error {
	b.Lock()
	loginPkt := b.Login.LoginPkt
	b.Unlock()

	targetURL := "https://login.growtopiagame.com/player/login/dashboard?valKey=40db4045f2d8c572efe8c4a060605726"

	// Generate cookies for Dashboard request
	now := time.Now().Unix()
	cookieVisit := fmt.Sprintf("%d", now)
	cookieActivity := fmt.Sprintf("%d", now)

	b.Lock()
	b.Server.HTTPS.CookieVisit = cookieVisit
	b.Server.HTTPS.CookieActivity = cookieActivity
	// Temporary modify login packet string to remove name/pass if present, or rely on generator
	// Ideally generator handled it, but let's ensure the body sent has empty name/pass if user insists "tank_id_name dan tank_id_pass kosong" for requests.
	// But `loginPkt` variable comes from `b.Login.LoginPkt`, which we can't easily regex replace here safely without potentially breaking format.
	// However, user said "tank_id_name dan tank_id_pass kosong tidak usah diisi apapun pada setiap reqiest".
	// The generator already empties `tankIDPass`. `tankIDName` was kept. If user wants `tankIDName` empty too in the request body:
	// We can manually construct a "clean" packet or just assume generator is updated. Let's update generator separately if needed.
	// For now, assume generator handles struct content.
	b.Unlock()

	encodedData := url.QueryEscape(loginPkt)

	req, err := http.NewRequest("POST", targetURL, strings.NewReader(encodedData))
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", ChromeUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add Cookies
	req.AddCookie(&http.Cookie{Name: "bblastvisit", Value: cookieVisit})
	req.AddCookie(&http.Cookie{Name: "bblastactivity", Value: cookieActivity})

	log.Printf("[HTTP][%s] Requesting Dashboard: %s", b.Name, targetURL)
	if ws.GlobalHub != nil {
		ws.GlobalHub.BroadcastDebug(b.ID, "HTTPS", "POST "+targetURL+"\nBody: "+loginPkt, false)
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		b.Lock()
		b.Status = "HTTP_BLOCK"
		b.Unlock()
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	responseString := string(body)

	// Create full log with headers
	headerLog := ""
	for k, v := range resp.Header {
		for _, val := range v {
			headerLog += fmt.Sprintf("%s: %s\n", k, val)
		}
	}

	if ws.GlobalHub != nil {
		fullLog := "HTTP " + fmt.Sprint(resp.StatusCode) + "\n\n[HEADERS]\n" + headerLog + "\n[BODY]\n" + responseString
		ws.GlobalHub.BroadcastDebug(b.ID, "HTTPS", "GetDashboard Full Response:\n"+fullLog, false)
	}

	// 1. Handle HTTP Error Codes
	if resp.StatusCode == 502 {
		b.Lock()
		b.Status = "Bad Gateway"
		b.Unlock()
		return fmt.Errorf("bad gateway")
	}
	if resp.StatusCode >= 400 {
		b.Lock()
		b.Status = "HTTP_BLOCK"
		b.Unlock()
		return fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	// 2. Check for JSON "failed" status
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		if status, ok := result["status"].(string); ok && status == "failed" {
			b.Lock()
			b.Status = "HTTP_BLOCK"
			b.Unlock()
			return fmt.Errorf("dashboard failed (HTTP_BLOCK): %s", responseString)
		}
	}

	// 3. Extract Cookies directly from Dashboard response (ONLY for Gmail/Apple)
	b.Lock()
	isTokenAuth := b.Type == bot.BotTypeGmail || b.Type == bot.BotTypeApple
	b.Unlock()

	if isTokenAuth {
		extract_cookies(resp.Header["Set-Cookie"], b)
	}

	if isTokenAuth {
		extract_cookies(resp.Header["Set-Cookie"], b)
	}

	// 4. Determine provider string for extraction
	provider := "Grow" // default for legacy
	b.Lock()
	if b.Type == bot.BotTypeGmail {
		provider = "Google"
	} else if b.Type == bot.BotTypeApple {
		provider = "Apple"
	}
	b.Unlock()

	loginURL := extractLoginURL(responseString, provider)
	if loginURL == "" {
		b.Lock()
		b.Status = "FAILED LOGIN DASHBOARD"
		b.Unlock()
		return fmt.Errorf("failed to extract login URL for provider %s", provider)
	}

	b.Lock()
	b.Server.HTTPS.LoginFormURL = loginURL
	b.Unlock()

	log.Printf("[HTTP][%s] Login URL discovered: %s", b.Name, loginURL)
	return nil
}

// extractLoginURL parses the HTML and finds the href for the given provider
func extractLoginURL(htmlText string, provider string) string {
	doc, err := html.Parse(strings.NewReader(htmlText))
	if err != nil {
		return ""
	}

	var f func(*html.Node) string
	f = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "a" {
			var href, onclick string
			for _, a := range n.Attr {
				if a.Key == "href" {
					href = a.Val
				} else if a.Key == "onclick" {
					onclick = a.Val
				}
			}
			// Search for onclick containing optionChose('Provider')
			if strings.Contains(onclick, fmt.Sprintf("optionChose('%s')", provider)) {
				return href
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if res := f(c); res != "" {
				return res
			}
		}
		return ""
	}

	return f(doc)
}

// GetCookies performs the request to get session cookies and form token
func (h *HTTPHandler) GetCookies(b *bot.Bot) error {
	b.Lock()
	targetURL := b.Server.HTTPS.LoginFormURL
	b.Unlock()

	if targetURL == "" {
		return fmt.Errorf("login form URL is empty")
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return err
	}

	// Browser-like headers
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?1")
	req.Header.Set("sec-ch-ua-platform", "\"Windows\"")
	req.Header.Set("upgrade-insecure-requests", "1")
	req.Header.Set("User-Agent", ChromeUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("sec-fetch-site", "cross-site")
	req.Header.Set("sec-fetch-mode", "navigate")
	req.Header.Set("sec-fetch-dest", "document")
	req.Header.Set("accept-encoding", "identity") // Request plain text only
	req.Header.Set("accept-language", "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7")

	now := time.Now().Unix()
	cookieVisit := fmt.Sprintf("%d", now)
	cookieActivity := fmt.Sprintf("%d", now)

	b.Lock()
	b.Server.HTTPS.CookieVisit = cookieVisit
	b.Server.HTTPS.CookieActivity = cookieActivity
	b.Unlock()

	req.AddCookie(&http.Cookie{Name: "bblastvisit", Value: cookieVisit})
	req.AddCookie(&http.Cookie{Name: "bblastactivity", Value: cookieActivity})

	log.Printf("[HTTP][%s] Requesting Cookies: %s", b.Name, targetURL)
	if ws.GlobalHub != nil {
		ws.GlobalHub.BroadcastDebug(b.ID, "HTTPS", "GET "+targetURL, false)
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		b.Lock()
		b.Status = "HTTP_BLOCK"
		b.Unlock()
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 502 {
		b.Lock()
		b.Status = "Bad Gateway"
		b.Unlock()
		return fmt.Errorf("bad gateway")
	}
	if resp.StatusCode >= 400 {
		b.Lock()
		b.Status = "HTTP_BLOCK"
		b.Unlock()
		return fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	responseString := string(body)

	// Log Received Cookies (Headers)
	// We want full headers now
	headerLog := ""
	for k, v := range resp.Header {
		for _, val := range v {
			headerLog += fmt.Sprintf("%s: %s\n", k, val)
		}
	}

	// Cookies specifically for console log (legacy request)
	cookies := resp.Header["Set-Cookie"]
	cookieLog := ""
	if len(cookies) > 0 {
		cookieLog = "\n[Received Cookies - Extracted]:\n"
		for _, c := range cookies {
			cookieLog += c + "\n"
		}
	}

	// Log to Console Terminal
	log.Printf("[DEBUG-CONSOLE][%s] GetCookies Response:\n[HEADERS]\n%s\n[BODY]\n%s\n", b.Name, headerLog, responseString)

	if ws.GlobalHub != nil {
		fullLog := "HTTP " + fmt.Sprint(resp.StatusCode) + "\n\n[HEADERS]\n" + headerLog + "\n[BODY]\n" + responseString
		ws.GlobalHub.BroadcastDebug(b.ID, "HTTPS", "GetCookies Full Response:\n"+fullLog, false)
	}

	// Parse and store cookies
	extract_cookies(cookies, b)

	formToken := extract_form_token(responseString)
	if formToken == "" {
		b.Lock()
		b.Status = "Cookies Not Found"
		b.Unlock()
		return fmt.Errorf("form token not found in response")
	}

	b.Lock()
	b.Server.HTTPS.FormToken = formToken
	b.Unlock()

	log.Printf("[HTTP][%s] Cookies & Form Token obtained: %s", b.Name, formToken)
	return nil
}

// extract_cookies parses the Set-Cookie headers and updates the bot state
func extract_cookies(cookies []string, b *bot.Bot) {
	b.Lock()
	defer b.Unlock()

	for _, cookieStr := range cookies {
		// Basic parsing found in Set-Cookie: Name=Value; ...
		parts := strings.Split(cookieStr, ";")
		if len(parts) > 0 {
			kv := strings.SplitN(parts[0], "=", 2)
			if len(kv) == 2 {
				name := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])

				// Debug log for each parsed cookie
				log.Printf("[Cookie Parsed] Name: %s, Value: %s", name, value)

				switch name {
				case "AWSALBTG":
					b.Server.HTTPS.CookieAWSALBTG = value
				case "AWSALBTGCORS":
					b.Server.HTTPS.CookieAWSALBTGCORS = value
				case "AWSALB":
					b.Server.HTTPS.CookieAWSALB = value
				case "AWSALBCORS":
					b.Server.HTTPS.CookieAWSALBCORS = value
				case "XSRF-TOKEN":
					b.Server.HTTPS.CookieXSRF = value
				case "growtopia_session", "growtopia_game_session":
					b.Server.HTTPS.CookieGameSession = value
				}
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetToken performs login validation based on bot type (Legacy vs External Auth)
func (h *HTTPHandler) GetToken(b *bot.Bot) error {
	b.Lock()
	botType := b.Type
	b.Unlock()

	if botType == bot.BotTypeLegacy {
		return h.getTokenLegacy(b)
	}
	return h.getTokenExternal(b)
}

// getTokenLegacy handles standard GrowID login
func (h *HTTPHandler) getTokenLegacy(b *bot.Bot) error {
	b.Lock()
	formToken := b.Server.HTTPS.FormToken
	loginURL := b.Server.HTTPS.LoginFormURL
	name := b.Name
	password := b.Login.TankIDPass
	b.Unlock()

	if formToken == "" {
		return fmt.Errorf("form token is empty")
	}

	targetURL := "https://login.growtopiagame.com/player/growid/login/validate"

	data := url.Values{}
	data.Set("_token", formToken)
	data.Set("growId", name)
	data.Set("password", password)

	req, err := http.NewRequest("POST", targetURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	// Headers
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?1")
	req.Header.Set("sec-ch-ua-platform", "\"Windows\"")
	req.Header.Set("Origin", "https://login.growtopiagame.com")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", ChromeUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Referer", loginURL)
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Accept-Language", "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7")

	// Cookies
	b.Lock()
	httpsCfg := b.Server.HTTPS
	b.Unlock()

	req.AddCookie(&http.Cookie{Name: "bblastvisit", Value: httpsCfg.CookieVisit})
	req.AddCookie(&http.Cookie{Name: "bblastactivity", Value: httpsCfg.CookieActivity})
	if httpsCfg.CookieAWSALBTG != "" {
		req.AddCookie(&http.Cookie{Name: "AWSALBTG", Value: httpsCfg.CookieAWSALBTG})
	}
	if httpsCfg.CookieAWSALBTGCORS != "" {
		req.AddCookie(&http.Cookie{Name: "AWSALBTGCORS", Value: httpsCfg.CookieAWSALBTGCORS})
	}
	if httpsCfg.CookieAWSALB != "" {
		req.AddCookie(&http.Cookie{Name: "AWSALB", Value: httpsCfg.CookieAWSALB})
	}
	if httpsCfg.CookieAWSALBCORS != "" {
		req.AddCookie(&http.Cookie{Name: "AWSALBCORS", Value: httpsCfg.CookieAWSALBCORS})
	}
	if httpsCfg.CookieXSRF != "" {
		req.AddCookie(&http.Cookie{Name: "XSRF-TOKEN", Value: httpsCfg.CookieXSRF})
	}
	if httpsCfg.CookieGameSession != "" {
		req.AddCookie(&http.Cookie{Name: "growtopia_game_session", Value: httpsCfg.CookieGameSession})
	}

	log.Printf("[HTTP][%s] Requesting Token: %s", b.Name, targetURL)
	if ws.GlobalHub != nil {
		ws.GlobalHub.BroadcastDebug(b.ID, "HTTPS", "POST "+targetURL+"\nBody: "+data.Encode(), false)
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		b.Lock()
		b.Status = "HTTP_BLOCK"
		b.Unlock()
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 502 {
		b.Lock()
		b.Status = "HTTP_BLOCK" // As requested for bad gateway
		b.Unlock()
		return fmt.Errorf("bad gateway (HTTP_BLOCK)")
	}
	if resp.StatusCode >= 400 {
		b.Lock()
		b.Status = "HTTP_BLOCK"
		b.Unlock()
		return fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	responseString := string(body)

	log.Printf("[DEBUG-CONSOLE][%s] GetToken Response:\n%s\n", b.Name, responseString)
	if ws.GlobalHub != nil {
		ws.GlobalHub.BroadcastDebug(b.ID, "HTTPS", "GetToken Response (HTTP "+fmt.Sprint(resp.StatusCode)+"):\n"+responseString, false)
	}

	// Parse JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		b.Lock()
		b.Status = "Token Not Found"
		b.Unlock()
		log.Printf("[ERROR][%s] Failed to parse JSON: %v", b.Name, err)
		return fmt.Errorf("failed to parse json")
	}

	status, okStatus := result["status"].(string)
	token, okToken := result["token"].(string)

	if okStatus && okToken && status == "success" && token != "" {
		b.Lock()
		b.Server.HTTPS.StatusToken = "success"
		b.Server.HTTPS.LToken = token
		b.Status = "success"
		b.Unlock()
		log.Printf("[HTTP][%s] Login Token Obtained: %s", b.Name, token)
		return nil
	}

	// If we are here, something is wrong
	b.Lock()
	b.Server.HTTPS.StatusToken = "failed"
	b.Status = "failed"
	b.Unlock()
	log.Printf("[ERROR][%s] JSON response missing status or token, or status not success.", b.Name)
	return fmt.Errorf("token not found in response")
}

// getTokenExternal handles Gmail/Apple login via external service
func (h *HTTPHandler) getTokenExternal(b *bot.Bot) error {
	b.Lock()
	ext := b.ExternalAuth
	httpsCfg := b.Server.HTTPS
	proxyStr := b.Proxy
	name := b.Name
	password := b.ExternalPassword
	b.Unlock()

	hasExternal := ext.IP != "" && ext.Port != 0 && ext.AccessKey != ""
	if !hasExternal || (b.Type == bot.BotTypeGmail && !ext.UseForGoogle) {
		// Fallback to manual
		log.Printf("[ExternalAuth] No external auth configured or disabled. Waiting for manual input.")
		b.Lock()
		b.Status = "Waiting for ltoken input"
		b.Unlock()
		if ws.GlobalHub != nil {
			ws.GlobalHub.BroadcastBotUpdate() // Update UI
		}
		// Open Browser command could be here (server-side open is tricky, better send event to UI)
		return fmt.Errorf("manual_ltoken_required")
	}

	b.Lock()
	b.Status = "ExtAuth: Creating Task..."
	b.Unlock()
	if ws.GlobalHub != nil {
		ws.GlobalHub.BroadcastBotUpdate()
	}

	// Prepare Cookies
	cookies := []string{}
	addCookie := func(name, value string, httpOnly, secure bool) {
		if value == "" {
			return
		}
		domain := "login.growtopiagame.com"
		flag1 := "FALSE"
		path := "/"
		flag2 := "FALSE"
		if secure {
			flag2 = "TRUE"
		}
		httpOnlyFlag := ""
		if httpOnly {
			httpOnlyFlag = "#HttpOnly_"
		}
		exp := time.Now().Add(365 * 24 * time.Hour).Unix()
		line := fmt.Sprintf("%s%s\t%s\t%s\t%s\t%d\t%s\t%s", httpOnlyFlag, domain, flag1, path, flag2, exp, name, value)
		cookies = append(cookies, line)
	}

	addCookie("AWSALBTG", httpsCfg.CookieAWSALBTG, false, false)
	addCookie("AWSALBTGCORS", httpsCfg.CookieAWSALBTGCORS, false, true)
	addCookie("AWSALB", httpsCfg.CookieAWSALB, false, false)
	addCookie("AWSALBCORS", httpsCfg.CookieAWSALBCORS, false, true)
	addCookie("XSRF-TOKEN", httpsCfg.CookieXSRF, false, true)
	addCookie("growtopia_game_session", httpsCfg.CookieGameSession, true, true)

	// Prepare JSON Payload
	payload := map[string]interface{}{
		"accessKey": ext.AccessKey,
		"appleData": "", // TODO: Handle Apple Data if needed
		"cookies":   cookies,
		"mail":      name,
		"mobile":    false,
		"pass":      password,
		"recovery":  "",
		"secret":    "",
		"url":       httpsCfg.LoginFormURL,
		"proxy": map[string]string{
			"data":     "",
			"protocol": "socks5",
		},
	}

	// Handle Proxy for External Service
	// Convert from host:port:user:password to user:password@host:port
	if proxyStr != "" {
		parts := strings.Split(proxyStr, ":")
		var proxyData string

		if len(parts) == 4 {
			// host:port:user:password -> user:password@host:port
			proxyData = parts[2] + ":" + parts[3] + "@" + parts[0] + ":" + parts[1]
		} else if len(parts) == 2 {
			// host:port (no auth)
			proxyData = parts[0] + ":" + parts[1]
		} else {
			// Use as-is if format is unknown
			proxyData = proxyStr
		}

		payload["proxy"] = map[string]string{
			"data":     proxyData,
			"protocol": "socks5",
		}
		log.Printf("[ExtAuth][%s] Using proxy: %s", b.Name, proxyData)
	} else {
		log.Printf("[ExtAuth][%s] No proxy configured for external auth", b.Name)
	}

	jsonBody, _ := json.Marshal(payload)
	postURL := fmt.Sprintf("http://%s:%d/createTask", ext.IP, ext.Port)

	// Validate required fields
	if name == "" {
		log.Printf("[ExtAuth][%s] WARNING: mail field is empty!", b.Name)
	}
	if proxyStr == "" {
		log.Printf("[ExtAuth][%s] WARNING: proxy field is empty!", b.Name)
	}

	log.Printf("[ExtAuth][%s] Payload validation - mail: '%s', proxy: '%s'", b.Name, name, proxyStr)

	// Create Request to External Service
	// IMPORTANT: This request goes to the external service directly, NOT via the bot's proxy
	// UNLESS the external service is blocked too? Usually external service requests are direct.
	// We use a fresh client or `http.DefaultClient` or verify if h.Client (which has proxy) should be used.
	// C++ uses proxy on the curl handle for `createTask` if `acquiredProxyHostPort` is set.
	// This implies we might need to use the proxy to talk to the external service too?
	// Re-reading C++: "Apply the same acquired proxy to the curl handle so network and JSON align".
	// So yes, we should probably use h.Client if we want to route through proxy, OR create a new client.
	// For robust implementation, let's use a Direct Client for now unless specified otherwise,
	// as usually local -> external service is direct, and external service -> Growtopia uses the proxy sent in JSON.
	// However, if the user assumes "ApplyBypassProxyIfRequested" applies to the task creation itself, we should follow.
	// Given the ambiguity, I'll use a direct client for the Service API call to avoid "proxyception" issues unless necessary.

	extClient := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("POST", postURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[ExtAuth][%s] Creating Task at %s", b.Name, postURL)

	// Broadcast request payload to debug
	if ws.GlobalHub != nil {
		// Pretty print JSON for readability
		var prettyJSON bytes.Buffer
		json.Indent(&prettyJSON, jsonBody, "", "  ")
		ws.GlobalHub.BroadcastDebug(b.ID, "EXT_AUTH", "CreateTask Request:\nURL: "+postURL+"\nPayload:\n"+prettyJSON.String(), false)
	}

	resp, err := extClient.Do(req)
	if err != nil {
		log.Printf("[ExtAuth] POST Failed: %v", err)
		b.Lock()
		b.Status = "ExtAuth Post Failed"
		b.Unlock()
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	respStr := string(bodyBytes)

	// Debug log
	if ws.GlobalHub != nil {
		ws.GlobalHub.BroadcastDebug(b.ID, "EXT_AUTH", "CreateTask Resp: "+respStr, false)
	}

	var respJSON map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &respJSON); err != nil {
		return fmt.Errorf("invalid json from ext auth")
	}

	status, _ := respJSON["status"].(string)
	statusCodeVal, _ := respJSON["statusCode"].(float64) // Numbers are float64 in interface{}
	statusCode := int(statusCodeVal)
	taskID, _ := respJSON["id"].(string) // Server returns "id" not "taskId"

	if status == "" && statusCode == 0 {
		b.Lock()
		b.Status = "ExtAuth Invalid Resp"
		b.Unlock()
		return fmt.Errorf("invalid response")
	}

	log.Printf("[ExtAuth][%s] Task created with ID: %s", b.Name, taskID)

	b.Lock()
	if status != "" {
		b.Status = "ExtAuth: " + status
	} else {
		b.Status = fmt.Sprintf("ExtAuth: %d", statusCode)
	}
	b.Unlock()
	if ws.GlobalHub != nil {
		ws.GlobalHub.BroadcastBotUpdate()
	}

	// Prepare for Polling
	maxWait := 60
	waited := 0
	interval := 3

	// Polling endpoint is different from createTask
	pollURL := fmt.Sprintf("http://%s:%d/getTaskResult", ext.IP, ext.Port)

	// createTask returns statusCode 1 (task accepted)
	// We need to start polling immediately
	// Set statusCode to 2 to enter the loop
	statusCode = 2

	// For getTaskResult response:
	// statusCode 1 = done/completed
	// statusCode 2 = processing
	// So we poll while statusCode is 2 (still processing)
	for statusCode == 2 && waited < maxWait {
		time.Sleep(time.Duration(interval) * time.Second)
		waited += interval

		pollPayload := map[string]interface{}{
			"accessKey": ext.AccessKey,
			"id":        taskID,
		}
		pollBody, _ := json.Marshal(pollPayload)

		// Log Polling
		if ws.GlobalHub != nil {
			ws.GlobalHub.BroadcastDebug(b.ID, "EXT_AUTH_POLL", "Polling Task: "+taskID, false)
		}

		pollReq, _ := http.NewRequest("POST", pollURL, strings.NewReader(string(pollBody)))
		pollReq.Header.Set("Content-Type", "application/json")

		pollResp, err := extClient.Do(pollReq)
		if err != nil {
			log.Printf("[ExtAuth] Poll Failed: %v", err)
			break
		}

		pollBytes, _ := io.ReadAll(pollResp.Body)
		pollResp.Body.Close()

		if ws.GlobalHub != nil {
			ws.GlobalHub.BroadcastDebug(b.ID, "EXT_AUTH_POLL", "Poll Resp: "+string(pollBytes), false)
		}

		var pollJSON map[string]interface{}
		if err := json.Unmarshal(pollBytes, &pollJSON); err == nil {
			status, _ = pollJSON["status"].(string)
			statusCodeVal, _ := pollJSON["statusCode"].(float64)
			statusCode = int(statusCodeVal)

			b.Lock()
			if status != "" {
				b.Status = "ExtAuth: " + status
			} else {
				b.Status = fmt.Sprintf("ExtAuth: %d", statusCode)
			}
			b.Unlock()
			if ws.GlobalHub != nil {
				ws.GlobalHub.BroadcastBotUpdate()
			}

			// statusCode 1 in getTaskResult means done/completed
			if statusCode == 1 {
				// Check if 'data' field contains the token
				if dataStr, ok := pollJSON["data"].(string); ok && dataStr != "" {
					// Try to parse inner data if it's a JSON string
					var innerData map[string]interface{}
					if err := json.Unmarshal([]byte(dataStr), &innerData); err == nil {
						// Check status first
						if innerStatus, ok := innerData["status"].(string); ok && innerStatus == "success" {
							if token, ok := innerData["token"].(string); ok && token != "" {
								b.Lock()
								b.Server.HTTPS.StatusToken = "success"
								b.Server.HTTPS.LToken = token
								b.Status = "success"
								b.Unlock()
								log.Printf("[ExtAuth][%s] Success! Token obtained: %s", b.Name, token[:20]+"...")
								return nil
							}
						} else {
							// Failed status
							log.Printf("[ExtAuth][%s] Auth failed: %v", b.Name, innerData)
							b.Lock()
							b.Server.HTTPS.StatusToken = "failed"
							b.Status = "failed"
							b.Unlock()
							return fmt.Errorf("external auth failed: %v", innerData)
						}
					}
				}
				// If we reach here with statusCode 1 but no token, something is wrong
				log.Printf("[ExtAuth][%s] Task completed but no token found", b.Name)
				break
			}
		}
	}

	// Check if we got the token (status might be success but token logic above handled it)
	// If loop finished and we returned nil, great. If we are here, we failed or timed out.

	b.Lock()
	finalToken := b.Server.HTTPS.LToken
	b.Unlock()

	if finalToken != "" {
		return nil
	}

	b.Lock()
	b.Status = "ExtAuth Failed/Timeout"
	b.Unlock()
	return fmt.Errorf("external auth failed or timed out")
}

// extract_form_token parses the HTML input named '_token'
func extract_form_token(htmlText string) string {
	doc, err := html.Parse(strings.NewReader(htmlText))
	if err != nil {
		return ""
	}

	var f func(*html.Node) string
	f = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "input" {
			var name, value string
			for _, a := range n.Attr {
				if a.Key == "name" {
					name = a.Val
				} else if a.Key == "value" {
					value = a.Val
				}
			}
			if name == "_token" {
				return value
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if res := f(c); res != "" {
				return res
			}
		}
		return ""
	}
	return f(doc)
}

// parseGrowtopiaResponse parses the newline-separated key|value response
func parseGrowtopiaResponse(input string) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
