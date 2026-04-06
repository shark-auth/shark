FROM debian:bookworm-slim

# Install ca-certificates in case the app needs to make TLS requests
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the compiled executable and the configuration file
COPY shark-auth .
COPY sharkauth.yaml .

# Ensure the executable has execute permissions
RUN chmod +x shark-auth

# Expose the application port
EXPOSE 8080

# Run the executable with the provided configuration file
CMD ["./shark-auth", "--config", "sharkauth.yaml"]
