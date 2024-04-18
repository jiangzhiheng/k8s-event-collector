package client

import (
	"context"
	"github.com/google/uuid"
	eventgrpc "github.com/jiangzhiheng/k8s-event-collector/pkg/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestRunClient(t *testing.T) {
	// 创建 gRPC 客户端连接
	dial, err := grpc.Dial("127.0.0.1:8112", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial server: %v", err)
	}
	defer dial.Close()

	// 创建 gRPC 客户端
	rpcClient := eventgrpc.NewSearchEventServiceClient(dial)

	// 创建并传递 requestID
	requestId := uuid.New().String()

	md := metadata.Pairs("RequestID", requestId)

	ctx := metadata.NewOutgoingContext(context.Background(), md)

	// 调用 RunClient 方法
	events, err := rpcClient.GetResourceEvents(ctx, &eventgrpc.DescribeEventRequest{
		ResourceName:      "argocd-server-7965b94c48",
		ResourceType:      "ReplicaSet",
		ResourceNamespace: "argocd",
	})
	if err != nil {
		t.Errorf("error occurred when calling gRPC client: %v", err)
	}

	// 进行断言或其他测试逻辑
	if len(events.Event) == 0 {
		t.Errorf("expected non-empty events, got empty events")
	}
	for _, e := range events.Event {
		t.Logf("get event succ, event info:%s, Msg: %s, ReqID:%s", e.InvolvedObjectName, e.Message, requestId)
	}

}

/*
=== RUN   TestRunClient
    client_test.go:41: get event succ, event info:argocd-server-7965b94c48, Msg: Created pod: argocd-server-7965b94c48-vjmw9
--- PASS: TestRunClient (0.24s)
PASS
*/
