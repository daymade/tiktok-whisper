package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"tiktok-whisper/web"
)

func main() {
	// 创建并启动服务器
	server, err := web.NewServer(":8080")
	if err != nil {
		log.Fatal("创建服务器失败:", err)
	}

	// 优雅关闭处理
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("正在优雅关闭服务器...")
		server.Close()
		os.Exit(0)
	}()

	// 启动服务器
	log.Fatal(server.Start())
}
