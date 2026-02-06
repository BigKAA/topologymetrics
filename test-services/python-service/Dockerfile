# Build context: корень проекта (dephealth/)
# docker build -t ... -f test-services/python-service/Dockerfile .

FROM python:3.12-slim AS builder

WORKDIR /build

# Зависимости (кэшируемый слой)
COPY test-services/python-service/requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# SDK
COPY sdk-python/pyproject.toml sdk-python/pyproject.toml
COPY sdk-python/dephealth/ sdk-python/dephealth/
COPY sdk-python/dephealth_fastapi/ sdk-python/dephealth_fastapi/
RUN pip install --no-cache-dir "./sdk-python[fastapi]"

# ---
FROM python:3.12-slim

# Копируем установленные пакеты из builder
COPY --from=builder /usr/local/lib/python3.12/site-packages/ /usr/local/lib/python3.12/site-packages/
COPY --from=builder /usr/local/bin/uvicorn /usr/local/bin/uvicorn

WORKDIR /app
COPY test-services/python-service/main.py .

EXPOSE 8080

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8080", "--log-level", "info"]
