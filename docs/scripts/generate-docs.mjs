import { generateFiles } from 'fumadocs-openapi';

void generateFiles({
  // the OpenAPI schema file
  input: ['./inaproc-api.yaml'],
  output: './content/docs/api',
  // we recommend to enable it
  // make sure your endpoint description doesn't break MDX syntax.
  includeDescription: true,
  // generate files per operation (each endpoint gets its own file)
//   per: 'operation',
//   // group operations by tags
//   groupBy: 'tag',
//   // add a comment indicating auto-generation
//   addGeneratedComment: true,
  // custom frontmatter
  frontmatter: (title, description) => ({
    title,
    description,
    full: true, // required for fumadocs UI
  }),
});
