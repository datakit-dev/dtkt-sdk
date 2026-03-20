import { defineConfig } from "tsdown";

export default defineConfig({
  entry: [
    "src/cloud/*.ts",
    "src/proto/**/*.ts",
    "src/protoformsdk/**/*.ts",
    "!**/*.test.ts",
    "!**/*.spec.ts",
  ],
  format: "esm",
  outDir: "dist",
  clean: true,
  dts: true,
  // Bundle all dependencies for zero-dependency distribution
  skipNodeModulesBundle: false,
  // Explicitly allow bundling external dependencies (disables warning/error)
  inlineOnly: false,
});
