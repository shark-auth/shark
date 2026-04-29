# Installation

SharkAuth is distributed as a single, zero-dependency binary. It embeds the database engine (SQLite) and the admin dashboard, so you don't need to manage external services or separate frontend deployments.

## Direct Binary Download (Recommended)

Download the pre-compiled binary for your platform from the [GitHub Releases](https://github.com/shark-auth/shark/releases) page.

### Linux / macOS

```bash
# 1. Download for your architecture (example: linux amd64)
curl -fsSL https://github.com/shark-auth/shark/releases/latest/download/shark_linux_x86_64.tar.gz -o shark.tar.gz

# 2. Extract
tar -xzf shark.tar.gz

# 3. Move to your path
sudo mv shark /usr/local/bin/

# 4. Verify
shark version
```

### Windows

1. Download the `shark_windows_x86_64.zip` from the releases page.
2. Extract the `shark.exe` to a folder of your choice.
3. Add that folder to your system `PATH`.
4. Open a terminal and run `shark version`.

## Docker

We provide a lightweight Docker image based on Alpine Linux.

```bash
docker run -p 8080:8080 -v $(pwd)/data:/data ghcr.io/sharkauth/sharkauth:latest
```

- `-p 8080:8080`: Maps the main API and dashboard port.
- `-v $(pwd)/data:/data`: Persists the SQLite database and install ID outside the container.

## Build from Source

If you have Go 1.22+ and Node.js 20+ installed, you can build SharkAuth yourself.

```bash
# 1. Clone
git clone https://github.com/shark-auth/shark
cd sharkauth

# 2. Build everything (frontend + backend)
make build

# 3. Run
./shark serve
```

## First Boot

On the very first run, SharkAuth will:
1. Create the `data/` directory.
2. Initialize the SQLite database and run 29 migrations.
3. Generate a 32-byte `server.secret`.
4. Generate a default admin API key.

**Watch the console output!** SharkAuth prints your admin key and a one-time bootstrap URL once.

```text
  ADMIN API KEY (shown once â€” save it now)

    sk_live_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

  Open http://localhost:8080/admin/?bootstrap=...  (expires in 10 minutes)
```

## Configuration

SharkAuth works out of the box with zero config, but you can customize it via `sharkauth.yaml` or environment variables.

See [Configuration Guide](./guides/configuration.md) for details.
