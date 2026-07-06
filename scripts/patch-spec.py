#!/usr/bin/env python3
"""Patch the Semaphore OpenAPI spec with endpoints the server implements
but does not document.

The upstream spec omits GET /project/{project_id}/schedules even though the
server has implemented it since before v2.16.51 (api/router.go:
projects.GetProjectSchedules). semctl relies on it for schedule
reconciliation in `semctl apply` and for `semctl export`.

Run automatically by scripts/generate-api.sh after fetching a fresh spec.
Idempotent: running it on an already-patched spec is a no-op.
"""

import sys

SCHEDULES_PATH_ANCHOR = "  /project/{project_id}/schedules:\n"

SCHEDULES_GET = """\
    get:
      tags:
        - schedule
      summary: get schedules of the project
      responses:
        200:
          description: list of schedules (includes tpl_name)
          schema:
            type: array
            items:
              $ref: "#/definitions/Schedule"
"""

SCHEDULE_DEF_ANCHOR = "  Schedule:\n    type: object\n    properties:\n"

# The list endpoint returns ScheduleWithTpl (Schedule + template name).
TPL_NAME_PROP = """\
      tpl_name:
        type: string
"""


def patch(text: str) -> str:
    # 1. Add GET to /project/{project_id}/schedules
    idx = text.find(SCHEDULES_PATH_ANCHOR)
    if idx == -1:
        sys.exit("patch-spec: anchor not found: /project/{project_id}/schedules path")
    block_start = idx + len(SCHEDULES_PATH_ANCHOR)
    post_idx = text.find("    post:", block_start)
    if post_idx == -1:
        sys.exit("patch-spec: post: not found under /project/{project_id}/schedules")
    get_idx = text.find("    get:", block_start, post_idx)
    if get_idx == -1:
        # Insert get: right before post: (after the shared parameters block)
        text = text[:post_idx] + SCHEDULES_GET + text[post_idx:]

    # 2. Add tpl_name to the Schedule definition
    idx = text.find(SCHEDULE_DEF_ANCHOR)
    if idx == -1:
        sys.exit("patch-spec: anchor not found: Schedule definition")
    props_start = idx + len(SCHEDULE_DEF_ANCHOR)
    # End of the Schedule block: next top-level definition (two-space indent)
    block_end = text.find("\n  ", props_start)
    while block_end != -1 and text[block_end + 3] == " ":
        block_end = text.find("\n  ", block_end + 1)
    schedule_block = text[props_start:block_end]
    if "tpl_name:" not in schedule_block:
        text = text[:props_start] + TPL_NAME_PROP + text[props_start:]

    return text


def main() -> None:
    if len(sys.argv) != 2:
        sys.exit("usage: patch-spec.py <api-docs.yml>")
    path = sys.argv[1]
    with open(path, encoding="utf-8") as f:
        original = f.read()
    patched = patch(original)
    if patched != original:
        with open(path, "w", encoding="utf-8") as f:
            f.write(patched)
        print(f"patch-spec: patched {path}")
    else:
        print(f"patch-spec: {path} already patched")


if __name__ == "__main__":
    main()
