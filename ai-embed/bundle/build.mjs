import { build } from 'esbuild';

const result = await build({
  entryPoints: ['entry.mjs'],
  bundle: true,
  format: 'iife',
  target: 'es2020',
  platform: 'browser',
  outfile: '../ai_sdk_bundle.js',
  minify: true,
  define: {
    'process.env.NODE_ENV': '"production"',
    'process.env': '{}',
  },
  logLevel: 'info',
});

// Report size
import { statSync } from 'node:fs';
const stats = statSync('../ai_sdk_bundle.js');
console.log(`Bundle size: ${(stats.size / 1024).toFixed(1)} KB`);
