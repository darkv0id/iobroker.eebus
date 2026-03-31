'use strict';

const { expect } = require('chai');
const { validateSKI, normalizeSKI } = require('./skiValidator');

describe('SKI Validator', () => {
	describe('validateSKI', () => {
		it('should accept valid lowercase 40-character hexadecimal SKI', () => {
			expect(() => validateSKI('69898d83b85363ab75428da04c4c31c52cf929f1')).to.not.throw();
			expect(() => validateSKI('0123456789abcdef0123456789abcdef01234567')).to.not.throw();
		});

		it('should reject SKI with uppercase characters', () => {
			expect(() => validateSKI('69898D83B85363AB75428DA04C4C31C52CF929F1')).to.throw(
				Error,
				/Invalid SKI format.*must be exactly 40 lowercase hexadecimal characters/,
			);
		});

		it('should reject SKI that is too short', () => {
			expect(() => validateSKI('69898d83b85363ab75428da04c4c31c52cf929')).to.throw(
				Error,
				/Invalid SKI format/,
			);
		});

		it('should reject SKI that is too long', () => {
			expect(() => validateSKI('69898d83b85363ab75428da04c4c31c52cf929f1a')).to.throw(
				Error,
				/Invalid SKI format/,
			);
		});

		it('should reject SKI with invalid characters', () => {
			expect(() => validateSKI('69898d83b85363ab75428da04c4c31c52cf929g1')).to.throw(
				Error,
				/Invalid SKI format/,
			);
			expect(() => validateSKI('69898d83-b853-63ab-7542-8da04c4c31c5')).to.throw(
				Error,
				/Invalid SKI format/,
			);
		});

		it('should reject empty string', () => {
			expect(() => validateSKI('')).to.throw(Error, /SKI must be a non-empty string/);
		});

		it('should reject null', () => {
			expect(() => validateSKI(null)).to.throw(Error, /SKI must be a non-empty string/);
		});

		it('should reject undefined', () => {
			expect(() => validateSKI(undefined)).to.throw(Error, /SKI must be a non-empty string/);
		});

		it('should reject non-string types', () => {
			expect(() => validateSKI(123)).to.throw(Error, /SKI must be a non-empty string/);
			expect(() => validateSKI({})).to.throw(Error, /SKI must be a non-empty string/);
			expect(() => validateSKI([])).to.throw(Error, /SKI must be a non-empty string/);
		});

		it('should reject SKI with whitespace', () => {
			expect(() => validateSKI(' 69898d83b85363ab75428da04c4c31c52cf929f1')).to.throw(
				Error,
				/Invalid SKI format/,
			);
			expect(() => validateSKI('69898d83b85363ab75428da04c4c31c52cf929f1 ')).to.throw(
				Error,
				/Invalid SKI format/,
			);
		});
	});

	describe('normalizeSKI', () => {
		it('should normalize uppercase SKI to lowercase', () => {
			expect(normalizeSKI('69898D83B85363AB75428DA04C4C31C52CF929F1')).to.equal(
				'69898d83b85363ab75428da04c4c31c52cf929f1',
			);
		});

		it('should normalize mixed case SKI to lowercase', () => {
			expect(normalizeSKI('69898d83B85363Ab75428Da04C4c31c52cF929f1')).to.equal(
				'69898d83b85363ab75428da04c4c31c52cf929f1',
			);
		});

		it('should trim whitespace from SKI', () => {
			expect(normalizeSKI('  69898d83b85363ab75428da04c4c31c52cf929f1  ')).to.equal(
				'69898d83b85363ab75428da04c4c31c52cf929f1',
			);
			expect(normalizeSKI('\t69898d83b85363ab75428da04c4c31c52cf929f1\n')).to.equal(
				'69898d83b85363ab75428da04c4c31c52cf929f1',
			);
		});

		it('should accept already normalized SKI', () => {
			expect(normalizeSKI('69898d83b85363ab75428da04c4c31c52cf929f1')).to.equal(
				'69898d83b85363ab75428da04c4c31c52cf929f1',
			);
		});

		it('should reject invalid SKI after normalization', () => {
			expect(() => normalizeSKI('INVALID')).to.throw(Error, /Invalid SKI format/);
			expect(() => normalizeSKI('too-short')).to.throw(Error, /Invalid SKI format/);
			expect(() => normalizeSKI('69898d83b85363ab75428da04c4c31c52cf929g1')).to.throw(
				Error,
				/Invalid SKI format/,
			);
		});

		it('should reject empty string', () => {
			expect(() => normalizeSKI('')).to.throw(Error, /SKI must be a non-empty string/);
		});

		it('should reject null', () => {
			expect(() => normalizeSKI(null)).to.throw(Error, /SKI must be a non-empty string/);
		});

		it('should reject undefined', () => {
			expect(() => normalizeSKI(undefined)).to.throw(Error, /SKI must be a non-empty string/);
		});

		it('should handle real-world SKI examples', () => {
			// Example from actual EEBus certificate
			const uppercase = '0F6E13495A33D6C041D8DAED2D053FCFC6439B91';
			const expected = '0f6e13495a33d6c041d8daed2d053fcfc6439b91';
			expect(normalizeSKI(uppercase)).to.equal(expected);
		});
	});
});
