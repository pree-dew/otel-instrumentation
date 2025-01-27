from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor, SimpleLogRecordProcessor, ConsoleLogExporter
from opentelemetry.instrumentation.logging import LoggingInstrumentor
from opentelemetry._logs import set_logger_provider
import logging
import time

resource = Resource.create({
    "service.name": "example-service",
    "service.version": "0.1.0"
})

logger_provider = LoggerProvider(resource=resource)
console_exporter = ConsoleLogExporter()
logger_provider.add_log_record_processor(SimpleLogRecordProcessor(console_exporter))
set_logger_provider(logger_provider)

handler = LoggingHandler(level=logging.NOTSET, logger_provider=logger_provider)

logging.basicConfig(
        level=logging.INFO,
        handlers=[handler]
    )

# Attach OTLP handler to root logger
logging.getLogger().addHandler(handler)

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
   logger.info("Application starting")
   i = 0
   while True:
       i += 1
       process_task(i)
   logger.info("Application shutdown")

if __name__ == "__main__":
   main()
