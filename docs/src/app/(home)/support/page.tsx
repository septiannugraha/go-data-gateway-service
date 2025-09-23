import Link from 'next/link';

export default function SupportPage() {
  return (
    <main className="flex flex-1 flex-col justify-center py-16">
      <div className="container mx-auto px-4 max-w-4xl">
        <div className="text-center mb-12">
          <h1 className="mb-6 text-4xl font-bold tracking-tight text-fd-foreground">
            Dukungan Teknis
            <span className="text-fd-primary"> INAPROC API</span>
          </h1>
          <p className="mx-auto max-w-2xl text-lg text-fd-muted-foreground leading-relaxed">
            Tim support kami siap membantu Anda mengintegrasikan dan menggunakan INAPROC API Gateway.
          </p>
        </div>

        {/* Support Channels */}
        <div className="grid md:grid-cols-2 gap-8 mb-12">
          <div className="p-6 border border-fd-border rounded-lg bg-fd-card">
            <div className="mb-4 text-3xl">📧</div>
            <h3 className="mb-3 text-xl font-semibold text-fd-foreground">Email Support</h3>
            <p className="text-fd-muted-foreground mb-4">
              Untuk pertanyaan teknis dan troubleshooting API
            </p>
            <a 
              href="mailto:api-support@inaproc.id" 
              className="text-fd-primary hover:opacity-80 font-medium"
            >
              api-support@inaproc.id
            </a>
          </div>
          
          <div className="p-6 border border-fd-border rounded-lg bg-fd-card">
            <div className="mb-4 text-3xl">📞</div>
            <h3 className="mb-3 text-xl font-semibold text-fd-foreground">Phone Support</h3>
            <p className="text-fd-muted-foreground mb-4">
              Senin - Jumat, 08:00 - 17:00 WIB
            </p>
            <div className="text-fd-primary font-medium">
              +62 21 1234 5678
            </div>
          </div>

          <div className="p-6 border border-fd-border rounded-lg bg-fd-card">
            <div className="mb-4 text-3xl">💬</div>
            <h3 className="mb-3 text-xl font-semibold text-fd-foreground">Live Chat</h3>
            <p className="text-fd-muted-foreground mb-4">
              Chat langsung dengan tim technical support
            </p>
            <button className="px-4 py-2 bg-fd-primary text-fd-primary-foreground rounded-lg hover:opacity-90 transition-all">
              Mulai Chat
            </button>
          </div>

          <div className="p-6 border border-fd-border rounded-lg bg-fd-card">
            <div className="mb-4 text-3xl">📚</div>
            <h3 className="mb-3 text-xl font-semibold text-fd-foreground">Knowledge Base</h3>
            <p className="text-fd-muted-foreground mb-4">
              FAQ dan dokumentasi lengkap
            </p>
            <Link 
              href="/docs/resources/faq" 
              className="text-fd-primary hover:opacity-80 font-medium"
            >
              Lihat FAQ →
            </Link>
          </div>
        </div>

        {/* Common Issues */}
        <div className="mb-12">
          <h2 className="mb-6 text-2xl font-bold text-fd-foreground text-center">
            Masalah yang Sering Terjadi
          </h2>
          <div className="space-y-4">
            <div className="p-4 border border-fd-border rounded-lg bg-fd-card">
              <h4 className="font-semibold text-fd-foreground mb-2">Authentication Error (401)</h4>
              <p className="text-fd-muted-foreground text-sm mb-2">
                Pastikan API key dan token JWT Anda valid dan belum expired.
              </p>
              <Link href="/docs/getting-started/authentication" className="text-fd-primary text-sm hover:opacity-80">
                Lihat panduan authentication →
              </Link>
            </div>
            
            <div className="p-4 border border-fd-border rounded-lg bg-fd-card">
              <h4 className="font-semibold text-fd-foreground mb-2">Rate Limit Exceeded (429)</h4>
              <p className="text-fd-muted-foreground text-sm mb-2">
                Anda telah mencapai batas maksimum request per detik/menit.
              </p>
              <Link href="/docs/getting-started/rate-limiting" className="text-fd-primary text-sm hover:opacity-80">
                Lihat panduan rate limiting →
              </Link>
            </div>
            
            <div className="p-4 border border-fd-border rounded-lg bg-fd-card">
              <h4 className="font-semibold text-fd-foreground mb-2">Server Error (500)</h4>
              <p className="text-fd-muted-foreground text-sm mb-2">
                Terjadi kesalahan di sisi server. Silakan coba lagi atau hubungi support.
              </p>
              <Link href="/docs/getting-started/error-handling" className="text-fd-primary text-sm hover:opacity-80">
                Lihat panduan error handling →
              </Link>
            </div>
          </div>
        </div>

        {/* Service Hours */}
        <div className="text-center p-6 bg-fd-muted rounded-lg">
          <h3 className="mb-4 text-lg font-semibold text-fd-foreground">Jam Operasional Support</h3>
          <div className="grid md:grid-cols-2 gap-4 text-sm text-fd-muted-foreground">
            <div>
              <strong>Email Support:</strong> 24/7
            </div>
            <div>
              <strong>Phone & Chat:</strong> Senin - Jumat, 08:00 - 17:00 WIB
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
