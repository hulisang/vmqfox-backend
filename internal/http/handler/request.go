package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
)

const maxScalarRequestBody = 1 << 20

// scalarRequestParams 统一读取 query、form 和 JSON 标量参数，保留签名所依赖的原始数字文本。
func scalarRequestParams(c *gin.Context) (map[string]string, error) {
	params := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	if c.Request.Body != nil {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxScalarRequestBody)
	}

	mediaType, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") {
		if c.Request.Body == nil {
			return params, nil
		}
		decoder := json.NewDecoder(c.Request.Body)
		decoder.UseNumber()
		var body map[string]any
		if err := decoder.Decode(&body); err != nil {
			if errors.Is(err, io.EOF) {
				return params, nil
			}
			return nil, err
		}
		var trailing any
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			if err == nil {
				return nil, errors.New("请求体只能包含一个 JSON 对象")
			}
			return nil, err
		}
		for key, value := range body {
			text, err := scalarRequestValue(value)
			if err != nil {
				return nil, fmt.Errorf("参数 %s 不是标量: %w", key, err)
			}
			params[key] = text
		}
		return params, nil
	}

	if err := c.Request.ParseForm(); err != nil {
		return nil, err
	}
	for key, values := range c.Request.PostForm {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	return params, nil
}

func scalarRequestValue(value any) (string, error) {
	switch value.(type) {
	case nil:
		return "", nil
	case string, json.Number, float64, bool:
		return php.StringValue(value), nil
	default:
		return "", errors.New("仅支持字符串、数字、布尔值或 null")
	}
}
