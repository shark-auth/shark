FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates tzdata wget && rm -rf /var/lib/apt/lists/*

# Create a non-privileged user
RUN groupadd -g 1000 shark && \
    useradd -r -u 1000 -g shark shark

WORKDIR /app

# Copy the precompiled binary
ARG BINARY=shark
COPY ${BINARY} sharkauth
RUN chmod +x sharkauth

# Ensure the data directory is owned by the shark user
RUN mkdir -p /app/data && chown -R shark:shark /app/data

USER shark

EXPOSE 8080

ENTRYPOINT ["./sharkauth", "serve"]
