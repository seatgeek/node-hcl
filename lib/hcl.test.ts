/*
 * Copyright SeatGeek
 * Licensed under the terms of the Apache-2.0 license. See LICENSE file in project root for terms.
 */
import { merge, mergeWrite, mergeFiles, mergeFilesWrite } from "./hcl";
import { writeFileSync, readFileSync } from "fs-extra";
import { randomBytes } from "crypto";

describe("merge", () => {
  it("should merge two hcl strings", async () => {
    const a = `variable "a" {
  type        = string
  description = "Variable A"
  default     = "a"
}`;
    const b = `variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}`;

    const expected = `variable "a" {
  type        = string
  description = "Variable A"
  default     = "a"
}

variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}
`;

    const out = await merge(a, b);
    expect(out).toBe(expected);
  });

  it("should merge when empty string", async () => {
    const a = ``;
    const b = `variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}`;

    const expected = `variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}
`;

    const actual = await merge(a, b);
    expect(actual).toBe(expected);
  });
});

describe("mergeWrite", () => {
  it("should merge two hcl strings and write the result to a file", async () => {
    const a = `variable "a" {
  type        = string
  description = "Variable A"
  default     = "a"
}`;
    const b = `variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}`;

    const expected = `variable "a" {
  type        = string
  description = "Variable A"
  default     = "a"
}

variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}
`;
    const outPath = `/tmp/${randomBytes(12).toString("hex")}.hcl`;
    await mergeWrite(a, b, outPath);

    const actual = await readFileSync(outPath, "utf8");
    expect(actual).toBe(expected);
  });
});

describe("mergeFiles", () => {
  it("should merge on nonexistent file", async () => {
    const b = `variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}`;

    const expected = `variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}
`;

    const aPath = `/tmp/${randomBytes(12).toString("hex")}.hcl`;

    const bPath = `/tmp/${randomBytes(12).toString("hex")}.hcl`;
    await writeFileSync(bPath, b, "utf8");

    const actual = await mergeFiles(aPath, bPath);
    expect(actual).toBe(expected);
  });
});

describe("mergeFilesWrite", () => {
  it("should merge two hcl files and write the result to a file", async () => {
    const a = `variable "a" {
  type        = string
  description = "Variable A"
  default     = "a"
}`;
    const b = `variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}`;

    const expected = `variable "a" {
  type        = string
  description = "Variable A"
  default     = "a"
}

variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}
`;

    // write a to file
    const aPath = `/tmp/${randomBytes(12).toString("hex")}.hcl`;
    await writeFileSync(aPath, a, "utf8");

    const bPath = `/tmp/${randomBytes(12).toString("hex")}.hcl`;
    await writeFileSync(bPath, b, "utf8");

    const outPath = `/tmp/${randomBytes(12).toString("hex")}.hcl`;
    await mergeFilesWrite(aPath, bPath, outPath);

    const actual = await readFileSync(outPath, "utf8");
    expect(actual).toBe(expected);
  });
});
