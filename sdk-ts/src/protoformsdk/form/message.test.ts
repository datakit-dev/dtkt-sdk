import { create } from "@bufbuild/protobuf";
import { reflect } from "@bufbuild/protobuf/reflect";
import { test } from "vitest";

import { ConnectionSchema } from "../../proto/dtkt/core/v1/messages_pb";

import { Message } from "./message";

test("test message", () => {
  const msg = new Message(reflect(ConnectionSchema, create(ConnectionSchema)));
  for (const field of msg.fieldGroup().fields) {
    if (field.elem.getType()) {
      console.log(field.desc.localName, field.elem.getType());
    }
  }
});
