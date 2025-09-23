# OpenAPI Documentation Setup

This project includes automated OpenAPI documentation generation using Fumadocs OpenAPI integration.

## Files Structure

- `inaproc-api.yaml` - OpenAPI specification file
- `scripts/generate-docs.mjs` - Script to generate API documentation
- `content/docs/api/` - Generated API documentation files
- `src/app/api/proxy/route.ts` - API proxy for testing endpoints

## Regenerating API Documentation

When the OpenAPI specification (`inaproc-api.yaml`) is updated, run:

```bash
node ./scripts/generate-docs.mjs
```

This will regenerate all the API documentation files in `content/docs/api/`.

## Features

- ✅ Interactive API playground for each endpoint
- ✅ Code samples in multiple languages
- ✅ Request/response examples
- ✅ Parameter documentation
- ✅ Schema validation
- ✅ CORS proxy for testing live endpoints

## API Sections

- **Authentication** - Login and token management
- **RUP** - Rencana Umum Pengadaan endpoints
- **Tender** - Tender and non-tender procurement
- **Vendor** - Vendor management and evaluation

## Testing Endpoints

The interactive playground allows you to test endpoints directly from the documentation. The proxy server handles CORS issues when testing against the actual API endpoints.

## Configuration

The OpenAPI integration is configured in:
- `src/lib/source.ts` - OpenAPI server setup
- `src/mdx-components.tsx` - APIPage component integration
- `src/app/global.css` - OpenAPI UI styles
