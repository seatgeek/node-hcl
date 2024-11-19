/*
 * Copyright SeatGeek
 * Licensed under the terms of the Apache-2.0 license. See LICENSE file in project root for terms.
 */
package hcl_test

import (
	"testing"

	"github.com/seatgeek/node-hcl/hcl"
	"github.com/stretchr/testify/assert"
)

func TestMerge(t *testing.T) {
	t.Parallel()

	type input struct {
		a string
		b string
	}

	tests := []struct {
		name    string
		input   input
		want    string
		wantErr error
	}{
		{
			name: "merge simple",
			input: input{
				a: `variable "a" {
  type        = string
  description = "Variable A"
  default     = "a"
}`,
				b: `variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}`,
			},
			want: `variable "a" {
  type        = string
  description = "Variable A"
  default     = "a"
}

variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}
`,
			wantErr: nil,
		},
		{
			name: "merge duplicate",
			input: input{
				a: `variable "a" {
  type        = string
  description = "Variable A"
  override    = false
  a					  = "a"
}`,
				b: `variable "a" {
  type        = string
  description = "Variable A"
  override    = true
  b           = "b"
}`,
			},
			want: `variable "a" {
  a           = "a"
  b           = "b"
  description = "Variable A"
  override    = true
  type        = string
}

`,
			wantErr: nil,
		},
		{
			name: "merge nested",
			input: input{
				a: `monitor "a" {
  description = "Monitor A"

  threshold {
    critical = 90
    warning = 80
  }
}`,
				b: `monitor "a" {
  description = "Monitor A"

  threshold {
    critical = 100
    recovery = 10
  }
}`,
			},
			want: `monitor "a" {
  description = "Monitor A"

  threshold {
    critical = 100
    recovery = 10
    warning  = 80
  }
}

`,
			wantErr: nil,
		},
		{
			name: "merge nested duplicate",
			input: input{
				a: `module "b" {

			c = {
				"foo" = {
					value = 1
				}
			}
		}
					`,
				b: `module "b" {

			c = {
				"bar" = {
					value = 2
				}
			}
		}
					`,
			},
			want: `module "b" {
  c = {
    "bar" = {
      value = 2
    }
    "foo" = {
      value = 1
    }
  }
}

`,
		},
		{
			name: "merge complicated nested duplicate",
			input: input{
				a: `module "b" {
	test_map = {
		string_key    = "string_value"
		"int_key"     = 42
		var_key       = var.value
		float_key     = 3.14
		bool_key      = true
		list_key      = ["item1", "item2", 3, true]
		nested_key    = {
			nested_string = "nested_value"
			deep_nested   = {
				deep_key = "deep_value"
			}
		}
		empty_map_key = {}
	}
}`,
				b: `module "b" {
	test_map = {
		string_key_2  = "string_value"
		int_key_2     = 43
		float_key     = 3.14
		bool_key      = true
		null_key      = null
		list_key      = ["item1", "item2", 3, true, "new"]
		nested_key    = {
			nested_string = "nested_value"
			nested_int    = 100
			deep_nested   = {
				deep_key = "deep_value"
				new      = "new"
			}
		}
		empty_map_key = {}
		"quoted.key"  = "quoted_value"
		mixed_key     = {
			inner_string = "inner_value"
			inner_list   = [1, 2, 3]
			inner_map    = { key = "value" }
		}
	}
}`,
			},
			want: `module "b" {
  test_map = {
    bool_key      = true
    empty_map_key = {}
    float_key     = 3.14
    "int_key"     = 42
    int_key_2     = 43
    list_key      = ["item1", "item2", 3, true, "new"]
    mixed_key = {
      inner_string = "inner_value"
      inner_list   = [1, 2, 3]
      inner_map    = { key = "value" }
    }
    nested_key = {
      nested_string = "nested_value"
      nested_int    = 100
      deep_nested = {
        deep_key = "deep_value"
        new      = "new"
      }
    }
    null_key     = null
    "quoted.key" = "quoted_value"
    string_key   = "string_value"
    string_key_2 = "string_value"
    var_key      = var.value
  }
}

`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := hcl.Merge(tc.input.a, tc.input.b)
			assert.Equal(t, tc.want, got)

			if tc.wantErr != nil {
				assert.ErrorIs(t, tc.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
