package sentrytemporal

import (
	"github.com/getsentry/sentry-go"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type workflowInboundInterceptor struct {
	interceptor.WorkflowInboundInterceptorBase
	root *workerInterceptor
}

// ExecuteWorkflow implements WorkflowInboundInterceptor.ExecuteWorkflow.
func (w *workflowInboundInterceptor) ExecuteWorkflow(
	ctx workflow.Context,
	in *interceptor.ExecuteWorkflowInput,
) (ret interface{}, err error) {
	configureScope := func(scope *sentry.Scope) {
		info := workflow.GetInfo(ctx)
		scope.SetContext("workflow info", mustStruct2Map(info))
		scope.SetContext("execute workflow input", mustStruct2Map(in))

		scope.SetTag("temporal_io_kind", "ExecuteWorkflow")

		scope.SetFingerprint(
			[]string{
				info.TaskQueueName,
				info.WorkflowType.Name,
				"{{ default }}",
			},
		)

		if w.root.options.WorkflowScopeCustomizer != nil {
			w.root.options.WorkflowScopeCustomizer(ctx, scope, err)
		}
	}

	defer func() {
		if x := recover(); x != nil {
			hub := w.root.hub.Clone()
			hub.ConfigureScope(configureScope)
			hub.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetLevel(sentry.LevelFatal)
			})
			_ = hub.Recover(x)
			panic(x)
		}
	}()

	ret, err = w.Next.ExecuteWorkflow(ctx, in)
	if err != nil {
		if isContinueAsNewError(err) {
			return
		}

		if temporal.IsCanceledError(err) || temporal.IsTimeoutError(err) || temporal.IsTerminatedError(err) {
			return
		}

		if skipper := w.root.options.WorkflowErrorSkipper; skipper != nil && skipper(ctx, err) {
			return
		}

		hub := w.root.hub.Clone()
		hub.ConfigureScope(configureScope)
		_ = hub.CaptureException(err)
	}

	return
}

// HandleQuery implements WorkflowInboundInterceptor.HandleQuery.
func (w *workflowInboundInterceptor) HandleQuery(
	ctx workflow.Context,
	in *interceptor.HandleQueryInput,
) (ret interface{}, err error) {
	configureScope := func(scope *sentry.Scope) {
		info := workflow.GetInfo(ctx)
		scope.SetContext("workflow info", mustStruct2Map(info))
		scope.SetContext("handle query input", mustStruct2Map(in))

		scope.SetTag("temporal_io_kind", "HandleQuery")

		scope.SetFingerprint(
			[]string{
				info.TaskQueueName,
				info.WorkflowType.Name,
				"{{ default }}",
			},
		)
	}

	defer func() {
		if x := recover(); x != nil {
			hub := w.root.hub.Clone()
			hub.ConfigureScope(configureScope)
			hub.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetLevel(sentry.LevelFatal)
			})
			_ = hub.Recover(x)
			panic(x)
		}
	}()

	ret, err = w.Next.HandleQuery(ctx, in)
	if err != nil {
		if temporal.IsCanceledError(err) || temporal.IsTimeoutError(err) || temporal.IsTerminatedError(err) {
			return
		}

		if skipper := w.root.options.WorkflowErrorSkipper; skipper != nil && skipper(ctx, err) {
			return
		}

		hub := w.root.hub.Clone()
		hub.ConfigureScope(configureScope)
		_ = hub.CaptureException(err)
	}

	return
}
