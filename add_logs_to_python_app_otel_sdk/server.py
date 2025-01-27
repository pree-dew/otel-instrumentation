from opentelemetry.sdk.resources import Resource
from opentelemetry._logs import set_logger_provider
from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor, ConsoleLogExporter
from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
import logging
import time

# Create resource
resource = Resource.create({
    "service.name": "example-service",
    "service.version": "0.1.0"
})

# Set up logger provider
logger_provider = LoggerProvider(resource=resource)

# Create OTLP log exporter
otlp_log_exporter = OTLPLogExporter(
    endpoint="http://localhost:4317",
    insecure=True
)

logger_provider.add_log_record_processor(
    BatchLogRecordProcessor(otlp_log_exporter)
)

set_logger_provider(logger_provider)

# Create and configure handler
handler = LoggingHandler(level=logging.NOTSET, logger_provider=logger_provider)

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    handlers=[handler],
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

# Get logger
logger = logging.getLogger("myapp")

def process_task(task_id):
    logger.info(
        f"Processing task {task_id}",
        extra={
            "task_id": task_id,
            "processor": "main",
            "timestamp": time.time()
        }
    )

    # Simulate work
    time.sleep(1)
   
    logger.info(
        f"Completed task {task_id}",
        extra={
            "task_id": task_id,
            "status": "completed"
        }
    )

def main():
    try:
        print("Starting application")
        logger.info("Application starting")
        
        # Adding counter and sleep to prevent overwhelming
        for i in range(10):  # Limited to 10 iterations for testing
            print(f"Processing task {i}")
            process_task(i)
            time.sleep(0.1)  # Small delay between tasks
            
    except KeyboardInterrupt:
        print("Shutting down...")
    finally:
        logger.info("Application shutdown")
        # Allow time for logs to be exported
        time.sleep(5)

if __name__ == "__main__":
    main()
