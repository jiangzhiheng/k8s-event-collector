package server

import (
	"fmt"
	eventgrpc "github.com/jiangzhiheng/k8s-event-collector/pkg/grpc"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"net"
	"os"
	"os/signal"
)

const GRPCServerPort = 8112

var server = &searchK8sEventServer{}

func Run() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", GRPCServerPort))
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(eventUnaryServerInterceptor),
	)
	// 注册服务到 grpc
	eventgrpc.RegisterSearchEventServiceServer(s, server)

	klog.Info("start grpc server...")
	go func() {
		if err := s.Serve(lis); err != nil {
			klog.Fatalf("failed to serve: %v", err)
		}
	}()

	// 等待中断信号，优雅地关闭服务器
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// 关闭服务器
	s.GracefulStop()
	klog.Info("grpc server stopped")
}
