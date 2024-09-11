# node-hcl
This WebAssembly wrapper provides a convenient way to use the HCL (HashiCorp Configuration Language) library. It allows you to parse and merge HCL files, making it easier to work with configuration files in your Node applications.

To get started, follow these steps:

1. Install the package by running the following command:
  ```
  npm install @seatgeek/node-hcl
  ```

2. Import the library into your project:
  ```javascript
  const hcl = require('@seatgeek/node-hcl');
  ```

3. Use the library:
  ```javascript
  // Example 1: Merge HCL content
  const a = `
  variable "a" {
    type        = string
    description = "Variable A"
    default     = "a"
  }`
  const b = `
  variable "b" {
    type        = string
    description = "Variable B"
    default     = "b"
  }`
  const mergeResult = hcl.merge(a, b);

  // Example 2: Merge HCL content and write to a file:
  const outputPath = "path/to/merged/variables.tf";
  hcl.mergeWrite(a, b, outputPath);

  // Example 3: Merge two files
  const fp1 = "path/to/variables.tf";
  const fp2 = "path/to/new/variables.tf";
  const mergeFilesResult = hcl.mergeFiles(fp1, fp2);

  // Example 4: Merge two files and write to a file:
  hcl.mergeFilesWrite(fp1, fp2, outputPath);
  ```
