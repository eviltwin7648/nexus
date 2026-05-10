# Nexus Web

Standalone Next.js UI for the Nexus Go API.

## Environment

Create `web/.env.local` with:

```bash
NEXUS_API_BASE_URL=http://localhost:8080
```

## Run

```bash
npm install
npm run dev
```

The browser UI will be available on the default Next.js port and will proxy API requests to the configured Nexus backend.
