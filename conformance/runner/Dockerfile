ARG REGISTRY=docker.io
FROM ${REGISTRY}/python:3.12-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY verify.py utils.py cross_verify.py ./
COPY ../scenarios /app/scenarios/

ENTRYPOINT ["python", "verify.py"]
