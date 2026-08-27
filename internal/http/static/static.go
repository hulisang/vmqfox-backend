package static

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed out/*
var distFS embed.FS

// AssetsFileSystem 提供前端静态资源文件系统（指向 out/assets）。
func AssetsFileSystem() (http.FileSystem, error) {
	sub, err := fs.Sub(distFS, "out/assets")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

// ReadIndexHTML 获取嵌入的 index.html 内容。
func ReadIndexHTML() ([]byte, error) {
	return distFS.ReadFile("out/index.html")
}
