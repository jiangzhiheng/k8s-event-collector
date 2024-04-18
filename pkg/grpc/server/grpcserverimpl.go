package server

import (
	"context"
	"github.com/jiangzhiheng/k8s-event-collector/pkg/elasticsearch"
	eventgrpc "github.com/jiangzhiheng/k8s-event-collector/pkg/grpc"
	"github.com/jiangzhiheng/k8s-event-collector/pkg/metrics"
	"github.com/jiangzhiheng/k8s-event-collector/pkg/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"k8s.io/klog/v2"
	"time"
)

type searchK8sEventServer struct {
	eventgrpc.UnimplementedSearchEventServiceServer
}

func (s *searchK8sEventServer) GetResourceEvents(ctx context.Context, req *eventgrpc.DescribeEventRequest) (*eventgrpc.DescribeEventResponse, error) {
	opts := options.NewOptions()
	opts.AddFlags()
	if err := opts.Parse(); err != nil {
		klog.Fatalf("failed to parse commandline args,err:%s", err.Error())
	}

	esClient, err := elasticsearch.NewES(&elasticsearch.ESConfig{
		Hosts:    opts.ESEndpoint,
		Username: opts.ESUsername,
		Password: opts.ESPassword,
	})
	if err != nil {
		klog.Errorf("Init elastic connect failed")
	}

	eventDocuments, totalCount, err := esClient.SearchEventDocuments(req.ResourceNamespace, req.ResourceType, req.ResourceName)
	if err != nil {
		return nil, err
	}

	events := make([]*eventgrpc.Event, len(eventDocuments))
	for i, doc := range eventDocuments {
		events[i] = &eventgrpc.Event{
			Type:                    doc.Type,
			Message:                 doc.Message,
			Reason:                  doc.Reason,
			Action:                  doc.Action,
			Name:                    doc.Name,
			Kind:                    doc.Kind,
			RelatedName:             doc.RelatedName,
			RelatedKind:             doc.RelatedKind,
			RelatedNamespace:        doc.RelatedNamespace,
			InvolvedObjectNamespace: doc.InvolvedObjectNamespace,
			InvolvedObjectKind:      doc.InvolvedObjectKind,
			InvolvedObjectName:      doc.InvolvedObjectName,
			Count:                   doc.Count,
		}
	}

	// 增加一次调用成功的指标
	metrics.AddSearchK8sEventServerTotal(req.ResourceNamespace)

	return &eventgrpc.DescribeEventResponse{
		Event:      events,
		TotalCount: uint32(totalCount),
	}, nil
}

// 实现 unary interceptors
func eventUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Pre-processing logic
	s := time.Now()

	// 获取 RequestID
	md, ok := metadata.FromIncomingContext(ctx)
	var RequestID string
	if ok {
		RequestID = md.Get("RequestID")[0]
	} else {

	}
	// 可以在这里对 req 进行校验或者修改
	m, err := handler(ctx, req)

	// Post processing logic
	klog.Infof("Method: %s, req: %s, resp: %s, latency: %s, RequestID:%s\n",
		info.FullMethod, req, m, time.Now().Sub(s), RequestID)

	return m, err
}
