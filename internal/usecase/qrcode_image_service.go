package usecase

import (
	"errors"
	"strings"

	"github.com/hulisang/vmqfox-backend/internal/domain/qrcode"
	"github.com/hulisang/vmqfox-backend/internal/port"
)

type QRCodePNG struct {
	Data     []byte
	CacheKey string
}

type QRCodeImageServiceDeps struct {
	Codec port.QRCodeImageCodec
}

type QRCodeImageService struct {
	codec port.QRCodeImageCodec
}

func NewQRCodeImageService(deps QRCodeImageServiceDeps) (*QRCodeImageService, error) {
	if deps.Codec == nil {
		return nil, errors.New("二维码图像用例依赖不完整")
	}
	return &QRCodeImageService{codec: deps.Codec}, nil
}

func (s *QRCodeImageService) GeneratePNG(content string) (QRCodePNG, error) {
	if strings.TrimSpace(content) == "" {
		return QRCodePNG{}, fail(CodeInvalidArgument, "URL参数不能为空")
	}
	data, err := s.codec.EncodePNG(content)
	if errors.Is(err, port.ErrQRCodeContentTooLong) {
		return QRCodePNG{}, fail(CodeInvalidArgument, "二维码内容过长")
	}
	if err != nil {
		return QRCodePNG{}, wrap(CodeDependency, "生成二维码失败", err)
	}
	return QRCodePNG{Data: data, CacheKey: qrcode.CacheKey(content)}, nil
}

func (s *QRCodeImageService) Decode(imageData []byte) (string, error) {
	if len(imageData) == 0 {
		return "", fail(CodeInvalidArgument, "二维码数据不能为空，请选择文件上传")
	}
	text, err := s.codec.Decode(imageData)
	switch {
	case errors.Is(err, port.ErrQRCodeNotFound):
		return "", fail(CodeInvalidArgument, "图片中未识别到二维码")
	case errors.Is(err, port.ErrQRCodeUnsupported):
		return "", fail(CodeInvalidArgument, "二维码图片格式暂不支持")
	case errors.Is(err, port.ErrQRCodeImageTooLarge):
		return "", fail(CodeInvalidArgument, "二维码图片尺寸过大")
	case err != nil:
		return "", wrap(CodeDependency, "解析二维码失败", err)
	case text == "":
		return "", fail(CodeInvalidArgument, "图片中未识别到二维码")
	default:
		return text, nil
	}
}
