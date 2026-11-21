FROM python:3.13-slim
WORKDIR /app
COPY hermes-worker/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY hermes-worker/hermes ./hermes
RUN useradd --system --uid 1001 hermes && mkdir -p /app/state && chown hermes:hermes /app/state && chmod 700 /app/state
ENV CREATA_WORKER_STATE_DIR=/app/state
USER hermes
CMD ["python", "-m", "hermes.main"]
