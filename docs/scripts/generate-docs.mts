import * as OpenAPI from 'fumadocs-openapi';
import { rimraf } from 'rimraf';
import type { OpenAPIV3_1 } from 'openapi-types';

export async function generateDocs() {
  await rimraf('./content/docs/spesifikasi-api/(generated)');

  await Promise.all([
    OpenAPI.generateFiles({
      input: ['./inaproc-api.yaml'],
      output: './content/docs/spesifikasi-api',
      per: 'operation',
      name: (output, document) => {
        if (output.type === 'operation') {
          const doc = document as OpenAPIV3_1.Document;
          const operation = doc.paths?.[output.item.path]?.[output.item.method];
          // Create a flat filename using the operation summary and method
          const summary = operation?.summary || output.item.path;
          const method = output.item.method.toUpperCase();
          // Clean the summary to make it filename-safe
          const cleanSummary = summary
            .replace(/[^a-zA-Z0-9\s]/g, '')
            .replace(/\s+/g, '-')
            .toLowerCase();
          return `${cleanSummary}-${method.toLowerCase()}`;
        }
        return 'webhook';
      },
      includeDescription: true,
    }),
  ]);

  // Generate index file with cards automatically
  await generateIndexFile();
}

async function generateIndexFile() {
  // Read the OpenAPI spec to get all operations
  const fs = await import('fs/promises');
  const yaml = await import('js-yaml');
  
  const yamlContent = await fs.readFile('./inaproc-api.yaml', 'utf8');
  const spec = yaml.load(yamlContent) as OpenAPIV3_1.Document;
  
  const operations: Array<{summary: string, method: string, path: string, filename: string}> = [];
  
  if (spec.paths) {
    for (const [path, pathItem] of Object.entries(spec.paths)) {
      if (pathItem) {
        for (const [method, operation] of Object.entries(pathItem)) {
          if (operation && typeof operation === 'object' && 'summary' in operation) {
            const summary = operation.summary || path;
            const cleanSummary = summary
              .replace(/[^a-zA-Z0-9\s]/g, '')
              .replace(/\s+/g, '-')
              .toLowerCase();
            const filename = `${cleanSummary}-${method.toLowerCase()}`;
            
            operations.push({
              summary,
              method: method.toUpperCase(),
              path,
              filename
            });
          }
        }
      }
    }
  }

  // Generate the index.mdx content
  const indexContent = `---
title: Spesifikasi API
description: Dokumentasi lengkap endpoint dan spesifikasi INAPROC API Gateway
---

import { Cards, Card } from 'fumadocs-ui/components/card';

# Spesifikasi API

Selamat datang di dokumentasi spesifikasi API INAPROC. Di sini Anda akan menemukan dokumentasi lengkap tentang semua endpoint API yang tersedia.

## Daftar API Endpoints

<Cards>
${operations.map(op => {
  const methodColor = op.method === 'GET' ? 'text-green-600 bg-green-50' : 
                     op.method === 'POST' ? 'text-blue-600 bg-blue-50' : 
                     op.method === 'PUT' ? 'text-orange-600 bg-orange-50' :
                     op.method === 'DELETE' ? 'text-red-600 bg-red-50' : 'text-gray-600 bg-gray-50';
  
  return `  <Card 
    href="/docs/spesifikasi-api/${op.filename}" 
    title="${op.summary}" 
    description="${op.method} ${op.path}"
    icon={<span className="inline-flex items-center px-2 py-1 rounded text-xs font-medium ${methodColor}">${op.method}</span>}
  />`;
}).join('\n')}
</Cards>

## Informasi Umum

### Base URL
\`\`\`
https://api.inaproc.id/v1
\`\`\`

### Format Response
Semua response API menggunakan format JSON dengan struktur standar:

\`\`\`json
{
  "success": boolean,
  "message": "string",
  "data": object | array,
  "meta": {
    "timestamp": "string",
    "version": "string"
  }
}
\`\`\`

### Authentication
Semua API endpoint memerlukan autentikasi menggunakan API Key:

\`\`\`
Authorization: Bearer YOUR_API_KEY
\`\`\`

### Rate Limiting
- **100 requests per minute** untuk endpoint umum
- **1000 requests per minute** untuk endpoint premium
- **10 requests per second** untuk endpoint real-time
`;

  await fs.writeFile('./content/docs/spesifikasi-api/index.mdx', indexContent);
}

async function main() {
  await generateDocs();
}

await main().catch((e) => {
  console.error('Failed to generate OpenAPI docs', e);
});
