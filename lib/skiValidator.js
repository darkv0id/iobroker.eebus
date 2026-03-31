'use strict';

/**
 * SKI (Subject Key Identifier) validation utilities
 *
 * SKI format follows SHIP 12.2 (RFC 3280 4.2.1.2):
 * - SHA-1 hash of the public key
 * - 20 bytes = 40 hexadecimal characters
 * - Must be lowercase
 */

/**
 * Validate SKI format
 * @param {string} ski - Device SKI to validate (must be lowercase)
 * @throws {Error} If SKI format is invalid
 */
function validateSKI(ski) {
	if (!ski || typeof ski !== 'string') {
		throw new Error('SKI must be a non-empty string');
	}

	// Validate format: exactly 40 lowercase hexadecimal characters
	if (!/^[a-f0-9]{40}$/.test(ski)) {
		throw new Error(
			`Invalid SKI format: "${ski}". SKI must be exactly 40 lowercase hexadecimal characters`,
		);
	}
}

/**
 * Normalize SKI to lowercase and validate
 * @param {string} ski - Device SKI to normalize
 * @returns {string} Normalized SKI (lowercase)
 * @throws {Error} If SKI format is invalid
 */
function normalizeSKI(ski) {
	if (!ski || typeof ski !== 'string') {
		throw new Error('SKI must be a non-empty string');
	}

	// Normalize first (lowercase and trim)
	const normalizedSKI = ski.toLowerCase().trim();

	// Then validate the normalized form
	validateSKI(normalizedSKI);

	return normalizedSKI;
}

module.exports = {
	validateSKI,
	normalizeSKI,
};
