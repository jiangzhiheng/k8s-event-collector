package client

import (
	"context"
	"fmt"
	eventgrpc "github.com/jiangzhiheng/k8s-event-collector/pkg/grpc"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"time"
)

const GRPCServerPort = 8112

func RunClient() {
	dial, err := grpc.Dial(fmt.Sprintf(":%v", GRPCServerPort), grpc.WithInsecure())
	if err != nil {
		klog.Fatalf("cannot dial server: %v", err)
		return
	}
	defer dial.Close()

	rpcClient := eventgrpc.NewSearchEventServiceClient(dial)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	events, err := rpcClient.GetResourceEvents(ctx, &eventgrpc.DescribeEventRequest{})
	if err != nil {
		klog.Errorf("error happen when call gRPC client:" + err.Error())
		return
	}
	fmt.Print(events)
}
