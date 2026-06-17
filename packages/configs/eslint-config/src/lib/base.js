import eslint from "@eslint/js";
import tseslint from "typescript-eslint";
import { defineConfig } from "eslint/config";

export const baseConfig = defineConfig(
  { ignores: ['dist/', 'node_modules/', '**/*.d.ts', '*.tsbuildinfo']},
  eslint.configs.recommended,
  tseslint.configs.recommended,
);
