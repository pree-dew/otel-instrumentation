package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/acrmp/mcp"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"golang.org/x/time/rate"
)

// Logger wraps OpenTelemetry logger
type Logger struct {
	logger log.Logger
}

func NewLogger(name string) *Logger {
	return &Logger{
		logger: global.GetLoggerProvider().Logger(name),
	}
}

func (l *Logger) Info(ctx context.Context, msg string, attrs ...log.KeyValue) {
	var record log.Record
	record.SetTimestamp(time.Now())
	record.SetSeverity(log.SeverityInfo)
	record.SetBody(log.StringValue(msg))
	record.AddAttributes(attrs...)
	l.logger.Emit(ctx, record)
}

func (l *Logger) Error(ctx context.Context, msg string, attrs ...log.KeyValue) {
	var record log.Record
	record.SetTimestamp(time.Now())
	record.SetSeverity(log.SeverityError)
	record.SetBody(log.StringValue(msg))
	record.AddAttributes(attrs...)
	l.logger.Emit(ctx, record)
}

func (l *Logger) Debug(ctx context.Context, msg string, attrs ...log.KeyValue) {
	var record log.Record
	record.SetTimestamp(time.Now())
	record.SetSeverity(log.SeverityDebug)
	record.SetBody(log.StringValue(msg))
	record.AddAttributes(attrs...)
	l.logger.Emit(ctx, record)
}

func (l *Logger) Warn(ctx context.Context, msg string, attrs ...log.KeyValue) {
	var record log.Record
	record.SetTimestamp(time.Now())
	record.SetSeverity(log.SeverityWarn)
	record.SetBody(log.StringValue(msg))
	record.AddAttributes(attrs...)
	l.logger.Emit(ctx, record)
}

// setupOTelLogging configures standard OpenTelemetry logging
func setupOTelLogging(ctx context.Context) (func(), error) {
	res, err := resource.New(ctx,
		resource.WithFromEnv(), // Standard OTEL env vars
		resource.WithAttributes(
			semconv.ServiceName(getEnv("OTEL_SERVICE_NAME", "mcp-server")),
			semconv.ServiceVersion(getEnv("OTEL_SERVICE_VERSION", "1.0.0")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var exporter sdklog.Exporter
	logsExporter := getEnv("OTEL_LOGS_EXPORTER", "console")

	switch logsExporter {
	case "otlp":
		exporter, err = otlploghttp.New(ctx,
			otlploghttp.WithEndpointURL(getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318")),
			otlploghttp.WithHeaders(map[string]string{
				"Authorization": getEnv("OTEL_EXPORTER_OTLP_AUTH_TOKEN", ""),
			}),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
	case "console":
		// CRITICAL: Use stderr to avoid MCP stdio protocol interference
		exporter, err = stdoutlog.New(
			stdoutlog.WithWriter(os.Stderr),
			stdoutlog.WithPrettyPrint(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create console exporter: %w", err)
		}
	case "none":
		exporter = &noopExporter{}
	default:
		return nil, fmt.Errorf("unsupported OTEL_LOGS_EXPORTER: %s", logsExporter)
	}

	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	global.SetLoggerProvider(loggerProvider)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := loggerProvider.Shutdown(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error shutting down logger provider: %v\n", err)
		}
	}

	return cleanup, nil
}

// noopExporter for disabled logging
type noopExporter struct{}

func (e *noopExporter) Export(ctx context.Context, records []sdklog.Record) error {
	return nil
}

func (e *noopExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *noopExporter) ForceFlush(ctx context.Context) error {
	return nil
}

// Global logger instance
var logger *Logger

// EchoTool implements the echo functionality
func EchoTool() func(mcp.CallToolRequestParams) (mcp.CallToolResult, error) {
	return func(params mcp.CallToolRequestParams) (mcp.CallToolResult, error) {
		logger.Info(context.Background(), "Echo tool called",
			log.String("tool_name", "echo"),
		)

		start := time.Now()

		// Extract message parameter
		message, ok := params.Arguments["message"].(string)
		if !ok {
			logger.Error(context.Background(), "Missing or invalid message parameter")
			return mcp.CallToolResult{
				Content: []any{
					mcp.TextContent{
						Text: "Error: message parameter is required",
						Type: "text",
					},
				},
			}, nil
		}

		logger.Debug(context.Background(), "Processing echo request",
			log.String("message", message),
		)

		duration := time.Since(start)
		logger.Info(context.Background(), "Echo tool completed",
			log.String("tool_name", "echo"),
			log.Int64("duration_ms", duration.Milliseconds()),
		)

		return mcp.CallToolResult{
			Content: []any{
				mcp.TextContent{
					Text: fmt.Sprintf("Echo: %s", message),
					Type: "text",
				},
			},
		}, nil
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	ctx := context.Background()

	// Setup standard OpenTelemetry logging
	cleanup, err := setupOTelLogging(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	// Initialize global logger
	logger = NewLogger("mcp-server")

	serviceName := getEnv("OTEL_SERVICE_NAME", "mcp-server")
	serviceVersion := getEnv("OTEL_SERVICE_VERSION", "1.0.0")

	logger.Info(ctx, "Starting MCP Server with OpenTelemetry logging",
		log.String("service_name", serviceName),
		log.String("service_version", serviceVersion),
		log.String("logs_exporter", getEnv("OTEL_LOGS_EXPORTER", "console")),
	)

	info := mcp.Implementation{
		Name: "last9-mcp",
	}

	echoToolDescription := "Echo back a message"
	tools := []mcp.ToolDefinition{
		{
			Metadata: mcp.Tool{
				Name:        "echo",
				Description: &echoToolDescription,
				InputSchema: mcp.ToolInputSchema{
					Type: "object",
					Properties: mcp.ToolInputSchemaProperties{
						"message": {
							"type":        "string",
							"description": "The message to echo back",
						},
					},
				},
			},
			Execute:   EchoTool(),
			RateLimit: rate.NewLimiter(rate.Every(1*time.Second), 5),
		},
	}

	// Create MCP server using acrmp/mcp
	server := mcp.NewServer(info, tools)

	logger.Info(ctx, "MCP Server configured with tools",
		log.String("available_tools", "echo, sha256, ping"),
	)

	// Start the server using stdio transport
	// This handles all MCP protocol details and logs to stderr via OpenTelemetry
	logger.Info(ctx, "Starting MCP server with stdio transport")

	server.Serve()
}
