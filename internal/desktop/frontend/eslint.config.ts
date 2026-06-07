import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import pluginVue from 'eslint-plugin-vue'
import prettier from 'eslint-config-prettier'
import { defineConfig } from 'eslint/config'

export default defineConfig([
  {
    ignores: ['dist/**', 'node_modules/**', 'wailsjs/**', 'src/vite-env.d.ts', '*.tsbuildinfo', 'package.json.md5'],
  },

  js.configs.recommended,

  ...tseslint.configs.recommended,

  ...pluginVue.configs['flat/essential'],

  {
    files: ['**/*.{js,mjs,cjs,ts,mts,cts,vue}'],
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.es2022,
      },
    },
  },

  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
      },
    },
    rules: {
      'vue/multi-word-component-names': 'off',
    },
  },

  prettier,
])
