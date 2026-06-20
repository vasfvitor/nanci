import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import pluginVue from 'eslint-plugin-vue'
import prettier from 'eslint-config-prettier'
import { defineConfig } from 'eslint/config'

export default defineConfig([
  {
    ignores: [
      'dist/**',
      'node_modules/**',
      'wailsjs/**',
      'src/vite-env.d.ts',
      '*.tsbuildinfo',
      'package.json.md5',
    ],
  },

  js.configs.recommended,
  ...tseslint.configs.strict,
  ...pluginVue.configs['flat/recommended'],

  {
    files: ['**/*.{js,mjs,cjs,ts,mts,cts,vue}'],
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.es2022,
      },
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
        extraFileExtensions: ['.vue'],
      },
    },
    rules: {
      'no-console': 'warn',
      'no-useless-assignment': 'off',
      eqeqeq: ['error', 'always'],
      curly: ['error', 'all'],
      'no-self-compare': 'error',

      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
        },
      ],
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
      'vue/block-lang': ['error', { script: { lang: 'ts' } }],
      'vue/define-emits-declaration': ['error', 'type-based'],
      'vue/define-props-declaration': ['error', 'type-based'],

      'vue/require-macro-variable-name': [
        'warn',
        {
          defineProps: 'props',
          defineEmits: 'emit',
          defineSlots: 'slots',
          useSlots: 'slots',
          useAttrs: 'attrs',
        },
      ],

      'vue/require-typed-ref': 'warn',
      'vue/no-empty-component-block': 'error',
      'vue/no-ref-object-reactivity-loss': 'error',
      'vue/prefer-true-attribute-shorthand': 'warn',
      'vue/v-for-delimiter-style': ['error', 'in'],
    },
  },

  {
    files: ['src/**/*.{ts,vue}'],
    ignores: [
      'src/platform/wails/client.ts',
      'src/platform/wails/events.ts',
      'src/platform/wails/runtime.ts',
      'src/platform/wails/client.test.ts',
    ],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['**/wailsjs/go/**', '@/wailsjs/go/**'],
              message:
                'Do not import generated Wails bindings directly. Use the wrappers in src/platform/wails/ instead.',
            },
          ],
        },
      ],
    },
  },

  {
    files: ['src/components/**/*.{ts,vue}', 'src/stores/**/*.ts', 'src/composables/**/*.ts'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['**/pages/**', '@/pages/**'],
              message: 'Components, stores, and composables must not import pages.',
            },
          ],
        },
      ],
    },
  },

  {
    files: ['src/pages/**/*.{ts,vue}'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['**/pages/**', '@/pages/**'],
              message:
                'Pages must not import other pages. Share code via components, composables, stores, or routes.',
            },
          ],
        },
      ],
    },
  },

  prettier,
])
