# ============= Stage 1: Build Go server =============
FROM golang:1.22 AS builder

WORKDIR /app
COPY server/ ./server/
COPY go.mod go.sum ./
RUN go mod tidy
WORKDIR /app/server
RUN go build -o /app/server_app

# ============= Stage 2: Final Image =============
FROM python:3.11-slim

WORKDIR /app

# Install Python deps
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Install Gradio
RUN pip install gradio requests

COPY ingest.py ui.py ./ 

# Copy Python ingest + UI
COPY docs/ ./docs/

# Copy Go server binary
COPY --from=builder /app/server_app ./server_app

# Expose ports
EXPOSE 8080 7860

# Run both Go + UI
CMD ["sh", "-c", "./server_app & python ui.py"]
