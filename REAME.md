## 这个项目做了什么
本项目作为一个学习项目，实现收集k8s的 event，并投递到 elasticsearch 中，并提供 grpc 查询接口。

## 部署
1. 本地运行
```shell
make bin
# 支持的启动参数
 _output/bin/event-collector -h
Usage of _output/bin/event-collector:
      --add_dir_header                   If true, adds the file directory to the header of the log messages
      --alsologtostderr                  log to standard error as well as files (no effect when -logtostderr=true)
      --esEndpoint stringArray           List of es endpoints.
      --esPassword string                elastic password.
      --esUsername string                elastic username (default "elastic")
      --kubeConfigPath string            The path of kubernetes configuration file
      --kubeMasterURL string             The URL of kubernetes apiserver to use as a master
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory (no effect when -logtostderr=true)
      --log_file string                  If non-empty, use this log file (no effect when -logtostderr=true)
      --log_file_max_size uint           Defines the maximum size a log file can grow to (no effect when -logtostderr=true). Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                      log to standard error instead of files (default true)
      --one_output                       If true, only write logs to their native severity level (vs also writing to each lower severity level; no effect when -logtostderr=true)
      --port int                         Port to expose event metrics on (default 9102)
      --skip_headers                     If true, avoid header prefixes in the log messages
      --skip_log_headers                 If true, avoid headers when opening log files (no effect when -logtostderr=true)
      --stderrthreshold severity         logs at or above this threshold go to stderr when writing to files and stderr (no effect when -logtostderr=true or -alsologtostderr=true) (default 2)
      --useGRPC                          enable grpc server (default true)
      --useHTTP                          enable http server (default true)
  -v, --v Level                          number for the log level verbosity
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

2. 部署到k8s集群中运行

## 开发指引
如果要使用其它语言调用日志查询接口，可参考如下命令生成对应语言的grpc代码

`protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=./pkg/grpc/ --go-grpc_opt==paths=source_relative pkg/grpc/service.proto`