import {
  type DescField,
  fromJson,
  merge,
  type Registry,
  ScalarType,
} from "@bufbuild/protobuf";
import {
  parsePath,
  type ReflectMessage,
} from "@bufbuild/protobuf/reflect";

/**
 * applyProtopathParams reads a URLSearchParams object and applies each
 * entry as a protopath set against the supplied message. The key of
 * each param is parsed as a buf protobuf path (e.g. `deployment`,
 * `meta.config.timeout`); the value is interpreted relative to the
 * destination leaf field's kind (string fields stay as strings;
 * numeric/bool fields are decoded; messages and enums set the matching
 * JSON form).
 *
 * Unknown paths are skipped silently - URL params on a `new` page often
 * include unrelated session keys (e.g. `?next=...`) that shouldn't
 * abort the prefill.
 *
 * Oneof selection is automatic: setting any field inside a oneof flips
 * the oneof case to that field (per buf's `parsePath` semantics, the
 * URL key is the branch field name directly, e.g. `?deployment=...`,
 * not `?type.deployment=...`).
 *
 * Lives next to the protoform form classes since it operates on the
 * same ReflectMessage that `Message` / `ScalarField` / etc. wrap.
 */
export function applyProtopathParams(
  message: ReflectMessage,
  params: URLSearchParams,
  options?: { registry?: Registry },
): void {
  for (const [key, value] of params.entries()) {
    try {
      applyOne(message, key, value, options?.registry);
    } catch (err) {
      console.debug(`applyProtopathParams: skipping ${key}=${value}:`, err);
    }
  }
}

function applyOne(
  message: ReflectMessage,
  key: string,
  value: string,
  registry?: Registry,
): void {
  const path = parsePath(message.desc, key, { registry });
  if (path.length === 0) return;

  // Field paths and map subscripts are supported. List subscripts and
  // extensions/oneofs are not (the URL convention is keyed by field).
  for (const p of path) {
    if (p.kind === "list_sub") {
      throw new Error(`protopath ${key}: list subscripts are not supported`);
    }
    if (p.kind === "extension" || p.kind === "oneof") {
      throw new Error(`protopath ${key}: ${p.kind} path elements are not supported`);
    }
  }

  // Build a nested JSON object that mirrors the path, with the encoded
  // leaf value at the bottom. The leaf must be a regular field (not a
  // subscript) so we can produce a sensible JSON shape.
  const leaf = path[path.length - 1];
  if (leaf?.kind !== "field") {
    throw new Error(`protopath ${key}: path must end at a field`);
  }
  const encoded = encodeLeaf(leaf, value);

  let json: unknown = { [jsonKey(leaf)]: encoded };
  for (let i = path.length - 2; i >= 0; i--) {
    const step = path[i];
    if (step === undefined) continue;
    if (step.kind === "field") {
      json = { [jsonKey(step)]: json };
    } else if (step.kind === "map_sub") {
      // map<string, M> JSON form is { "<key>": { ... } }. Map keys are
      // always rendered as JSON strings per protobuf-JSON.
      json = { [String(step.key)]: json };
    }
  }

  // Parse the JSON into a fresh message of the same type, then merge
  // it into the target so existing fields are preserved.
  const patch = fromJson(message.desc, json as Parameters<typeof fromJson>[1], { registry });
  merge(message.desc, message.message as Parameters<typeof merge>[1], patch);
}

function jsonKey(field: DescField): string {
  return field.jsonName !== "" ? field.jsonName : field.name;
}

function encodeLeaf(field: DescField, raw: string): unknown {
  switch (field.fieldKind) {
    case "scalar":
      return encodeScalar(field.scalar, raw);
    case "enum":
      return raw;
    case "message": {
      try {
        return JSON.parse(raw);
      } catch {
        throw new Error(`message field ${field.name}: value must be JSON`);
      }
    }
    case "list":
      throw new Error(`field ${field.name}: list leaves are not supported`);
    case "map":
      // Map leaves are only reachable via subscript; without a subscript
      // the URL would have to encode the entire map as JSON, which is
      // beyond the prefill contract.
      throw new Error(`field ${field.name}: write the map via subscript syntax, e.g. ${field.name}[key].subfield=value`);
  }
}

function encodeScalar(scalar: ScalarType, raw: string): unknown {
  switch (scalar) {
    case ScalarType.STRING:
      return raw;
    case ScalarType.BOOL:
      return raw === "true";
    case ScalarType.INT32:
    case ScalarType.SINT32:
    case ScalarType.SFIXED32:
    case ScalarType.UINT32:
    case ScalarType.FIXED32:
      return Number.parseInt(raw, 10);
    case ScalarType.INT64:
    case ScalarType.SINT64:
    case ScalarType.SFIXED64:
    case ScalarType.UINT64:
    case ScalarType.FIXED64:
      return raw; // JSON encodes 64-bit ints as strings
    case ScalarType.FLOAT:
    case ScalarType.DOUBLE:
      return Number.parseFloat(raw);
    case ScalarType.BYTES:
      return raw; // base64-encoded string per protobuf JSON spec
    default:
      return raw;
  }
}
