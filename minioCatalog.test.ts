import assert from "node:assert/strict";
import test from "node:test";
import type { BucketItem } from "minio";
import { narrativeInfoFromObject } from "./minioCatalog.ts";

const modified = new Date("2026-07-22T18:30:00.000Z");

test("extracts narrative metadata from a direct Markdown object", () => {
  const object: BucketItem = {
    name: "summary/cids/abc123.md",
    size: 2048,
    etag: "etag",
    lastModified: modified,
  };

  assert.deepEqual(narrativeInfoFromObject(object), {
    cid: "abc123",
    last_modified: modified.toISOString(),
    size_bytes: 2048,
  });
});

test("ignores unrelated and nested objects", () => {
  const keys = [
    "summary/cids/abc123.meta.json",
    "summary/cids/archive/abc123.md",
    "reports/cids/abc123.md",
    "summary/cids/.md",
  ];

  for (const name of keys) {
    const object: BucketItem = { name, size: 0, etag: "etag", lastModified: modified };
    assert.equal(narrativeInfoFromObject(object), null, name);
  }
});

test("ignores MinIO directory-prefix entries", () => {
  const object: BucketItem = { prefix: "summary/cids/", size: 0 };
  assert.equal(narrativeInfoFromObject(object), null);
});
