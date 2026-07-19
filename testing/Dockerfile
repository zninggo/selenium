FROM golang:1.22-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
    xvfb default-jre-headless unzip chromium firefox-esr bzip2 \
    && rm -rf /var/lib/apt/lists/*
VOLUME /code
