/**
 * Generate TypeScript types from JSON Schema contracts.
 * Run from apps/desktop/:  node scripts/generate-contracts.mjs
 * Check drift:             node scripts/generate-contracts.mjs --check
 *
 * Reads:  ../../shared/api-contracts/index.json (relative to this script)
 * Writes: ../../shared/generated/<output> per entry in index.json
 */

import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import { join, dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { compile } from 'json-schema-to-typescript';

const __filename = fileURLToPath(import.meta.url);
const scriptsDir = dirname(__filename);             // apps/desktop/scripts/
const repoRoot = resolve(scriptsDir, '../../..');   // repo root
const contractsDir = join(repoRoot, 'shared/api-contracts');
const generatedDir = join(repoRoot, 'shared/generated');

const isCheck = process.argv.includes('--check');

const BANNER = [
  '/* AUTO-GENERATED — do not edit by hand.',
  ' * Source: shared/api-contracts/',
  ' * Regenerate: (cd apps/desktop && pnpm generate:contracts)',
  ' */',
].join('\n');

/** Compile one JSON Schema file to a TypeScript string */
async function compileSchema(inputPath) {
  const schema = JSON.parse(readFileSync(inputPath, 'utf8'));
  return compile(schema, schema.title ?? 'Schema', {
    bannerComment: BANNER,
    style: { singleQuote: true, semi: true },
    additionalProperties: false,
    enableConstEnums: false,
    strictIndexSignatures: false,
    unknownAny: false,
    cwd: dirname(inputPath),
  });
}

/** Build the barrel index content from the schema manifest */
function buildBarrel(index) {
  return [
    BANNER,
    '',
    ...index.schemas.map(e => `export * from './${e.output.replace(/\.ts$/, '.js')}';`),
    '',
  ].join('\n');
}

async function buildAll() {
  const index = JSON.parse(readFileSync(join(contractsDir, 'index.json'), 'utf8'));
  const results = {};
  for (const entry of index.schemas) {
    const inputPath = join(contractsDir, entry.input);
    results[entry.output] = await compileSchema(inputPath);
  }
  // Include barrel index in the generated map so both write and --check handle it uniformly
  results['index.ts'] = buildBarrel(index);
  return { index, results };
}

async function main() {
  const { results } = await buildAll();

  if (isCheck) {
    let drifted = false;
    for (const [output, content] of Object.entries(results)) {
      const dest = join(generatedDir, output);
      if (!existsSync(dest)) {
        console.error(`DRIFT: missing  ${output}`);
        drifted = true;
      } else {
        const existing = readFileSync(dest, 'utf8');
        if (existing !== content) {
          console.error(`DRIFT: differs  ${output}`);
          drifted = true;
        }
      }
    }
    if (drifted) {
      console.error('\nFix: (cd apps/desktop && pnpm generate:contracts)');
      process.exit(1);
    }
    console.log(`✓ No contract drift detected (${Object.keys(results).length} files)`);
    return;
  }

  // Write all generated files (schemas + barrel index)
  for (const [output, content] of Object.entries(results)) {
    const dest = join(generatedDir, output);
    mkdirSync(dirname(dest), { recursive: true });
    writeFileSync(dest, content, 'utf8');
    console.log(`generated  ${output}`);
  }
  console.log(`✓ Generated ${Object.keys(results).length} files into shared/generated/`);
}

main().catch(err => {
  console.error(err.message ?? err);
  process.exit(1);
});
