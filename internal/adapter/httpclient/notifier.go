package httpclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/config"
	"github.com/hulisang/vmqfox-backend/internal/port"
)

const maxNotificationResponseBytes = 1024 * 1024

type Notifier struct {
	client *http.Client
}

// NewNotifier 创建具备 SSRF 防御能力的通知出站 HTTP 客户端
func NewNotifier(client *http.Client, outboundCfg ...config.OutboundConfig) *Notifier {
	var cfg config.OutboundConfig
	if len(outboundCfg) > 0 {
		cfg = outboundCfg[0]
	}

	if client == nil {
		dialer := &net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
			Control: func(network, address string, c syscall.RawConn) error {
				return nil
			},
		}

		transport := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}

				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				if len(ips) == 0 {
					return nil, errors.New("无法解析目标主机 IP")
				}

				// 检查解析出的每个 IP 是否安全（防止 DNS Rebinding 逃逸）
				for _, ipAddr := range ips {
					if err := checkIPSecurity(ipAddr.IP, cfg); err != nil {
						return nil, fmt.Errorf("SSRF 拦截: %w", err)
					}
				}

				// 使用解析出的首个已校验 IP 建立连接
				targetAddr := net.JoinHostPort(ips[0].IP.String(), port)
				return dialer.DialContext(ctx, network, targetAddr)
			},
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}

		client = &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		}
	} else {
		copy := *client
		client = &copy
	}

	// 重定向可能改变方法或导致未授权跳转，始终保留原始 3xx 响应
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Notifier{client: client}
}

func checkIPSecurity(ip net.IP, cfg config.OutboundConfig) error {
	if ip == nil {
		return errors.New("无效的目标 IP")
	}

	// 1. 优先判断是否显式命中了白名单（如果配置了 127.0.0.1 或指定私网网段则直接放行）
	for _, allowedIP := range cfg.AllowedIPs {
		if allowedIP.Equal(ip) {
			return nil
		}
	}
	for _, cidr := range cfg.AllowedCIDRs {
		if cidr.Contains(ip) {
			return nil
		}
	}

	// 2. 高危元数据地址与未指定地址（未显式白名单时绝对拒绝）
	if ip.IsUnspecified() || ip.String() == "169.254.169.254" {
		return fmt.Errorf("目标地址属于敏感或受限地址: %s", ip.String())
	}

	// 3. 默认阻止私网、回环、链路本地等内网地址
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return fmt.Errorf("目标地址属于未白名单放行的私网或回环地址: %s", ip.String())
	}

	return nil
}

func (n *Notifier) Send(ctx context.Context, notification port.Notification) (port.NotificationResult, error) {
	if notification.URL == "" || notification.Method == "" {
		return port.NotificationResult{}, errors.New("通知 URL 或方法为空")
	}

	parsedURL, err := url.Parse(notification.URL)
	if err != nil {
		return port.NotificationResult{}, fmt.Errorf("通知 URL 无效: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return port.NotificationResult{}, fmt.Errorf("不支持的通知协议 Scheme: %s", parsedURL.Scheme)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		notification.Method,
		notification.URL,
		bytes.NewReader(notification.Body),
	)
	if err != nil {
		return port.NotificationResult{}, err
	}
	for key, value := range notification.Headers {
		request.Header.Set(key, value)
	}

	response, err := n.client.Do(request)
	if err != nil {
		return port.NotificationResult{}, err
	}
	defer response.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxNotificationResponseBytes+1))
	if len(body) > maxNotificationResponseBytes {
		body = body[:maxNotificationResponseBytes]
		if readErr == nil {
			readErr = errors.New("通知响应正文超过限制")
		}
	}
	return port.NotificationResult{StatusCode: response.StatusCode, Body: body}, readErr
}

var _ port.Notifier = (*Notifier)(nil)

