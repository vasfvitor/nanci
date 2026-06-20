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

  // Strict JS/TS Rules
  {
    files: ['**/*.{js,mjs,cjs,ts,mts,cts,vue}'],
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.es2022,
      },
    },
    rules: {
      'no-console': 'warn',
      '@typescript-eslint/no-explicit-any': 'error',
      'no-useless-assignment': 'off',
      'eqeqeq': ['error', 'always'],
      'curly': ['error', 'all'],
      'no-self-compare': 'error',
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }
      ],
    },
  },

  // Vue-specific Strict Rules
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
        'error',
        {
          defineProps: 'props',
          defineEmits: 'emit',
          defineSlots: 'slots',
          useSlots: 'slots',
          useAttrs: 'attrs',
        }
      ],
      'vue/require-typed-ref': 'error',
      'vue/no-empty-component-block': 'error',
      'vue/no-ref-object-reactivity-loss': 'error',
      'vue/prefer-true-attribute-shorthand': 'error',
      'vue/v-for-delimiter-style': ['error', 'in'],
    },
  },

  // Architectural Boundaries: Restrict direct Wails imports outside wrappers
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
              message: 'Do not import generated Wails bindings directly. Use the wrappers in src/platform/wails/ instead.'
            }
          ]
        }
      ]
    }
  },

  // Architectural Boundaries: Prevent importing pages from components/stores/composables
  {
    files: [
      'src/components/**/*.{ts,vue}',
      'src/stores/**/*.ts',
      'src/composables/**/*.ts',
    ],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['**/pages/**', '@/pages/**'],
              message: 'Components, stores, and composables must not import pages.'
            }
          ]
        }
      ]
    }
  },

  // Architectural Boundaries: Prevent pages from importing other pages
  {
    files: ['src/pages/**/*.{ts,vue}'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['**/pages/*', '@/pages/*'],
              message: 'Pages must not import other pages. Share code via components, composables, or stores.'
            }
          ]
        }
      ]
    }
  },

  prettier,
])
