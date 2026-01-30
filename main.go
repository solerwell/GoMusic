package main

import (
	"GoMusic/handler"
	"GoMusic/misc/config"
	"GoMusic/misc/log"
	_ "GoMusic/repo/db"
)

func main() {
	// 获取配置
	cfg := config.GetConfig()

	// 启动路由
	r := handler.NewRouter()
	serverAddr := cfg.GetServerAddr()
	log.Infof("服务器启动于端口: %s", serverAddr)

	if err := r.Run(serverAddr); err != nil {
		log.Errorf("启动服务器失败: %v", err)
		panic(err)
	}
}
