// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package k8sattributesprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"k8s.io/client-go/informers"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/k8sconfig"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor/internal/kube"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor/internal/metadata"
)

var kubeClientProvider = kube.ClientProvider(nil)
var consumerCapabilities = consumer.Capabilities{MutatesData: true}
var defaultExcludes = ExcludeConfig{Pods: []ExcludePodConfig{{Name: "jaeger-agent"}, {Name: "jaeger-collector"}}}

type factory struct {
	sync.Mutex
	factories      map[string]informers.SharedInformerFactory
	processorCount map[string]int
}

// NewFactory returns a new factory for the k8s processor.
func NewFactory() processor.Factory {
	f := &factory{
		factories:      make(map[string]informers.SharedInformerFactory),
		processorCount: make(map[string]int),
	}
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithTraces(f.createTracesProcessor, metadata.TracesStability),
		processor.WithMetrics(f.createMetricsProcessor, metadata.MetricsStability),
		processor.WithLogs(f.createLogsProcessor, metadata.LogsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		APIConfig: k8sconfig.APIConfig{AuthType: k8sconfig.AuthTypeServiceAccount},
		Exclude:   defaultExcludes,
		Extract: ExtractConfig{
			Metadata: enabledAttributes(),
		},
	}
}

func (f *factory) createTracesProcessor(
	ctx context.Context,
	params processor.Settings,
	cfg component.Config,
	next consumer.Traces,
) (processor.Traces, error) {
	return f.createTracesProcessorWithOptions(ctx, params, cfg, next)
}

func (f *factory) createLogsProcessor(
	ctx context.Context,
	params processor.Settings,
	cfg component.Config,
	nextLogsConsumer consumer.Logs,
) (processor.Logs, error) {
	return f.createLogsProcessorWithOptions(ctx, params, cfg, nextLogsConsumer)
}

func (f *factory) createMetricsProcessor(
	ctx context.Context,
	params processor.Settings,
	cfg component.Config,
	nextMetricsConsumer consumer.Metrics,
) (processor.Metrics, error) {
	return f.createMetricsProcessorWithOptions(ctx, params, cfg, nextMetricsConsumer)
}

func (f *factory) createTracesProcessorWithOptions(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	next consumer.Traces,
	options ...option,
) (processor.Traces, error) {
	kp := f.createKubernetesProcessor(set, cfg, options...)

	return processorhelper.NewTraces(
		ctx,
		set,
		cfg,
		next,
		kp.processTraces,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(kp.Start),
		processorhelper.WithShutdown(kp.Shutdown))
}

func (f *factory) createMetricsProcessorWithOptions(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextMetricsConsumer consumer.Metrics,
	options ...option,
) (processor.Metrics, error) {
	kp := f.createKubernetesProcessor(set, cfg, options...)

	return processorhelper.NewMetrics(
		ctx,
		set,
		cfg,
		nextMetricsConsumer,
		kp.processMetrics,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(kp.Start),
		processorhelper.WithShutdown(kp.Shutdown))
}

func (f *factory) createLogsProcessorWithOptions(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextLogsConsumer consumer.Logs,
	options ...option,
) (processor.Logs, error) {
	kp := f.createKubernetesProcessor(set, cfg, options...)

	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextLogsConsumer,
		kp.processLogs,
		processorhelper.WithCapabilities(consumerCapabilities),
		processorhelper.WithStart(kp.Start),
		processorhelper.WithShutdown(kp.Shutdown))
}

func (f *factory) createKubernetesProcessor(
	params processor.Settings,
	cfg component.Config,
	options ...option,
) *kubernetesprocessor {
	kp := &kubernetesprocessor{logger: params.Logger,
		cfg:               cfg,
		options:           options,
		telemetrySettings: params.TelemetrySettings,
	}

	return kp
}

func createProcessorOpts(cfg component.Config) []option {
	oCfg := cfg.(*Config)
	var opts []option
	if oCfg.Passthrough {
		opts = append(opts, withPassthrough())
	}

	// extraction rules
	opts = append(opts, withExtractMetadata(oCfg.Extract.Metadata...))
	opts = append(opts, withExtractLabels(oCfg.Extract.Labels...))
	opts = append(opts, withExtractAnnotations(oCfg.Extract.Annotations...))

	// filters
	opts = append(opts, withFilterNode(oCfg.Filter.Node, oCfg.Filter.NodeFromEnvVar))
	opts = append(opts, withFilterNamespace(oCfg.Filter.Namespace))
	opts = append(opts, withFilterLabels(oCfg.Filter.Labels...))
	opts = append(opts, withFilterFields(oCfg.Filter.Fields...))
	opts = append(opts, withAPIConfig(oCfg.APIConfig))

	opts = append(opts, withExtractPodAssociations(oCfg.Association...))

	opts = append(opts, withExcludes(oCfg.Exclude))

	return opts
}
