FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates wget && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the precompiled benchmark binary
COPY bench bench_runner
RUN chmod +x bench_runner

# Default environment variables
ENV TARGET_URL=http://localhost:8080
ENV PROFILE=smoke
ENV ADMIN_KEY=""
ENV DURATION=15s

ENTRYPOINT ["/bin/sh", "-c", "echo ${ADMIN_KEY} > /tmp/admin.key && ./bench_runner -profile ${PROFILE} -target-url ${TARGET_URL} -admin-key-file /tmp/admin.key -duration ${DURATION}"]
