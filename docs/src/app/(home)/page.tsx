import Link from 'next/link';
import Image from 'next/image';

export default function HomePage() {
  return (
    <main className="flex flex-1 flex-col justify-center py-16">
      {/* Hero Section */}
      <div className="container mx-auto px-4">
        <div className="grid lg:grid-cols-2 gap-12 items-center mb-16">
          {/* Left side - Text and Buttons */}
          <div className="text-center lg:text-left">
            <h1 className="mb-6 text-5xl font-bold tracking-tight text-fd-foreground">
              INAPROC
              <span className="text-fd-primary"> API Gateway</span>
            </h1>
            <p className="mb-8 text-xl text-fd-muted-foreground leading-relaxed">
              Platform integrasi terpadu untuk mengakses berbagai sistem pengadaan pemerintah Indonesia
              melalui satu titik akses yang aman dan terstandarisasi.
            </p>
            
            {/* CTA Buttons */}
            <div className="flex flex-col sm:flex-row gap-4 justify-center lg:justify-start">
              <Link
                href="/docs"
                className="px-8 py-3 bg-fd-primary text-fd-primary-foreground font-semibold rounded-lg hover:opacity-90 transition-all"
              >
                Mulai Sekarang
              </Link>
              <Link
                href="/docs/getting-started/quick-start"
                className="px-8 py-3 border border-fd-border text-fd-foreground font-semibold rounded-lg hover:bg-fd-accent transition-colors"
              >
                Quick Start Guide
              </Link>
            </div>
          </div>

          {/* Right side - Banner Image - Hidden on mobile/tablet */}
          <div className="hidden lg:flex justify-center">
            <Image 
              src="/img/apigw.png" 
              alt="API Gateway - Collaboration between parties using the API gateway" 
              width={600} 
              height={400} 
              className="w-full h-auto max-w-lg"
              priority
            />
          </div>
        </div>

        {/* Features Grid */}
        <div className="text-center">
          <div className="grid md:grid-cols-3 gap-8 max-w-6xl mx-auto mb-16">
          <div className="p-6 border border-fd-border rounded-lg bg-fd-card">
            <div className="mb-4 text-4xl">🚀</div>
            <h3 className="mb-3 text-xl font-semibold text-fd-foreground">Performa Tinggi</h3>
            <p className="text-fd-muted-foreground">
              Response time &lt; 100ms, 99.9% uptime SLA, dan auto-scaling infrastructure.
            </p>
          </div>
          
          <div className="p-6 border border-fd-border rounded-lg bg-fd-card">
            <div className="mb-4 text-4xl">🔒</div>
            <h3 className="mb-3 text-xl font-semibold text-fd-foreground">Keamanan Berlapis</h3>
            <p className="text-fd-muted-foreground">
              OAuth 2.0 authentication, rate limiting per API key, dan end-to-end encryption.
            </p>
          </div>
          
          <div className="p-6 border border-fd-border rounded-lg bg-fd-card">
            <div className="mb-4 text-4xl">📊</div>
            <h3 className="mb-3 text-xl font-semibold text-fd-foreground">Monitoring Real-time</h3>
            <p className="text-fd-muted-foreground">
              Dashboard analytics, usage metrics, dan error tracking komprehensif.
            </p>
          </div>
        </div>

        {/* Services Section */}
        <div className="mb-16">
          <h2 className="mb-8 text-3xl font-bold text-fd-foreground">Layanan yang Tersedia</h2>
          <div className="grid md:grid-cols-3 gap-6 max-w-4xl mx-auto">
            <Link href="/docs/tender" className="group p-6 border border-fd-border rounded-lg bg-fd-card hover:bg-fd-accent transition-colors">
              <div className="mb-3 text-3xl">📋</div>
              <h3 className="mb-2 text-lg font-semibold text-fd-foreground group-hover:text-fd-primary">Tender API</h3>
              <p className="text-fd-muted-foreground text-sm">Data tender & non-tender pengadaan pemerintah</p>
            </Link>
            
            <Link href="/docs/vendor" className="group p-6 border border-fd-border rounded-lg bg-fd-card hover:bg-fd-accent transition-colors">
              <div className="mb-3 text-3xl">👥</div>
              <h3 className="mb-2 text-lg font-semibold text-fd-foreground group-hover:text-fd-primary">Vendor API</h3>
              <p className="text-fd-muted-foreground text-sm">Evaluasi dan monitoring kinerja penyedia</p>
            </Link>
            
            <Link href="/docs/rup" className="group p-6 border border-fd-border rounded-lg bg-fd-card hover:bg-fd-accent transition-colors">
              <div className="mb-3 text-3xl">📅</div>
              <h3 className="mb-2 text-lg font-semibold text-fd-foreground group-hover:text-fd-primary">RUP API</h3>
              <p className="text-fd-muted-foreground text-sm">Rencana Umum Pengadaan tahunan</p>
            </Link>
          </div>
        </div>

        {/* Stats Section */}
        <div className="mb-16 p-8 bg-fd-muted rounded-lg">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
            <div className="text-center">
              <div className="text-3xl font-bold text-fd-primary mb-2">1000+</div>
              <div className="text-fd-muted-foreground">Instansi Terdaftar</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-fd-primary mb-2">50M+</div>
              <div className="text-fd-muted-foreground">API Calls/Month</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-fd-primary mb-2">99.9%</div>
              <div className="text-fd-muted-foreground">Uptime SLA</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-fd-primary mb-2">&lt;100ms</div>
              <div className="text-fd-muted-foreground">Avg Response</div>
            </div>
          </div>
        </div>

        {/* Quick Links */}
        <div>
          <h3 className="mb-6 text-xl font-semibold text-fd-foreground">Membutuhkan Bantuan?</h3>
          <div className="flex flex-wrap justify-center gap-4">
            <Link href="/docs/getting-started" className="text-fd-primary hover:opacity-80 font-medium">
              📚 Getting Started
            </Link>
            <span className="text-fd-muted-foreground">•</span>
            <Link href="/docs/getting-started/authentication" className="text-fd-primary hover:opacity-80 font-medium">
              🔐 Authentication
            </Link>
            <span className="text-fd-muted-foreground">•</span>
            <Link href="/docs/getting-started/error-handling" className="text-fd-primary hover:opacity-80 font-medium">
              🆘 Troubleshooting
            </Link>
            <span className="text-fd-muted-foreground">•</span>
            <a href="mailto:api-support@inaproc.id" className="text-fd-primary hover:opacity-80 font-medium">
              📧 Support
            </a>
          </div>
        </div>
        </div>
      </div>
    </main>
  );
}
