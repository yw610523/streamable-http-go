package main

import (
	"fmt"
	"net/http"
	"time"
)

// genericStreamHandler 通用流式响应处理器（非 SSE 协议，纯 HTTP 分块输出）
func genericStreamHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 设置通用流式响应头（无 SSE 专属格式）
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")        // 禁用缓存
	w.Header().Set("Connection", "keep-alive")         // 保持长连接
	w.Header().Set("Transfer-Encoding", "chunked")     // 显式声明分块传输
	w.Header().Set("Access-Control-Allow-Origin", "*") // 允许跨域（前端调试）

	// 2. 获取 Flusher 接口（强制刷新缓冲区，关键）
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "服务器不支持流式响应（Flusher 不可用）", http.StatusInternalServerError)
		return
	}

	// 3. 监听客户端断开连接（避免无效写入）
	clientQuit := r.Context().Done()

	// 4. 模拟实时分块推送数据
	counter := 0
	ticker := time.NewTicker(1 * time.Second) // 每秒推送1次
	defer ticker.Stop()

	fmt.Println("客户端建立通用流式连接")

	for {
		select {
		// 客户端断开连接，终止流
		case <-clientQuit:
			fmt.Println("客户端断开通用流式连接")
			return

		// 定时推送数据
		case <-ticker.C:
			counter++
			// 自定义流式数据格式（无 SSE 约束，纯文本）
			now := time.Now().Format("2006-01-02 15:04:05")
			data := fmt.Sprintf("第 %d 帧数据 | 时间：%s\n", counter, now)

			// 写入响应并强制刷新（核心：立即发送，不等待缓冲区满）
			_, err := fmt.Fprint(w, data)
			if err != nil {
				fmt.Printf("数据写入失败：%v\n", err)
				return
			}
			flusher.Flush() // 强制刷出缓冲区数据

			// 模拟流终止（推送10条后结束）
			if counter >= 10 {
				fmt.Fprint(w, "[流结束] 所有数据推送完成\n")
				flusher.Flush()
				fmt.Println("通用流式推送完成，主动关闭连接")
				return
			}
		}
	}
}

func main() {
	// 注册通用流式接口
	http.HandleFunc("/generic-stream", genericStreamHandler)
	// 注册静态文件服务（前端页面）
	http.Handle("/", http.FileServer(http.Dir(".")))

	// 启动服务
	addr := ":8080"
	fmt.Printf("通用流式 HTTP 服务启动：http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("服务启动失败：%v\n", err)
	}
}
