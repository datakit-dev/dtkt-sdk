import { type CodegenConfig } from "@graphql-codegen/cli";

const config: CodegenConfig = {
  schema: "../graph/schema.graphqls",
  documents: "../graph/**/*.graphql",
  generates: {
    "./src/graphql/generated.ts": {
      plugins: [
        "typescript",
        "typescript-operations",
        "typescript-generic-sdk",
      ],
      config: {
        immutableTypes: true,
        strictScalars: true,
        scalars: {
          Any: "unknown",
          Cursor: "string",
          Int64: "number",
          Map: "Record<string, unknown>",
          Time: "string",
          Bytes: "string",
        },
      },
    },
  },
};

export default config;
