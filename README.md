# node-hcl

This WebAssembly wrapper provides a convenient way to use the HCL (HashiCorp Configuration Language) library. It allows you to parse and merge HCL files, making it easier to work with configuration files in your Node applications.

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
