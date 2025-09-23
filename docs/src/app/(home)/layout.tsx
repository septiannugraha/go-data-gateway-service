import type { ReactNode } from 'react';
import { HomeLayout } from 'fumadocs-ui/layouts/home';
import { baseOptions } from '@/app/layout.config';
import {
  NavbarMenu,
  NavbarMenuContent,
  NavbarMenuLink,
  NavbarMenuTrigger,
} from 'fumadocs-ui/layouts/home/navbar';
import Link from 'fumadocs-core/link';
import Image from 'next/image';
import ApiGatewayPreview from '../../../public/img/apigw_code.png';
import { Book, ComponentIcon, Pencil, PlusIcon, Server } from 'lucide-react';

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <HomeLayout
      {...baseOptions}
      links={[
        {
          type: 'custom',
          on: 'nav',
          children: (
            <NavbarMenu>
              <NavbarMenuTrigger>
                <Link href="/docs">Dokumentasi</Link>
              </NavbarMenuTrigger>
              <NavbarMenuContent className="text-[15px]">
                <NavbarMenuLink href="/docs" className="md:row-span-2">
                  <div className="-mx-3 -mt-3 h-32 overflow-hidden">
                    <Image
                      src={ApiGatewayPreview}
                      alt="INAPROC API Gateway Preview"
                      className="rounded-t-lg object-cover w-full h-full"
                      style={{
                        maskImage:
                          'linear-gradient(to bottom,white 60%,transparent)',
                      }}
                    />
                  </div>
                  <p className="font-medium">Getting Started</p>
                  <p className="text-fd-muted-foreground text-sm">
                    Pelajari cara menggunakan INAPROC API Gateway untuk dokumentasi API Anda.
                  </p>
                </NavbarMenuLink>

                <NavbarMenuLink
                  href="/docs/spesifikasi-api"
                  className="lg:col-start-2"
                >
                  <Server className="bg-fd-primary text-fd-primary-foreground p-1 mb-2 rounded-md" />
                  <p className="font-medium">Spesifikasi API</p>
                  <p className="text-fd-muted-foreground text-sm">
                    Dokumentasi lengkap endpoint dan spesifikasi API.
                  </p>
                </NavbarMenuLink>

                <NavbarMenuLink
                  href="/docs/docs/tutorial"
                  className="lg:col-start-2"
                >
                  <Pencil className="bg-fd-primary text-fd-primary-foreground p-1 mb-2 rounded-md" />
                  <p className="font-medium">Tutorial</p>
                  <p className="text-fd-muted-foreground text-sm">
                    Panduan langkah demi langkah untuk implementasi API.
                  </p>
                </NavbarMenuLink>

                <NavbarMenuLink
                  href="/docs/docs/guides"
                  className="lg:col-start-3 lg:row-start-1"
                >
                  <ComponentIcon className="bg-fd-primary text-fd-primary-foreground p-1 mb-2 rounded-md" />
                  <p className="font-medium">Guides</p>
                  <p className="text-fd-muted-foreground text-sm">
                    Panduan best practices dan tips penggunaan.
                  </p>
                </NavbarMenuLink>

                <NavbarMenuLink
                  href="/docs/docs/examples"
                  className="lg:col-start-3 lg:row-start-2"
                >
                  <PlusIcon className="bg-fd-primary text-fd-primary-foreground p-1 mb-2 rounded-md" />
                  <p className="font-medium">Examples</p>
                  <p className="text-fd-muted-foreground text-sm">
                    Contoh implementasi dan use cases umum.
                  </p>
                </NavbarMenuLink>
              </NavbarMenuContent>
            </NavbarMenu>
          ),
        },
        {
          text: 'Spesifikasi API',
          url: '/docs/spesifikasi-api',
        },
        {
          text: 'Support',
          url: 'https://bantuan.inaproc.id/hc/id-id',
        },
      ]}
    >
      {children}
    </HomeLayout>
  );
}
