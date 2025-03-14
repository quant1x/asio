package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	// 注册处理函数，处理所有路径的请求
	http.HandleFunc("/", echoHandler)

	// 启动HTTP服务，监听8080端口
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	// 读取整个请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// 构造响应数据结构
	response := map[string]interface{}{
		"method":  r.Method,
		"url":     r.URL.String(),
		"headers": r.Header,
		"body":    string(body),
	}

	// 设置JSON响应头
	w.Header().Set("Content-Type", "application/json")

	// 编码并发送JSON响应
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error generating response", http.StatusInternalServerError)
	}
	fmt.Println("response:", response)
}
