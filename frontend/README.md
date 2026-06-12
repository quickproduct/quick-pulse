# QuickPulse Frontend

SvelteKit SPA for the QuickPulse control panel. Production deployment is handled by the root `Dockerfile`, which builds this frontend and embeds the generated static files into the Go backend binary.

## Development

Install dependencies and start the Vite dev server:

```sh
npm install
npm run dev
```

The dev server proxies `/api` and `/ws` requests to `http://localhost:8000`, so run the Go backend separately during local development.

## Checks

```sh
npm run check
npm run build
```
