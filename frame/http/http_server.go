package http

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"wataru.com/gogo/config"
	httpcontext "wataru.com/gogo/frame/context"
	"wataru.com/gogo/frame/db"
	"wataru.com/gogo/frame/router"
	"wataru.com/gogo/frame/task"
	"wataru.com/gogo/logger"
	"wataru.com/gogo/redis"
	"wataru.com/gogo/util"
)

const (
	BANNER = `
	▄▄ •        ▄▄ •       
   ▐█ ▀ ▪▪     ▐█ ▀ ▪▪     
   ▄█ ▀█▄ ▄█▀▄ ▄█ ▀█▄ ▄█▀▄ 
   ▐█▄▪▐█▐█▌.▐▌▐█▄▪▐█▐█▌.▐▌
   ·▀▀▀▀  ▀█▄▀▪·▀▀▀▀  ▀█▄▀▪

                    GOGO v1.0.0`
)

type HttpServer struct {
	router *router.Router
}

func (server *HttpServer) Router() *router.Router {
	return server.router
}

func (server *HttpServer) CreateRouter() {
	server.router = router.NewRouter()
}

func (server *HttpServer) doInitializer() {
	for i := httpcontext.InitializerList.Front(); i != nil; i = i.Next() {
		br := i.Value.(httpcontext.Initializer)
		br.F()
	}
}

func (server *HttpServer) Run() {
	startTime := time.Now().UnixNano()
	// init logger
	logConf := util.ValueOrDefault((*config.GlobalConfig.Map)["log"], make(map[interface{}]interface{})).(map[interface{}]interface{})
	logger.Config(logConf, logger.InfoLevel, logger.ByDay, 2)

	logger.Raw(BANNER)

	// init server
	serverConf := util.ValueOrDefault((*config.GlobalConfig.Map)["server"], make(map[interface{}]interface{})).(map[interface{}]interface{})
	logger.Info("Run in %s mode", config.GlobalConfig.Env)
	// router.Handle("/hello/golang/", &BaseHander{})

	// 初始化数据源连接
	_, dbCancel := db.InitDb()
	defer dbCancel()

	// 初始化Redis
	redisConf := util.ValueOrDefault((*config.GlobalConfig.Map)["redis"], make(map[interface{}]interface{})).(map[interface{}]interface{})
	if len(redisConf) > 0 {
		redis.NewClient(redisConf)
	}

	// 初始化初始化器
	server.doInitializer()

	// 初始化路由中间件
	server.router.InitRouterMiddleware()
	server.router.LogRouterSummary()

	// 启动定时任务
	task.StartTaskSchedule()

	port := util.ValueOrDefault(serverConf["port"], 8080).(int)
	netSrv := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: server.router,
	}
	go func() {
		// service connections
		if err := netSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen: %s", err)
			os.Exit(1)
		}
	}()
	logger.Info("Started server [%d] in %.3f seconds", port, float32(time.Now().UnixNano()-startTime)/1e9)

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	logger.Info("Shutdown server ...")

	ctx, httpCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer httpCancel()
	exitServer(netSrv, &ctx)
}

func exitServer(netSrv *http.Server, ctx *context.Context) {
	exitCode := 0
	if err := netSrv.Shutdown(*ctx); err != nil {
		logger.Error("Server force stop: %v", err)
		exitCode = 1
	}
	logger.Info("Server exiting")
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}