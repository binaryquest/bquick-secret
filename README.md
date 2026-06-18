# bQuick Secret

bQuick Secret is a privacy-first encrypted secret sharing app and a free-to-use product of Binary Quest Limited. Secrets are encrypted in the browser before upload, stored as ciphertext in Postgres, and decrypted only in the recipient browser with a URL fragment key that the backend never receives.

## Structure

```text
apps/web      React + Vite + TypeScript frontend
apps/api      Go API
db/migrations Postgres schema
deploy        Docker Compose and Coolify env example
docs          Product, SRS, and security notes
```

## License

bQuick Secret is released under the MIT License. See [LICENSE](LICENSE).

## Local Development

1. Copy environment examples:

```sh
cp .env.example .env
cp deploy/coolify.env.example deploy/.env
```

2. Start Postgres and services:

```sh
docker compose -f deploy/docker-compose.yml up --build
```

3. Open the web app at `http://localhost:8080`.

For frontend-only development, run the Vite app from `apps/web`. For API-only development, run the Go service from `apps/api`.

## Coolify Deployment

Use `deploy/coolify-compose.yml` for production Coolify deployments. Assign your public domain to the `web` service on container port `80`; keep `api` and `postgres` private inside the Compose network.

## Zero-Knowledge Email Behavior

The backend never receives URL fragments, decrypt keys, plaintext secrets, or passphrases. Because of that, SES email sends a keyless notification link to `/s/{publicId}`. The sender must separately share the full secure link containing `#key=...` or the fragment key through another channel.

## Security Invariant

Plaintext secrets, decrypt keys, passphrases, recipient emails after send, and full URLs with fragments must not be stored or logged.
