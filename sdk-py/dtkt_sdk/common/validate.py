from protovalidate import Violation

from typing import Generator


def violation_messages(violations: list[Violation]) -> Generator[str, None, None]:
    for violation in violations:
        if violation.proto.field:
            path = ".".join(
                [elem.field_name for elem in violation.proto.field.elements]
            )
            yield f"{path}: {violation.proto.message}"
        else:
            yield violation.proto.message
