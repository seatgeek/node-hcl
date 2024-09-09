"use strict";

globalThis.require = require;

if (!globalThis.crypto) {
  const crypto = require("crypto");
  globalThis.crypto = {
    getRandomValues(b) {
      return crypto.randomFillSync(b);
    },
  };
}

require("./wasm_exec");
