import { defineConfig } from 'vite';
import { compileScript, parse } from '@vue/compiler-sfc';
import { transform as transformWithEsbuild } from 'esbuild';

export default defineConfig({
  define: {
    __VUE_OPTIONS_API__: 'true',
    __VUE_PROD_DEVTOOLS__: 'false',
    __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: 'false',
  },
  plugins: [
    {
      name: 'local-vue-sfc-compiler',
      enforce: 'pre',
      async transform(code, id) {
        if (!id.endsWith('.vue')) return null;

        const { descriptor, errors } = parse(code, { filename: id });
        if (errors.length) {
          throw errors[0];
        }

        const script = compileScript(descriptor, {
          id,
          genDefaultAs: '__sfc__',
          inlineTemplate: true,
        });

        const compiled = `${script.content}
export default __sfc__;`;
        const js = await transformWithEsbuild(compiled, { loader: 'ts', sourcemap: false });
        return { code: js.code, map: null };
      },
    },
  ],
  server: {
    port: 5198,
  },
});
