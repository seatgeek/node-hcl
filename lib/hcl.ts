/*
 * Copyright SeatGeek
 * Licensed under the terms of the Apache-2.0 license. See LICENSE file in project root for terms.
 */
import { wasm } from "./bridge";
import { writeFileSync, readFileSync } from "fs-extra";

async function readFileSafe(path: string): Promise<string> {
  try {
    return readFileSync(path, "utf8");
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") {
      console.warn(
        `file not found at path ${path}, defaulting to empty string`,
      );
      return "";
    } else {
      console.error(`error reading hcl file: ${(error as Error).message}`);
      throw error;
    }
  }
}

export async function merge(a: string, b: string): Promise<string> {
  return await wasm.merge(a, b);
}

export async function mergeWrite(
  a: string,
  b: string,
  outPath: string,
): Promise<void> {
  const out = await merge(a, b);

  try {
    await writeFileSync(outPath, out, "utf8");
  } catch (error) {
    console.error(`error writing hcl file: ${(error as Error).message}`);
  }
}

export async function mergeFiles(
  aPath: string,
  bPath: string,
): Promise<string> {
  const a = await readFileSafe(aPath);
  const b = await readFileSafe(bPath);

  return await merge(a, b);
}

export async function mergeFilesWrite(
  aPath: string,
  bPath: string,
  outPath: string,
): Promise<void> {
  const out = await mergeFiles(aPath, bPath);

  try {
    await writeFileSync(outPath, out, "utf8");
  } catch (error) {
    console.error(`error writing hcl file: ${(error as Error).message}`);
  }
}
