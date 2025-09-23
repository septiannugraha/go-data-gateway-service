import { openapi } from '@/lib/source';
import { RootProvider } from 'fumadocs-ui/provider';
import defaultMdxComponents from 'fumadocs-ui/mdx';
import { APIPage } from 'fumadocs-openapi/ui';
import Link from 'next/link';
import fs from 'fs';
import path from 'path';

// Read the OpenAPI document directly
function getOpenAPIDocument() {
  const yamlPath = path.join(process.cwd(), 'inaproc-api.yaml');
  const yamlContent = fs.readFileSync(yamlPath, 'utf8');
  
  // Parse YAML to JSON (simple approach for demo)
  // In production, you'd use a proper YAML parser like js-yaml
  const yaml = require('js-yaml');
  return yaml.load(yamlContent);
}

export default function APIDocsPage() {
  const document = getOpenAPIDocument();
  
  // Group endpoints by tags
  const endpointsByTag: Record<string, any[]> = {};
  
  if (document.paths) {
    Object.entries(document.paths).forEach(([path, methods]: [string, any]) => {
      Object.entries(methods).forEach(([method, operation]: [string, any]) => {
        if (operation.tags && operation.tags.length > 0) {
          const tag = operation.tags[0];
          if (!endpointsByTag[tag]) {
            endpointsByTag[tag] = [];
          }
          endpointsByTag[tag].push({
            path,
            method: method.toUpperCase(),
            operation,
            operationId: operation.operationId,
            summary: operation.summary,
            description: operation.description,
          });
        }
      });
    });
  }

  const tagDisplayNames: Record<string, { name: string; icon: string; description: string }> = {
    authentication: {
      name: 'Authentication',
      icon: '🔐',
      description: 'Login, token refresh, and authentication endpoints'
    },
    tender: {
      name: 'Tender',
      icon: '📋',
      description: 'Tender and non-tender procurement data'
    },
    rup: {
      name: 'RUP (Rencana Umum Pengadaan)',
      icon: '📅',
      description: 'Annual procurement planning data'
    },
    vendor: {
      name: 'Vendor',
      icon: '👥',
      description: 'Vendor evaluation and performance monitoring'
    }
  };

  return (
    <div className="min-h-screen bg-fd-background">
      {/* Header */}
      <header className="border-b border-fd-border bg-fd-card sticky top-0 z-50">
        <div className="container mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Link href="/" className="flex items-center gap-2 text-fd-foreground hover:text-fd-primary">
                <svg
                  width="28"
                  height="28"
                  xmlns="http://www.w3.org/2000/svg"
                  className="text-fd-primary"
                >
                  <defs>
                    <linearGradient id="logo-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
                      <stop offset="0%" stopColor="currentColor" />
                      <stop offset="100%" stopColor="currentColor" stopOpacity="0.8" />
                    </linearGradient>
                  </defs>
                  <rect x="2" y="2" width="24" height="24" rx="6" fill="url(#logo-gradient)" />
                  <path
                    d="M8 8h8v2H8V8zm0 4h8v2H8v-2zm0 4h6v2H8v-2z"
                    fill="white"
                  />
                </svg>
                <span className="font-bold text-fd-primary">INAPROC</span>
                <span className="text-fd-muted-foreground text-sm font-medium">API Gateway</span>
              </Link>
            </div>
            <nav className="flex items-center gap-6">
              <Link href="/docs" className="text-fd-muted-foreground hover:text-fd-foreground">
                Documentation
              </Link>
              <Link href="/api-docs" className="text-fd-primary font-medium">
                API Reference
              </Link>
              <Link href="/support" className="text-fd-muted-foreground hover:text-fd-foreground">
                Support
              </Link>
            </nav>
          </div>
        </div>
      </header>

      <div className="flex">
        {/* Sidebar TOC */}
        <aside className="w-64 h-screen sticky top-16 border-r border-fd-border bg-fd-card overflow-y-auto">
          <div className="p-6">
            <h3 className="font-semibold text-fd-foreground mb-4">API Reference</h3>
            <nav className="space-y-2">
              {Object.entries(tagDisplayNames).map(([tag, info]) => (
                <a
                  key={tag}
                  href={`#${tag}`}
                  className="block px-3 py-2 text-sm text-fd-muted-foreground hover:text-fd-foreground hover:bg-fd-accent rounded-md"
                >
                  {info.icon} {info.name}
                </a>
              ))}
            </nav>
          </div>
        </aside>

        {/* Main Content */}
        <main className="flex-1">
          <div className="container mx-auto px-6 py-8 max-w-5xl">
            <div className="mb-8 text-center">
              <h1 className="mb-4 text-4xl font-bold text-fd-foreground">
                INAPROC API Reference
              </h1>
              <p className="text-lg text-fd-muted-foreground max-w-2xl mx-auto">
                Complete API documentation for INAPROC Gateway. Test endpoints directly in the browser.
              </p>
            </div>

            {/* API Groups */}
            <div className="space-y-16">
              {Object.entries(endpointsByTag).map(([tag, endpoints]) => {
                const tagInfo = tagDisplayNames[tag];
                if (!tagInfo) return null;
                
                return (
                  <section key={tag} id={tag}>
                    <div className="mb-8">
                      <h2 className="mb-2 text-3xl font-bold text-fd-foreground">
                        {tagInfo.icon} {tagInfo.name}
                      </h2>
                      <p className="text-fd-muted-foreground">{tagInfo.description}</p>
                      <div className="mt-4 border-b border-fd-border"></div>
                    </div>
                    
                    <div className="space-y-8">
                      {endpoints.map((endpoint, index) => (
                        <div key={`${endpoint.path}-${endpoint.method}-${index}`} className="border border-fd-border rounded-lg overflow-hidden bg-fd-card">
                          <div className="p-6">
                            <div className="flex items-center gap-4 mb-4">
                              <span className={`px-3 py-1 rounded-full text-xs font-semibold ${
                                endpoint.method === 'GET' ? 'bg-green-100 text-green-800' :
                                endpoint.method === 'POST' ? 'bg-blue-100 text-blue-800' :
                                endpoint.method === 'PUT' ? 'bg-yellow-100 text-yellow-800' :
                                endpoint.method === 'DELETE' ? 'bg-red-100 text-red-800' :
                                'bg-gray-100 text-gray-800'
                              }`}>
                                {endpoint.method}
                              </span>
                              <code className="text-sm font-mono bg-fd-muted px-2 py-1 rounded">
                                {endpoint.path}
                              </code>
                            </div>
                            
                            <h3 className="text-lg font-semibold text-fd-foreground mb-2">
                              {endpoint.summary || endpoint.operationId}
                            </h3>
                            
                            {endpoint.description && (
                              <p className="text-fd-muted-foreground mb-4">
                                {endpoint.description}
                              </p>
                            )}
                            
                            {/* Try It Out Button */}
                            <button className="px-4 py-2 bg-fd-primary text-fd-primary-foreground rounded-lg hover:opacity-90 transition-all">
                              Try it out
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  </section>
                );
              })}
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}
