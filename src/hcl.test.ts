/*
 * Copyright SeatGeek
 * Licensed under the terms of the Apache-2.0 license. See LICENSE file in project root for terms.
 */
import { merge } from "./hcl";

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

  it("should merge nested map keys", async () => {
    const a = `variable "a" {
  foo = "bar"
  map1 = {
    "key1" = {
      numval           = 1
      numval2          = 3
      varval           = local.myvar
      nested_map = {
        "nested_num"    = 100
        "nested_string" = "baz"
      }
    },
  }
  map2 = {
    "key2" = {
      foo = "bar"
    },
  }
}
`;
    const b = `variable "a" {
  bar = "baz"
  map1 = {
    "key1" = {
      numval = 9
    }
  }
  map3 = {
    "key3" = {
      numval = 9
      nested_map = {
        "nested_num"    = 100
        "nested_string" = "baz"
      }
    },
  }
}
`;

    const expected = `variable "a" {
  bar = "baz"
  foo = "bar"
  map1 = {
    "key1" = {
      numval = 9
    }
  }
  map2 = {
    "key2" = {
      foo = "bar"
    },
  }
  map3 = {
    "key3" = {
      numval = 9
      nested_map = {
        "nested_num"    = 100
        "nested_string" = "baz"
      }
    },
  }
}

`;
    const options = { mergeMapKeys: true };
    const actual = await merge(a, b, options);
    expect(actual).toBe(expected);
  });
});
