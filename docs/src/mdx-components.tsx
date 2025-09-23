import defaultMdxComponents from 'fumadocs-ui/mdx';
import { APIPage } from 'fumadocs-openapi/ui';
import { openapi } from '@/lib/source';
import type { MDXComponents } from 'mdx/types';

// use this function to get MDX components, you will need it for rendering MDX
export function getMDXComponents(components?: MDXComponents): MDXComponents {
  return {
    ...defaultMdxComponents,
    APIPage: (props) => <APIPage {...openapi.getAPIPageProps(props)} />,
    ...components,
  };
}
