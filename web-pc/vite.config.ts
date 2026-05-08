import { defineConfig, loadEnv } from 'vite';
import vue from '@vitejs/plugin-vue';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const apiTarget = env.VITE_HIGOOS_API_BASE_URL || 'http://127.0.0.1:18082';

  return {
    plugins: [vue()],
    server: {
      host: '0.0.0.0',
      proxy: {
        '/api': {
          target: env.VITE_HIGOOS_API_BASE_URL || apiTarget,
          changeOrigin: true,
        },
      },
    },
    preview: {
      host: '0.0.0.0',
    },
  };
});
