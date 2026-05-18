import logging
import logging.handlers
import os
import sys

import structlog


class _LevelFilter(logging.Filter):
    """Pass only records at or above min_level but below max_level."""

    def __init__(self, min_level: int, max_level: int = logging.CRITICAL + 1):
        super().__init__()
        self.min_level = min_level
        self.max_level = max_level

    def filter(self, record: logging.LogRecord) -> bool:
        return self.min_level <= record.levelno < self.max_level


def _make_rotating_handler(
    path: str, min_level: int, max_level: int = logging.CRITICAL + 1
) -> logging.handlers.RotatingFileHandler:
    handler = logging.handlers.RotatingFileHandler(
        path,
        maxBytes=10 * 1024 * 1024,  # 10 MB
        backupCount=5,
        encoding="utf-8",
    )
    handler.setLevel(min_level)
    handler.addFilter(_LevelFilter(min_level, max_level))
    handler.setFormatter(logging.Formatter("%(message)s"))
    return handler


def setup_logging(debug: bool = False, log_dir: str | None = None) -> None:
    log_level = logging.DEBUG if debug else logging.INFO

    # Shared pre-processors applied before the final renderer
    shared_processors: list = [
        structlog.contextvars.merge_contextvars,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.processors.StackInfoRenderer(),
        structlog.dev.set_exc_info,
        structlog.processors.TimeStamper(fmt="iso"),
    ]

    # Route structlog through stdlib so all records reach the file handlers
    structlog.configure(
        processors=shared_processors
        + [
            structlog.stdlib.ProcessorFormatter.wrap_for_formatter,
        ],
        logger_factory=structlog.stdlib.LoggerFactory(),
        wrapper_class=structlog.stdlib.BoundLogger,
        cache_logger_on_first_use=True,
    )

    # Stdlib formatter that renders structlog-wrapped records
    formatter = structlog.stdlib.ProcessorFormatter(
        processor=structlog.dev.ConsoleRenderer()
        if debug
        else structlog.processors.JSONRenderer(),
        foreign_pre_chain=shared_processors,
    )

    root = logging.getLogger()
    root.setLevel(log_level)
    # Remove any handlers added by previous calls (e.g. uvicorn bootstrap)
    root.handlers.clear()

    stdout_handler = logging.StreamHandler(sys.stdout)
    stdout_handler.setLevel(log_level)
    stdout_handler.setFormatter(formatter)
    root.addHandler(stdout_handler)

    # File handlers — only when log_dir is provided and accessible
    if log_dir:
        try:
            os.makedirs(log_dir, exist_ok=True)
            # File handlers use plain JSON regardless of debug flag
            json_formatter = structlog.stdlib.ProcessorFormatter(
                processor=structlog.processors.JSONRenderer(),
                foreign_pre_chain=shared_processors,
            )
            for handler in [
                _make_rotating_handler(os.path.join(log_dir, "info.log"), logging.INFO, logging.WARNING),
                _make_rotating_handler(os.path.join(log_dir, "warning.log"), logging.WARNING, logging.ERROR),
                _make_rotating_handler(os.path.join(log_dir, "error.log"), logging.ERROR, logging.CRITICAL),
                _make_rotating_handler(os.path.join(log_dir, "critical.log"), logging.CRITICAL),
            ]:
                handler.setFormatter(json_formatter)
                root.addHandler(handler)
        except OSError as exc:
            logging.getLogger(__name__).warning(
                "log_dir_unavailable", extra={"path": log_dir, "error": str(exc)}
            )

    # Quieten noisy third-party loggers
    logging.getLogger("uvicorn.access").setLevel(logging.WARNING)
    logging.getLogger("sqlalchemy.engine").setLevel(logging.WARNING if not debug else logging.DEBUG)


def get_logger(name: str | None = None) -> structlog.stdlib.BoundLogger:
    return structlog.get_logger(name)
