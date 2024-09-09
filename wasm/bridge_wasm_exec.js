"use strict";

globalThis.require = require;
globalThis.fs = require("fs");
globalThis.TextEncoder = require("util").TextEncoder;
globalThis.TextDecoder = require("util").TextDecoder;

globalThis.performance ??= require("performance");

if (!globalThis.crypto) {
	const crypto = require("crypto");
	globalThis.crypto = {
		getRandomValues(b) {
			return crypto.randomFillSync(b);
		},
	};
}

require("./wasm_exec");
