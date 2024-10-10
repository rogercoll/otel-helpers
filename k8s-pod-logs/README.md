# K8s Pod logs format generator

The generated output should be used with the [OpenTelemtry Container Operator](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/stanza/operator/parser/container)

Supported container runtime format:

- [x] Containerd

## Usage

```
$ echo '37.18.133.47 - - [25/Jun/2024:15:16:56 +0000] "POST /flagservice/flagd.evaluation.v1.Service/EventStream HTTP/1.1" 400 47 "http://37.18.133.47/" "Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/127.0" 522 12.065 [default-my-otel-demo-frontendproxy-8080] [] 10.80.0.67:8080 47 12.065 200 797ee0551e7c23b541757e13b5b6b4e6' | go run cmd/main.go


// Output
2024-10-10T10:59:52.029269771Z stdout F 37.18.133.47 - - [25/Jun/2024:15:16:56 +0000] "POST /flagservice/flagd.evaluation.v1.Service/EventStream HTTP/1.1" 400 47 "http://37.18.133.47/" "Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/127.0" 522 12.065 [default-my-otel-demo-frontendproxy-8080] [] 10.80.0.67:8080 47 12.065 200 797ee0551e7c23b541757e13b5b6b4e6
```
