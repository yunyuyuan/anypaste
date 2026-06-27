import { defineConfig, loadEnv } from 'vite';
import { devtools } from '@tanstack/devtools-vite';

import { tanstackRouter } from '@tanstack/router-plugin/vite';

import viteReact from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

const config = defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const base_url = env.DEBUG_BASE_URL || '/';

  return {
    resolve: { tsconfigPaths: true },
    plugins: [
      devtools(),
      tailwindcss(),
      tanstackRouter({ target: 'react', autoCodeSplitting: true }),
      viteReact(),
    ],
    base: base_url,
    server: {
      host: true,
      strictPort: true,
      allowedHosts: true,
      proxy: {
        [`${base_url}api`]: {
          target: 'http://localhost:8080',
          changeOrigin: true,
        }
      }
    }
  };
});

export default config;
