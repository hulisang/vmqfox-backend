// Package php 提供 PHP/Android 协议兼容所需的无框架辅助类型与函数。
package php

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	ContentTypeJSON = "application/json; charset=utf-8"
	ContentTypeHTML = "text/html; charset=utf-8"
	ContentTypePNG  = "image/png"
)

// Envelope 是新版接口通用响应；Code 可由调用方指定，包括 -1 或 1 等错误码。
type Envelope struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func NewEnvelope(code int, msg string, data any) Envelope {
	return Envelope{Code: code, Msg: msg, Data: data}
}
func Success(data any) Envelope                     { return NewEnvelope(200, "", data) }
func Error(code int, msg string, data any) Envelope { return NewEnvelope(code, msg, data) }

// LayuiEnvelope 独立表示 Layui 约定的 code=0 成功响应，避免与新版 envelope 混用。
type LayuiEnvelope struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Count int64  `json:"count,omitempty"`
	Data  any    `json:"data"`
}

func LayuiSuccess(data any) LayuiEnvelope {
	return LayuiEnvelope{Code: 0, Msg: "", Data: data}
}

func LayuiPage(total int64, data any) LayuiEnvelope {
	return LayuiEnvelope{Code: 0, Msg: "", Count: total, Data: data}
}

// Param 从参数映射读取值并转为字符串；不存在或 nil 时返回 fallback。
func Param(params map[string]any, name, fallback string) string {
	v, ok := params[name]
	if !ok || v == nil {
		return fallback
	}
	return StringValue(v)
}

// StringValue 将常见 PHP/JSON 数字值规范化为协议字符串，避免 float 的指数表示。
func StringValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case int:
		return strconv.Itoa(x)
	case int8:
		return strconv.FormatInt(int64(x), 10)
	case int16:
		return strconv.FormatInt(int64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint:
		return strconv.FormatUint(uint64(x), 10)
	case uint8:
		return strconv.FormatUint(uint64(x), 10)
	case uint16:
		return strconv.FormatUint(uint64(x), 10)
	case uint32:
		return strconv.FormatUint(uint64(x), 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case float32:
		return NormalizeNumber(strconv.FormatFloat(float64(x), 'f', -1, 32))
	case float64:
		return NormalizeNumber(strconv.FormatFloat(x, 'f', -1, 64))
	default:
		return fmt.Sprint(v)
	}
}

func NormalizeNumber(s string) string {
	s = strings.TrimSpace(s)
	if strings.ContainsAny(s, ".eE") {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return strconv.FormatFloat(f, 'f', -1, 64)
		}
	}
	return s
}

func md5Hex(s string) string { sum := md5.Sum([]byte(s)); return hex.EncodeToString(sum[:]) }

// CreateSignNew 保留新版创建订单的五段格式；创建请求不包含 reallyPrice。
func CreateSignNew(payID, param, typ, price, key string) string {
	return md5Hex("payId=" + payID + "&param=" + param + "&type=" + typ + "&price=" + price + "&key=" + key)
}

func CallbackSignNew(payID, param, typ, price, reallyPrice, key string) string {
	return md5Hex("payId=" + payID + "&param=" + param + "&type=" + typ + "&price=" + price + "&reallyPrice=" + reallyPrice + "&key=" + key)
}

func HeartbeatSign(timestamp, key string) string { return md5Hex(timestamp + key) }

// PushSign 严格按 type+price+t+key 拼接后计算 MD5。
func PushSign(typ, price, timestamp, key string) string { return md5Hex(typ + price + timestamp + key) }

// FormatAmountCents 将分转换为两位小数金额字符串。
func FormatAmountCents(cents int64) string { return fmt.Sprintf("%d.%02d", cents/100, abs(cents%100)) }

// FormatAmount 将金额规范化为固定两位小数，采用四舍五入。
func FormatAmount(amount float64) string { return fmt.Sprintf("%.2f", math.Round(amount*100)/100) }

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
