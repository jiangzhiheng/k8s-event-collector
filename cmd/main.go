package main

import (
	"fmt"
	"github.com/jiangzhiheng/k8s-event-collector/pkg/collector"
	grpcserver "github.com/jiangzhiheng/k8s-event-collector/pkg/grpc/server"
	"github.com/jiangzhiheng/k8s-event-collector/pkg/options"
	"github.com/jiangzhiheng/k8s-event-collector/pkg/signal"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"net/http"
	"time"
)

const (
	RESYNC = time.Minute * 5
)

func main() {
	opts := options.NewOptions()
	opts.AddFlags()
	if err := opts.Parse(); err != nil {
		klog.Fatalf("failed to parse commandline args,err:%s", err.Error())
	}

	group, stopChan := signal.SetupStopSignalContext()

	kubeConfig, err := clientcmd.BuildConfigFromFlags(opts.KubeMasterURL, opts.KubeConfigPath)
	if err != nil {
		klog.Fatalf("failed to build kubernetes cluster configuration,err:%s", err.Error())
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		klog.Fatalf("failed to build kubernetes client,err:%s", err.Error())
	}
	factory := informers.NewSharedInformerFactory(clientset, RESYNC)
	eventCollector := collector.NewEventCollector(clientset, factory, opts)
	factory.Start(stopChan)


	group.Go(func() error {
		if err := eventCollector.Run(stopChan); err != nil {
			return fmt.Errorf("eventCollector run err:%s", err.Error())
		}
		return nil
	})

	// grpc server
	if opts.UseGRPC {
		go grpcserver.Run()
	}

	klog.Infof("starting prometheus metrics server on http://localhost:%d", opts.MetricsPort)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// 研究下 prometheus default register！！！
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(fmt.Sprintf(":%d", opts.MetricsPort), nil); err != nil {
		klog.Fatal(err)
	}

	if err := group.Wait(); err != nil {
		klog.Fatal(err)
	}

}
