/*
 * Copyright SeatGeek
 * Licensed under the terms of the Apache-2.0 license. See LICENSE file in project root for terms.
 */
import { wasm } from "./bridge";

// MergeOptions is the options for the merge function
export class MergeOptions {
  // mergeMapKeys merges map keys for hcl block attributes that are a map.
  // note: this will not merge the values of the keys.
  public mergeMapKeys: boolean = false;
}

// merge merges two HCL strings
export async function merge(
  a: string,
  b: string,
  options?: MergeOptions,
): Promise<string> {
  return await wasm.merge(a, b, options);
}
