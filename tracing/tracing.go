package tracing

import (
	"fmt"
	"io"

	"github.com/uber/jaeger-client-go/config"

	"github.com/opentracing/opentracing-go"
)

// Initializes a tracer configuring the servicename as the app name
func MustInit(appName string) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		ServiceName: appName,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}
