package qrcodeimage

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"

	"github.com/hulisang/vmqfox-backend/internal/port"
	goqr "github.com/piglig/go-qr"
)

const (
	contentSizePixels = 180
	marginPixels      = 10
	maxImagePixels    = 24_000_000
)

type Codec struct{}

func New() Codec { return Codec{} }

func (Codec) EncodePNG(content string) ([]byte, error) {
	segments, err := goqr.MakeSegments(content)
	if err != nil {
		return nil, fmt.Errorf("拆分二维码内容失败: %w", err)
	}
	// go-qr v1.1.0 的标准入口会把版本 40 传给严格小于 40 的边界校验；
	// adapter 将上限收敛到 39，避免把第三方版本缺陷扩散到用例层。
	qr, err := goqr.EncodeSegments(segments, goqr.High, goqr.MinVersion, goqr.MaxVersion-1, -1, true)
	if err != nil {
		if errors.Is(err, goqr.ErrDataTooLong) {
			return nil, fmt.Errorf("%w: %v", port.ErrQRCodeContentTooLong, err)
		}
		return nil, fmt.Errorf("编码二维码失败: %w", err)
	}

	// Endroid 的 Margin 模式会把 180px 内容区按整数模块缩小，并把余量并入两侧边距。
	blockSize := contentSizePixels / qr.Size()
	if blockSize < 1 {
		return nil, port.ErrQRCodeContentTooLong
	}
	canvasSize := contentSizePixels + marginPixels*2
	renderedSize := blockSize * qr.Size()
	offset := (canvasSize - renderedSize) / 2

	img := image.NewGray(image.Rect(0, 0, canvasSize, canvasSize))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	black := image.NewUniform(color.Black)
	for y := 0; y < qr.Size(); y++ {
		for x := 0; x < qr.Size(); x++ {
			if !qr.Module(x, y) {
				continue
			}
			left := offset + x*blockSize
			top := offset + y*blockSize
			draw.Draw(img, image.Rect(left, top, left+blockSize, top+blockSize), black, image.Point{}, draw.Src)
		}
	}

	var output bytes.Buffer
	if err := png.Encode(&output, img); err != nil {
		return nil, fmt.Errorf("输出二维码 PNG 失败: %w", err)
	}
	return output.Bytes(), nil
}

func (Codec) Decode(imageData []byte) (string, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(imageData))
	if errors.Is(err, image.ErrFormat) {
		return "", fmt.Errorf("%w: %v", port.ErrQRCodeUnsupported, err)
	}
	if err != nil {
		return "", fmt.Errorf("%w: %v", port.ErrQRCodeNotFound, err)
	}
	width, height := int64(config.Width), int64(config.Height)
	if width <= 0 || height <= 0 || width > maxImagePixels/height {
		return "", fmt.Errorf("%w: %dx%d", port.ErrQRCodeImageTooLarge, config.Width, config.Height)
	}

	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("%w: %v", port.ErrQRCodeNotFound, err)
	}
	text, err := goqr.Decode(img)
	if err == nil && text != "" {
		return text, nil
	}
	if err == nil {
		return "", port.ErrQRCodeNotFound
	}

	switch {
	case errors.Is(err, goqr.ErrUnsupportedSymbol):
		return "", fmt.Errorf("%w: %v", port.ErrQRCodeUnsupported, err)
	case errors.Is(err, goqr.ErrNoQRCode), errors.Is(err, goqr.ErrDecodeFailed):
		return "", fmt.Errorf("%w: %v", port.ErrQRCodeNotFound, err)
	default:
		return "", fmt.Errorf("解码二维码失败: %w", err)
	}
}
