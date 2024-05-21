package main

import (
	"fmt"
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
	mux.Handle("/", http.FileServer(http.Dir("D:\\Project\\vue\\http-server\\dist")))
	mux.HandleFunc("/setCookie", src.HandleLoginRequest)
	fmt.Println("Server listening on port 8800...")
	if err := http.ListenAndServe(":8800", mux); err != nil {
		fmt.Println(err)
	}

}
