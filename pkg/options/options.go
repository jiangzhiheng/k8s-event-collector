package options

import (
	"flag"
	"fmt"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
	"os"
)

type Options struct {
	KubeMasterURL  string
	KubeConfigPath string
	EventType      []string
	ESEndpoint     []string
	ESUsername     string
	ESPassword     string
	MetricsPort    int
	UseGRPC        bool
	UseHTTP        bool
	flag           *pflag.FlagSet
}

func NewOptions() *Options {
	return &Options{}
}

func (o *Options) AddFlags() {
	o.flag = pflag.NewFlagSet("", pflag.ExitOnError)
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	o.flag.AddGoFlagSet(klogFlags)

	o.flag.StringVar(&o.KubeMasterURL, "kubeMasterURL", "", "The URL of kubernetes apiserver to use as a master")
	o.flag.StringVar(&o.KubeConfigPath, "kubeConfigPath", "", "The path of kubernetes configuration file")
	o.flag.StringArrayVar(&o.ESEndpoint, "esEndpoint", []string{""}, "List of es endpoints.")
	o.flag.StringVar(&o.ESUsername, "esUsername", "elastic", "elastic username")
	o.flag.StringVar(&o.ESPassword, "esPassword", "", "elastic password.")
	o.flag.IntVar(&o.MetricsPort, "port", 9102, "Port to expose event metrics on")
	o.flag.BoolVar(&o.UseGRPC, "useGRPC", true, "enable grpc server")
	o.flag.BoolVar(&o.UseHTTP, "useHTTP", true, "enable http server")

	o.flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		o.flag.PrintDefaults()
	}
}

func (o *Options) Parse() error {
	return o.flag.Parse(os.Args)
}

func (o *Options) Usage() {
	o.flag.Usage()
}
