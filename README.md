# node-hcl

This WebAssembly wrapper for https://github.com/hashicorp/hcl provides a convenient way to use the HCL (HashiCorp Configuration Language) library in your Node applications.

## Usage

```
yarn add @seatgeek/node-hcl
```

### Merge HCL content

```javascript
import { merge } from "@seatgeek/node-hcl";

const a = `
variable "a" {
  type        = string
  description = "Variable A"
  default     = "a"
}`;
const b = `
variable "b" {
  type        = string
  description = "Variable B"
  default     = "b"
}`;
const result = merge(a, b);
```
