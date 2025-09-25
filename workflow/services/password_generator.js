/**
 * Secure Password Generator Service
 *
 * Generates cryptographically secure passwords for new users
 * Used by n8n workflow during automated user registration
 */

const crypto = require('crypto');

class PasswordGenerator {
    constructor(options = {}) {
        this.length = options.length || parseInt(process.env.PASSWORD_LENGTH) || 16;
        this.includeNumbers = options.includeNumbers !== false;
        this.includeSymbols = options.includeSymbols !== false;
        this.includeUppercase = options.includeUppercase !== false;
        this.includeLowercase = options.includeLowercase !== false;

        // Character sets
        this.lowercase = 'abcdefghijklmnopqrstuvwxyz';
        this.uppercase = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
        this.numbers = '0123456789';
        this.symbols = '!@#$%^&*()_+-=[]{}|;:,.<>?';

        // Build character pool based on options
        this.buildCharacterPool();
    }

    /**
     * Build the character pool based on configured options
     */
    buildCharacterPool() {
        this.characterPool = '';
        this.requiredChars = [];

        if (this.includeLowercase) {
            this.characterPool += this.lowercase;
            this.requiredChars.push(this.lowercase);
        }
        if (this.includeUppercase) {
            this.characterPool += this.uppercase;
            this.requiredChars.push(this.uppercase);
        }
        if (this.includeNumbers) {
            this.characterPool += this.numbers;
            this.requiredChars.push(this.numbers);
        }
        if (this.includeSymbols) {
            this.characterPool += this.symbols;
            this.requiredChars.push(this.symbols);
        }

        if (this.characterPool.length === 0) {
            throw new Error('At least one character type must be enabled');
        }
    }

    /**
     * Generate a secure random password
     *
     * @returns {string} Generated password
     */
    generate() {
        let password = [];

        // Ensure at least one character from each required set
        for (const charSet of this.requiredChars) {
            const randomIndex = crypto.randomInt(0, charSet.length);
            password.push(charSet[randomIndex]);
        }

        // Fill the rest with random characters from the pool
        const remainingLength = this.length - password.length;
        for (let i = 0; i < remainingLength; i++) {
            const randomIndex = crypto.randomInt(0, this.characterPool.length);
            password.push(this.characterPool[randomIndex]);
        }

        // Shuffle the password to avoid predictable patterns
        password = this.shuffleArray(password);

        return password.join('');
    }

    /**
     * Shuffle an array using Fisher-Yates algorithm
     *
     * @param {Array} array - Array to shuffle
     * @returns {Array} Shuffled array
     */
    shuffleArray(array) {
        const shuffled = [...array];
        for (let i = shuffled.length - 1; i > 0; i--) {
            const j = crypto.randomInt(0, i + 1);
            [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
        }
        return shuffled;
    }

    /**
     * Generate multiple unique passwords
     *
     * @param {number} count - Number of passwords to generate
     * @returns {Array<string>} Array of generated passwords
     */
    generateMultiple(count) {
        const passwords = new Set();
        while (passwords.size < count) {
            passwords.add(this.generate());
        }
        return Array.from(passwords);
    }

    /**
     * Check password strength
     *
     * @param {string} password - Password to check
     * @returns {Object} Strength analysis
     */
    static checkStrength(password) {
        const analysis = {
            length: password.length,
            hasLowercase: /[a-z]/.test(password),
            hasUppercase: /[A-Z]/.test(password),
            hasNumbers: /[0-9]/.test(password),
            hasSymbols: /[^a-zA-Z0-9]/.test(password),
            strength: 'weak'
        };

        // Calculate strength score
        let score = 0;
        if (analysis.length >= 8) score++;
        if (analysis.length >= 12) score++;
        if (analysis.length >= 16) score++;
        if (analysis.hasLowercase) score++;
        if (analysis.hasUppercase) score++;
        if (analysis.hasNumbers) score++;
        if (analysis.hasSymbols) score++;

        // Determine strength level
        if (score >= 6) analysis.strength = 'very strong';
        else if (score >= 5) analysis.strength = 'strong';
        else if (score >= 3) analysis.strength = 'medium';
        else analysis.strength = 'weak';

        analysis.score = score;
        return analysis;
    }

    /**
     * Generate a memorable password using word combinations
     *
     * @returns {string} Memorable password
     */
    generateMemorable() {
        const adjectives = [
            'quick', 'lazy', 'happy', 'brave', 'bright',
            'calm', 'eager', 'fancy', 'gentle', 'jolly'
        ];
        const nouns = [
            'tiger', 'eagle', 'shark', 'panda', 'falcon',
            'dragon', 'phoenix', 'lion', 'wolf', 'bear'
        ];
        const verbs = [
            'runs', 'jumps', 'flies', 'swims', 'hunts',
            'plays', 'sleeps', 'eats', 'walks', 'climbs'
        ];

        const adjective = adjectives[crypto.randomInt(0, adjectives.length)];
        const noun = nouns[crypto.randomInt(0, nouns.length)];
        const verb = verbs[crypto.randomInt(0, verbs.length)];
        const number = crypto.randomInt(100, 999);
        const symbol = this.symbols[crypto.randomInt(0, this.symbols.length)];

        // Capitalize first letter of each word
        const capitalizedAdjective = adjective.charAt(0).toUpperCase() + adjective.slice(1);
        const capitalizedNoun = noun.charAt(0).toUpperCase() + noun.slice(1);

        return `${capitalizedAdjective}${capitalizedNoun}${verb}${number}${symbol}`;
    }

    /**
     * Generate password with specific requirements
     *
     * @param {Object} requirements - Password requirements
     * @returns {string} Generated password meeting requirements
     */
    generateWithRequirements(requirements) {
        const generator = new PasswordGenerator({
            length: requirements.minLength || 16,
            includeNumbers: requirements.requireNumbers !== false,
            includeSymbols: requirements.requireSymbols !== false,
            includeUppercase: requirements.requireUppercase !== false,
            includeLowercase: requirements.requireLowercase !== false
        });

        let password;
        let attempts = 0;
        const maxAttempts = 100;

        do {
            password = generator.generate();
            attempts++;

            // Check if password meets all requirements
            const meetsRequirements =
                password.length >= (requirements.minLength || 8) &&
                password.length <= (requirements.maxLength || 128) &&
                (!requirements.requireNumbers || /[0-9]/.test(password)) &&
                (!requirements.requireSymbols || /[^a-zA-Z0-9]/.test(password)) &&
                (!requirements.requireUppercase || /[A-Z]/.test(password)) &&
                (!requirements.requireLowercase || /[a-z]/.test(password));

            if (meetsRequirements) {
                return password;
            }
        } while (attempts < maxAttempts);

        throw new Error('Could not generate password meeting requirements');
    }
}

// Export for use in n8n
module.exports = PasswordGenerator;

// Example usage and testing
if (require.main === module) {
    const generator = new PasswordGenerator({
        length: 16,
        includeNumbers: true,
        includeSymbols: true,
        includeUppercase: true,
        includeLowercase: true
    });

    console.log('Generated Passwords:');
    console.log('===================');

    // Generate standard passwords
    for (let i = 0; i < 5; i++) {
        const password = generator.generate();
        const strength = PasswordGenerator.checkStrength(password);
        console.log(`${i + 1}. ${password} (${strength.strength}, score: ${strength.score}/7)`);
    }

    console.log('\nMemorable Passwords:');
    console.log('===================');

    // Generate memorable passwords
    for (let i = 0; i < 3; i++) {
        const password = generator.generateMemorable();
        const strength = PasswordGenerator.checkStrength(password);
        console.log(`${i + 1}. ${password} (${strength.strength})`);
    }

    console.log('\nWith Specific Requirements:');
    console.log('===========================');

    // Generate with specific requirements
    const customPassword = generator.generateWithRequirements({
        minLength: 20,
        maxLength: 24,
        requireNumbers: true,
        requireSymbols: true,
        requireUppercase: true,
        requireLowercase: true
    });
    const customStrength = PasswordGenerator.checkStrength(customPassword);
    console.log(`Custom: ${customPassword} (${customStrength.strength})`);
}