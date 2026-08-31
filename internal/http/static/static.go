package static

import (
	"embed"
	"io/fs"
	"net/http"
)

// 使用 all: 前缀，使仓库中仅有 .gitkeep 占位时也能通过编译。
// 真实资源由 web 构建（pnpm build → ../internal/http/static/out）或
// Docker 的 web-builder 阶段写入本目录。
//
//go:embed all:out
var distFS embed.FS

// AssetsFileSystem 提供前端静态资源文件系统（指向 out/assets）。
// 尚未执行前端构建时返回错误，由路由层决定跳过静态托管而不是伪造页面。
func AssetsFileSystem() (http.FileSystem, error) {
	sub, err := fs.Sub(distFS, "out/assets")
	if err != nil {
		return nil, err
	}
	// fs.Sub 对不存在的目录也会成功返回，这里通过实际读取确认资源已就位。
	if _, err := fs.ReadDir(sub, "."); err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

// ReadIndexHTML 获取嵌入的 index.html 内容。
func ReadIndexHTML() ([]byte, error) {
	return distFS.ReadFile("out/index.html")
}
