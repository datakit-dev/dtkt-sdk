import { ScalarType } from "@bufbuild/protobuf";
import { expect, test } from "vitest";

import { parseScalarValue } from "./scalar";

// Regression: Boolean("false") === true, so the confirm widget used to send
// `true` for both Approve and Decline. A bool must parse from its literal
// string, not from JS truthiness.
test("parseScalarValue BOOL parses the literal, not truthiness", () => {
  expect(parseScalarValue(ScalarType.BOOL, "true")).toBe(true);
  expect(parseScalarValue(ScalarType.BOOL, "false")).toBe(false);
  expect(parseScalarValue(ScalarType.BOOL, "")).toBe(false);
});

test("parseScalarValue passes strings through and parses numbers", () => {
  expect(parseScalarValue(ScalarType.STRING, "hello")).toBe("hello");
  expect(parseScalarValue(ScalarType.INT32, "42")).toBe(42);
  expect(parseScalarValue(ScalarType.DOUBLE, "1.5")).toBe(1.5);
});
