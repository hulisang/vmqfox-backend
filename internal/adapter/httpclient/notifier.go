package httpclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/port"
)

const maxNotificationResponseBytes = 1024 * 1024

type Notifier struct {
	client *http.Client
}

func NewNotifier(client *http.Client) *Notifier {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	} else {
		copy := *client
		client = &copy
	}
	// 重定向可能改变方法或重复通知，所有客户端都必须保留原始 3xx 响应。
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Notifier{client: client}
}

func (n *Notifier) Send(ctx context.Context, notification port.Notification) (port.NotificationResult, error) {
	if notification.URL == "" || notification.Method == "" {
		return port.NotificationResult{}, errors.New("通知 URL 或方法为空")
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
