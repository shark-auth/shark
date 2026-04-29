FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates tzdata wget && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the precompiled binary
COPY shark sharkauth
RUN chmod +x sharkauth

RUN mkdir -p /app/data

EXPOSE 8080

ENTRYPOINT ["./sharkauth", "serve"]
