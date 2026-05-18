from pydantic import field_validator
from pydantic_settings import BaseSettings
from functools import lru_cache
from typing import Any


class Settings(BaseSettings):
    APP_NAME: str = "QuickPulse"
    APP_VERSION: str = "0.1.0"
    DEBUG: bool = False

    DATABASE_URL: str = "postgresql+asyncpg://quickpulse:quickpulse@localhost:5432/quickpulse"
    DATABASE_POOL_SIZE: int = 10
    DATABASE_MAX_OVERFLOW: int = 20
    DATABASE_POOL_TIMEOUT: int = 30
    DATABASE_POOL_PRE_PING: bool = True

    REDIS_URL: str = "redis://localhost:6379/0"

    JWT_SECRET_KEY: str = "change-me-in-production"
    JWT_ALGORITHM: str = "HS256"
    ACCESS_TOKEN_EXPIRE_MINUTES: int = 30
    REFRESH_TOKEN_EXPIRE_DAYS: int = 7

    DOCKER_SOCKET_PATH: str = "/var/run/docker.sock"

    METRICS_COLLECTION_INTERVAL_SECONDS: int = 10
    LOG_TAIL_DEFAULT: int = 100
    LOG_TAIL_MAX: int = 5000

    RATE_LIMIT_PER_MINUTE: int = 60
    RATE_LIMIT_LOGIN_PER_MINUTE: int = 10

    CORS_ORIGINS: Any = ["http://localhost:5173", "http://localhost:3000"]

    @field_validator("CORS_ORIGINS", mode="before")
    @classmethod
    def assemble_cors_origins(cls, v: Any) -> Any:
        if isinstance(v, str) and not v.startswith("["):
            return [i.strip() for i in v.split(",")]
        elif isinstance(v, (list, str)):
            return v
        raise ValueError(v)

    DEFAULT_ADMIN_EMAIL: str = "admin@quickpulse.local"
    DEFAULT_ADMIN_PASSWORD: str = "changeme"

    # Logging
    LOG_DIR: str = "/app/logs"

    # SaaS / billing
    ALLOW_REGISTRATION: bool = False
    DEFAULT_PLAN: str = "free"
    TRIAL_DAYS: int = 14
    STRIPE_SECRET_KEY: str = ""
    STRIPE_WEBHOOK_SECRET: str = ""

    model_config = {"env_file": ".env", "env_file_encoding": "utf-8", "case_sensitive": True}


@lru_cache
def get_settings() -> Settings:
    return Settings()
