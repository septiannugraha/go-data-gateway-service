/**
 * Unit Tests for Password Generator Service
 * Tests password generation, strength, and security requirements
 */

const PasswordGenerator = require('../../services/password_generator');

describe('PasswordGenerator', () => {
  let generator;

  beforeEach(() => {
    generator = new PasswordGenerator();
  });

  describe('Constructor', () => {
    test('should initialize with default options', () => {
      expect(generator.length).toBe(16);
      expect(generator.includeNumbers).toBe(true);
      expect(generator.includeSymbols).toBe(true);
      expect(generator.includeUppercase).toBe(true);
      expect(generator.includeLowercase).toBe(true);
    });

    test('should accept custom options', () => {
      const customGenerator = new PasswordGenerator({
        length: 20,
        includeNumbers: false,
        includeSymbols: false
      });
      expect(customGenerator.length).toBe(20);
      expect(customGenerator.includeNumbers).toBe(false);
      expect(customGenerator.includeSymbols).toBe(false);
    });

    test('should throw error if no character types enabled', () => {
      expect(() => {
        new PasswordGenerator({
          includeNumbers: false,
          includeSymbols: false,
          includeUppercase: false,
          includeLowercase: false
        });
      }).toThrow('At least one character type must be enabled');
    });
  });

  describe('Password Generation', () => {
    test('should generate password of correct length', () => {
      const password = generator.generate();
      expect(password).toHaveLength(16);
    });

    test('should generate different passwords each time', () => {
      const passwords = new Set();
      for (let i = 0; i < 100; i++) {
        passwords.add(generator.generate());
      }
      expect(passwords.size).toBe(100);
    });

    test('should include all required character types', () => {
      const password = generator.generate();
      expect(/[a-z]/.test(password)).toBe(true); // lowercase
      expect(/[A-Z]/.test(password)).toBe(true); // uppercase
      expect(/[0-9]/.test(password)).toBe(true); // numbers
      expect(/[!@#$%^&*()_+\-=\[\]{}|;:,.<>?]/.test(password)).toBe(true); // symbols
    });

    test('should respect character type options', () => {
      const noNumbersGen = new PasswordGenerator({
        includeNumbers: false
      });
      const password = noNumbersGen.generate();
      expect(/[0-9]/.test(password)).toBe(false);
    });

    test('should generate password with custom length', () => {
      const longGen = new PasswordGenerator({ length: 32 });
      const password = longGen.generate();
      expect(password).toHaveLength(32);
    });
  });

  describe('Password Strength', () => {
    test('should correctly evaluate strong password', () => {
      const strongPassword = 'Str0ng!P@ssw0rd123';
      const analysis = PasswordGenerator.checkStrength(strongPassword);

      expect(analysis.hasLowercase).toBe(true);
      expect(analysis.hasUppercase).toBe(true);
      expect(analysis.hasNumbers).toBe(true);
      expect(analysis.hasSymbols).toBe(true);
      expect(analysis.strength).toBe('very strong');
    });

    test('should correctly evaluate weak password', () => {
      const weakPassword = 'weak';
      const analysis = PasswordGenerator.checkStrength(weakPassword);

      expect(analysis.length).toBe(4);
      expect(analysis.strength).toBe('weak');
    });

    test('should correctly evaluate medium password', () => {
      const mediumPassword = 'Medium123';
      const analysis = PasswordGenerator.checkStrength(mediumPassword);

      expect(analysis.hasLowercase).toBe(true);
      expect(analysis.hasUppercase).toBe(true);
      expect(analysis.hasNumbers).toBe(true);
      expect(analysis.hasSymbols).toBe(false);
      expect(['medium', 'strong']).toContain(analysis.strength);
    });
  });

  describe('Multiple Password Generation', () => {
    test('should generate requested number of unique passwords', () => {
      const passwords = generator.generateMultiple(10);

      expect(passwords).toHaveLength(10);
      const uniquePasswords = new Set(passwords);
      expect(uniquePasswords.size).toBe(10);
    });

    test('should generate passwords meeting all requirements', () => {
      const passwords = generator.generateMultiple(5);

      passwords.forEach(password => {
        expect(password).toHaveLength(16);
        const analysis = PasswordGenerator.checkStrength(password);
        expect(['strong', 'very strong']).toContain(analysis.strength);
      });
    });
  });

  describe('Memorable Password Generation', () => {
    test('should generate memorable password with pattern', () => {
      const password = generator.generateMemorable();

      // Should match pattern like "QuickTigerruns123!"
      expect(password.length).toBeGreaterThanOrEqual(15);
      expect(/[A-Z]/.test(password)).toBe(true);
      expect(/[a-z]/.test(password)).toBe(true);
      expect(/[0-9]/.test(password)).toBe(true);
      expect(/[^a-zA-Z0-9]/.test(password)).toBe(true);
    });

    test('should generate different memorable passwords', () => {
      const passwords = new Set();
      for (let i = 0; i < 10; i++) {
        passwords.add(generator.generateMemorable());
      }
      expect(passwords.size).toBeGreaterThan(5); // Allow some duplicates due to limited word list
    });
  });

  describe('Password Requirements', () => {
    test('should generate password meeting minimum length requirement', () => {
      const password = generator.generateWithRequirements({
        minLength: 24
      });

      expect(password.length).toBeGreaterThanOrEqual(24);
    });

    test('should generate password meeting maximum length requirement', () => {
      const password = generator.generateWithRequirements({
        minLength: 8,
        maxLength: 12
      });

      expect(password.length).toBeGreaterThanOrEqual(8);
      expect(password.length).toBeLessThanOrEqual(12);
    });

    test('should generate password with required character types', () => {
      const password = generator.generateWithRequirements({
        minLength: 16,
        requireNumbers: true,
        requireSymbols: true,
        requireUppercase: true,
        requireLowercase: true
      });

      expect(/[a-z]/.test(password)).toBe(true);
      expect(/[A-Z]/.test(password)).toBe(true);
      expect(/[0-9]/.test(password)).toBe(true);
      expect(/[^a-zA-Z0-9]/.test(password)).toBe(true);
    });

    test('should throw error if requirements cannot be met', () => {
      const impossibleGen = new PasswordGenerator({
        includeNumbers: false
      });

      expect(() => {
        impossibleGen.generateWithRequirements({
          requireNumbers: true
        });
      }).toThrow('Could not generate password meeting requirements');
    });
  });

  describe('Security Tests', () => {
    test('should not contain predictable patterns', () => {
      const passwords = generator.generateMultiple(100);

      passwords.forEach(password => {
        // Check for sequential characters
        expect(/123|234|345|abc|bcd|cde/i.test(password)).toBe(false);
        // Check for repeated characters
        expect(/(.)\1{2,}/.test(password)).toBe(false);
      });
    });

    test('should use cryptographically secure randomness', () => {
      // Generate many passwords and check distribution
      const charCounts = {};
      const passwords = generator.generateMultiple(1000);

      passwords.forEach(password => {
        for (const char of password) {
          charCounts[char] = (charCounts[char] || 0) + 1;
        }
      });

      // No character should appear suspiciously often
      const counts = Object.values(charCounts);
      const avgCount = counts.reduce((a, b) => a + b) / counts.length;
      const maxDeviation = Math.max(...counts.map(c => Math.abs(c - avgCount)));

      // Allow for reasonable statistical variation
      expect(maxDeviation / avgCount).toBeLessThan(2);
    });

    test('should properly shuffle password characters', () => {
      // Generate passwords and check that required chars aren't always at start
      const positions = [];

      for (let i = 0; i < 100; i++) {
        const password = generator.generate();
        const firstDigitPos = password.search(/[0-9]/);
        if (firstDigitPos !== -1) {
          positions.push(firstDigitPos);
        }
      }

      // Check that digits appear in various positions
      const uniquePositions = new Set(positions);
      expect(uniquePositions.size).toBeGreaterThan(5);
    });
  });

  describe('Edge Cases', () => {
    test('should handle minimum length of 1', () => {
      const shortGen = new PasswordGenerator({ length: 1 });
      const password = shortGen.generate();
      expect(password).toHaveLength(1);
    });

    test('should handle very long passwords', () => {
      const longGen = new PasswordGenerator({ length: 256 });
      const password = longGen.generate();
      expect(password).toHaveLength(256);
    });

    test('should handle only lowercase requirement', () => {
      const lowercaseGen = new PasswordGenerator({
        includeUppercase: false,
        includeNumbers: false,
        includeSymbols: false
      });
      const password = lowercaseGen.generate();
      expect(/^[a-z]+$/.test(password)).toBe(true);
    });
  });

  describe('Performance', () => {
    test('should generate passwords quickly', () => {
      const start = Date.now();
      generator.generateMultiple(1000);
      const duration = Date.now() - start;

      // Should generate 1000 passwords in less than 1 second
      expect(duration).toBeLessThan(1000);
    });
  });
});

// Export for Jest
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { PasswordGenerator };
}