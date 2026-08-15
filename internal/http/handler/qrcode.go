package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
	"github.com/hulisang/vmqfox-backend/internal/domain/qrcode"
	"github.com/hulisang/vmqfox-backend/internal/port"
	"github.com/hulisang/vmqfox-backend/internal/usecase"
)

const (
	maxQRCodeUploadBytes  = 8 << 20
	maxQRCodeRequestBytes = maxQRCodeUploadBytes + (1 << 20)
)

type QRCodeImageManager interface {
	GeneratePNG(content string) (usecase.QRCodePNG, error)
	Decode(imageData []byte) (string, error)
}

// QRCodeManager 只暴露管理端所需用例，不让数据库对象进入 transport。
type QRCodeManager interface {
	List(context.Context, usecase.ListQRCodesInput) (port.QRCodePage, error)
	Create(context.Context, usecase.CreateQRCodeInput) (qrcode.QRCode, error)
	SetState(context.Context, int64, qrcode.State) error
	Delete(context.Context, int64, *payment.Type) error
}

type QRCodeHandlers struct {
	GenerateAPI     gin.HandlerFunc
	ParseAPI        gin.HandlerFunc
	ListAPI         gin.HandlerFunc
	ListWechatAPI   gin.HandlerFunc
	ListAlipayAPI   gin.HandlerFunc
	CreateAPI       gin.HandlerFunc
	CreateWechatAPI gin.HandlerFunc
	CreateAlipayAPI gin.HandlerFunc
	SetStateAPI     gin.HandlerFunc
	DeleteAPI       gin.HandlerFunc
	DeleteWechatAPI gin.HandlerFunc
	DeleteAlipayAPI gin.HandlerFunc
}

func newQRCodeHandlers(images QRCodeImageManager, manager QRCodeManager) QRCodeHandlers {
	return QRCodeHandlers{
		GenerateAPI:     generateQRCodeHandler(images),
		ParseAPI:        parseQRCodeHandler(images),
		ListAPI:         listQRCodesHandler(manager, nil),
		ListWechatAPI:   listQRCodesHandler(manager, paymentTypePointer(payment.Wechat)),
		ListAlipayAPI:   listQRCodesHandler(manager, paymentTypePointer(payment.Alipay)),
		CreateAPI:       createQRCodeHandler(manager, nil),
		CreateWechatAPI: createQRCodeHandler(manager, paymentTypePointer(payment.Wechat)),
		CreateAlipayAPI: createQRCodeHandler(manager, paymentTypePointer(payment.Alipay)),
		SetStateAPI:     setQRCodeStateHandler(manager),
		DeleteAPI:       deleteQRCodeHandler(manager, nil),
		DeleteWechatAPI: deleteQRCodeHandler(manager, paymentTypePointer(payment.Wechat)),
		DeleteAlipayAPI: deleteQRCodeHandler(manager, paymentTypePointer(payment.Alipay)),
	}
}

func paymentTypePointer(value payment.Type) *payment.Type {
	return &value
}

func listQRCodesHandler(service QRCodeManager, fixedType *payment.Type) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeQRCodeUnavailable(c)
			return
		}
		page, err := positiveQueryInt(c.Query("page"), 1)
		if err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "分页参数无效", nil))
			return
		}
		limit, err := positiveQueryInt(c.Query("limit"), 10)
		if err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "分页参数无效", nil))
			return
		}

		filterType := fixedType
		if filterType == nil && strings.TrimSpace(c.Query("type")) != "" {
			value, parseErr := strconv.Atoi(c.Query("type"))
			parsed := payment.Type(value)
			if parseErr != nil || !parsed.Valid() {
				c.JSON(http.StatusOK, php.NewEnvelope(400, "支付类型错误", nil))
				return
			}
			filterType = &parsed
		}

		result, err := service.List(c.Request.Context(), usecase.ListQRCodesInput{
			Type:  filterType,
			Page:  page,
			Limit: limit,
		})
		if err != nil {
			writeQRCodeError(c, err)
			return
		}
		items := make([]gin.H, 0, len(result.Items))
		for _, value := range result.Items {
			items = append(items, apiQRCodeData(value))
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{
			"total": result.Total,
			"items": items,
		}))
	}
}

func createQRCodeHandler(service QRCodeManager, fixedType *payment.Type) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeQRCodeUnavailable(c)
			return
		}
		params, err := scalarRequestParams(c)
		if err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "二维码参数无效", nil))
			return
		}

		paymentType := payment.Type(0)
		if fixedType != nil {
			paymentType = *fixedType
		} else {
			value, parseErr := strconv.Atoi(params["type"])
			paymentType = payment.Type(value)
			if parseErr != nil || !paymentType.Valid() {
				c.JSON(http.StatusOK, php.NewEnvelope(400, "支付类型错误", nil))
				return
			}
		}

		_, err = service.Create(c.Request.Context(), usecase.CreateQRCodeInput{
			Type:   paymentType,
			PayURL: params["pay_url"],
			Price:  params["price"],
		})
		if err != nil {
			writeQRCodeError(c, err)
			return
		}
		message := "添加二维码成功"
		if fixedType != nil {
			message = "添加" + qrcodePaymentTypeText(*fixedType) + "二维码成功"
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, message, nil))
	}
}

func setQRCodeStateHandler(service QRCodeManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeQRCodeUnavailable(c)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "二维码ID无效", nil))
			return
		}
		params, err := scalarRequestParams(c)
		if err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "二维码参数无效", nil))
			return
		}
		if params["state"] == "" {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "状态参数不能为空", nil))
			return
		}
		stateNumber, err := strconv.Atoi(params["state"])
		state := qrcode.State(stateNumber)
		if err != nil || !state.Valid() {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "二维码状态无效", nil))
			return
		}
		if err := service.SetState(c.Request.Context(), id, state); err != nil {
			writeQRCodeError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "设置二维码状态成功", nil))
	}
}

func deleteQRCodeHandler(service QRCodeManager, expectedType *payment.Type) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeQRCodeUnavailable(c)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "二维码ID无效", nil))
			return
		}
		if err := service.Delete(c.Request.Context(), id, expectedType); err != nil {
			writeQRCodeError(c, err)
			return
		}
		message := "删除二维码成功"
		if expectedType != nil {
			message = "删除" + qrcodePaymentTypeText(*expectedType) + "二维码成功"
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, message, nil))
	}
}

func apiQRCodeData(value qrcode.QRCode) gin.H {
	return gin.H{
		"id":         value.ID,
		"type":       int(value.Type),
		"type_text":  qrcodePaymentTypeText(value.Type),
		"pay_url":    value.PayURL,
		"price":      json.Number(order.FormatCents(value.PriceCents)),
		"state":      int(value.State),
		"state_text": qrcodeStateText(value.State),
	}
}

func qrcodePaymentTypeText(value payment.Type) string {
	if value == payment.Wechat {
		return "微信"
	}
	return "支付宝"
}

func qrcodeStateText(value qrcode.State) string {
	if value == qrcode.StateEnabled {
		return "正常"
	}
	return "禁用"
}

func generateQRCodeHandler(service QRCodeImageManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeQRCodeImageUnavailable(c)
			return
		}
		content := c.Query("url")
		if content == "" {
			content = c.Param("url")
		}
		if content == "" {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "URL参数不能为空", nil))
			return
		}
		result, err := service.GeneratePNG(content)
		if err != nil {
			writeQRCodeImageError(c, err)
			return
		}

		etag := `"` + result.CacheKey + `"`
		c.Header("ETag", etag)
		c.Header("Cache-Control", "private, max-age=31536000, immutable")
		c.Header("X-Content-Type-Options", "nosniff")
		if c.GetHeader("If-None-Match") == etag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Data(http.StatusOK, "image/png", result.Data)
	}
}

func parseQRCodeHandler(service QRCodeImageManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeQRCodeImageUnavailable(c)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxQRCodeRequestBytes)
		fileHeader, err := c.FormFile("file")
		if err != nil {
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				c.JSON(http.StatusOK, php.NewEnvelope(400, "二维码图片不能超过8MB", nil))
				return
			}
			c.JSON(http.StatusOK, php.NewEnvelope(400, "二维码数据不能为空，请选择文件上传", nil))
			return
		}
		if fileHeader.Size <= 0 {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "二维码数据不能为空，请选择文件上传", nil))
			return
		}
		if fileHeader.Size > maxQRCodeUploadBytes {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "二维码图片不能超过8MB", nil))
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "读取二维码图片失败", nil))
			return
		}
		defer file.Close()

		imageData, err := io.ReadAll(io.LimitReader(file, maxQRCodeUploadBytes+1))
		if err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "读取二维码图片失败", nil))
			return
		}
		if len(imageData) == 0 {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "二维码数据不能为空，请选择文件上传", nil))
			return
		}
		if len(imageData) > maxQRCodeUploadBytes {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "二维码图片不能超过8MB", nil))
			return
		}

		text, err := service.Decode(imageData)
		if err != nil {
			writeQRCodeImageError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{"url": text}))
	}
}

func writeQRCodeUnavailable(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "二维码服务不可用", nil))
}

func writeQRCodeImageUnavailable(c *gin.Context) {
	writeQRCodeUnavailable(c)
}

func writeQRCodeImageError(c *gin.Context, err error) {
	writeQRCodeError(c, err)
}

func writeQRCodeError(c *gin.Context, err error) {
	var appError *usecase.Error
	if !errors.As(err, &appError) {
		c.JSON(http.StatusOK, php.NewEnvelope(500, "二维码服务异常", nil))
		return
	}
	code := 400
	if appError.Code == usecase.CodeDependency {
		code = 500
	}
	c.JSON(http.StatusOK, php.NewEnvelope(code, appError.Message, nil))
}
