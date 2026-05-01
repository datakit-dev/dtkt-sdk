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
          Bytes: "string",
          Cursor: "string",
          Duration: "string",
          Int64: "number",
          Map: "Record<string, unknown>",
          Time: "string",
          UUID: "string",
        },
      },
    },
  },
};

export default config;
