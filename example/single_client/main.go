package main

import (
	"fmt"
	"github.com/cr-mao/loric"
	"github.com/cr-mao/loric/cluster"
	"github.com/cr-mao/loric/cluster/client"
	"github.com/cr-mao/loric/example/internal/pb"
	"github.com/cr-mao/loric/log"
	"github.com/cr-mao/loric/network/tcp"
	"sync/atomic"
	"time"
)

var sendMsgCount int32 = 0
var startTime = time.Now()

func main() {
	// 创建容器
	container := loric.NewContainer()
	// 创建网关组件
	component := client.NewClient(
		client.WithClient(tcp.NewClient()),
	)
	// 初始化事件和路由
	initEvent(component.Proxy())
	initRoute(component.Proxy())
	// 添加网关组件
	container.Add(component)
	// 启动容器
	container.Serve()
}

func initEvent(proxy client.Proxy) {
	// 打开连接
	proxy.AddEventListener(cluster.Connect, onConnect)
	// 重新连接
	proxy.AddEventListener(cluster.Reconnect, onReconnect)
	// 断开连接
	proxy.AddEventListener(cluster.Disconnect, onDisconnect)
}

func onConnect(proxy client.Proxy) {
	log.Infof("connection is opened")

	err := proxy.Push(0, int32(pb.Route_Login), &pb.LoginReq{
		Token: "cr-mao",
	})
	if err != nil {
		log.Errorf("push create room message failed: %v", err)
	}
}

func onReconnect(proxy client.Proxy) {
	log.Infof("connection is reopened")
	err := proxy.Push(0, int32(pb.Route_Login), &pb.LoginReq{
		Token: "cr-mao",
	})
	if err != nil {
		log.Errorf("push login message failed: %v", err)
	}
}

func onDisconnect(proxy client.Proxy) {
	log.Infof("connection is closed")

	err := proxy.Reconnect()
	if err != nil {
		log.Errorf("reconnect failed: %v", err)
	}
}

func initRoute(proxy client.Proxy) {
	// 用户注册
	//proxy.AddRouteHandler(route.Register, registerHandler)
	// 用户登录
	proxy.AddRouteHandler(int32(pb.Route_Login), loginHandler)
	// 创建房间, 创建房间这个 服务器会进行绑定node操作，后续有状态 请求则打到该台服务器上.
	proxy.AddRouteHandler(int32(pb.Route_CreateRoom), createRoomHandler)
	//// 通知消息
	proxy.AddRouteHandler(int32(pb.Route_SendMsg), notifyMessageHandler)
}

func loginHandler(r client.Request) {
	res := &pb.LoginRes{}

	err := r.Parse(res)
	if err != nil {
		log.Errorf("invalid login response message, err: %v", err)
		return
	}

	switch res.Code {
	case pb.LoginCode_Failed:
		log.Error("user login failed")
		return
	}
	log.Infof("登录结果:%s", res.Code)

	err = r.Proxy().Push(0, int32(pb.Route_CreateRoom), &pb.CreateRoomReq{
		Name: "room1",
	})

	if err != nil {
		log.Errorf("push create room message failed: %v", err)
	}
}

func createRoomHandler(r client.Request) {
	res := &pb.CreateRoomRes{}

	err := r.Parse(res)
	if err != nil {
		log.Errorf("invalid login response message, err: %v", err)
		return
	}

	switch res.Code {
	case pb.CreateRoomCode_Failed, pb.CreateRoomCode_NameExists:
		log.Error("create room failed")
		return
	}
	log.Info("创建房间成功")
	err = r.Proxy().Push(0, int32(pb.Route_SendMsg), &pb.SendMsgReq{
		Msg: "hello",
	})
	if err != nil {
		log.Errorf("push message failed: %v", err)
	}

}

func notifyMessageHandler(r client.Request) {
	res := &pb.SendMsgRes{}
	err := r.Parse(res)
	if err != nil {
		log.Errorf("invalid login response message, err: %v", err)
		return
	}
	//log.Infof("%s say: %s", res.UserName, res.Msg)
	atomic.AddInt32(&sendMsgCount, 1)
	if atomic.LoadInt32(&sendMsgCount) < 100000 {
		err = r.Proxy().Push(0, int32(pb.Route_SendMsg), &pb.SendMsgReq{
			Msg: "hello",
		})
		//fmt.Println(time.Now().Unix(), sendMsgCount)
	} else {
		fmt.Println(time.Now().Sub(startTime).Milliseconds())
	}
}
