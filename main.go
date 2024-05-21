package main

import (
	"github.com/sirupsen/logrus"
	"httpServer/src"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/download/", src.HandleDownloadRequest)
	mux.HandleFunc("/delete", src.HandleRemoveRequest)
	mux.HandleFunc("/query", src.HandleQueryRequest)
	mux.HandleFunc("/upload", src.HandleUploadRequest)
	mux.HandleFunc("/createDir", src.HandleMkdirRequest)
	mux.HandleFunc("/base", src.HandleBaseRequest)
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.HandleFunc("/setCookie", src.HandleLoginRequest)
	logrus.Info("欢迎使用Moki-HttpServer服务器!")
	logrus.Info("服务已启动, 并开始监听端口8800...")
	if err := http.ListenAndServe(":8800", mux); err != nil {
		logrus.Error("启动发生异常, 异常为: ", err)
	}

}
