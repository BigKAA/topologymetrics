FROM harbor.kryukov.lan/homelab/python:3.12-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY verify.py utils.py ./
COPY ../scenarios /app/scenarios/

ENTRYPOINT ["python", "verify.py"]
