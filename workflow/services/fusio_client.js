/**
 * Fusio API Client for User Registration
 *
 * This service handles programmatic user creation in Fusio API Gateway
 * Used by n8n workflow to automate user onboarding from Google Forms
 */

const axios = require('axios');

class FusioClient {
    constructor(config = {}) {
        this.apiUrl = config.apiUrl || process.env.FUSIO_API_URL;
        this.apiKey = config.apiKey || process.env.FUSIO_API_KEY;
        this.adminUser = config.adminUser || process.env.FUSIO_ADMIN_USER;
        this.adminPassword = config.adminPassword || process.env.FUSIO_ADMIN_PASSWORD;
        this.timeout = config.timeout || 30000;
        this.accessToken = null;
        this.tokenExpiry = null;
    }

    /**
     * Authenticate as admin to get access token
     * Required for creating users programmatically
     */
    async authenticate() {
        try {
            const response = await axios.post(
                `${this.apiUrl}/consumer/login`,
                {
                    username: this.adminUser,
                    password: this.adminPassword
                },
                {
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    timeout: this.timeout
                }
            );

            if (response.data && response.data.token) {
                this.accessToken = response.data.token;
                // Token typically expires in 1 hour, refresh after 50 minutes
                this.tokenExpiry = Date.now() + (50 * 60 * 1000);
                return this.accessToken;
            }

            throw new Error('Failed to obtain access token from Fusio');
        } catch (error) {
            throw new Error(`Fusio authentication failed: ${error.message}`);
        }
    }

    /**
     * Check if token needs refresh and refresh if needed
     */
    async ensureAuthenticated() {
        if (!this.accessToken || Date.now() >= this.tokenExpiry) {
            await this.authenticate();
        }
    }

    /**
     * Register a new user in Fusio
     *
     * @param {Object} userData - User registration data
     * @param {string} userData.name - User's full name
     * @param {string} userData.email - User's email address
     * @param {string} userData.password - Generated secure password
     * @param {Array} userData.scopes - Optional API scopes
     * @returns {Object} Registration result
     */
    async registerUser(userData) {
        try {
            await this.ensureAuthenticated();

            const registrationData = {
                name: userData.name,
                email: userData.email,
                password: userData.password,
                status: 1, // Active status
                scopes: userData.scopes || ['consumer', 'api']
            };

            const response = await axios.post(
                `${this.apiUrl}/consumer/register`,
                registrationData,
                {
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${this.accessToken}`,
                        'X-API-Key': this.apiKey
                    },
                    timeout: this.timeout
                }
            );

            return {
                success: true,
                userId: response.data.id,
                message: 'User registered successfully',
                requiresActivation: true,
                data: response.data
            };
        } catch (error) {
            // Handle specific error cases
            if (error.response) {
                if (error.response.status === 409) {
                    return {
                        success: false,
                        error: 'User already exists',
                        details: error.response.data
                    };
                }
                if (error.response.status === 400) {
                    return {
                        success: false,
                        error: 'Invalid registration data',
                        details: error.response.data
                    };
                }
            }

            return {
                success: false,
                error: `Registration failed: ${error.message}`,
                details: error.response?.data
            };
        }
    }

    /**
     * Activate a user account (skip email verification)
     * This is used when we want to auto-activate accounts
     *
     * @param {string} email - User's email to activate
     * @returns {Object} Activation result
     */
    async activateUser(email) {
        try {
            await this.ensureAuthenticated();

            // Admin endpoint to directly activate user
            const response = await axios.put(
                `${this.apiUrl}/backend/user`,
                {
                    email: email,
                    status: 1 // Active status
                },
                {
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${this.accessToken}`,
                        'X-API-Key': this.apiKey
                    },
                    timeout: this.timeout
                }
            );

            return {
                success: true,
                message: 'User activated successfully',
                data: response.data
            };
        } catch (error) {
            return {
                success: false,
                error: `Activation failed: ${error.message}`,
                details: error.response?.data
            };
        }
    }

    /**
     * Check if a user exists
     *
     * @param {string} email - Email to check
     * @returns {boolean} True if user exists
     */
    async userExists(email) {
        try {
            await this.ensureAuthenticated();

            const response = await axios.get(
                `${this.apiUrl}/backend/user`,
                {
                    params: { email: email },
                    headers: {
                        'Authorization': `Bearer ${this.accessToken}`,
                        'X-API-Key': this.apiKey
                    },
                    timeout: this.timeout
                }
            );

            return response.data && response.data.entry && response.data.entry.length > 0;
        } catch (error) {
            console.error(`Error checking user existence: ${error.message}`);
            return false;
        }
    }

    /**
     * Create user with auto-activation (complete flow)
     *
     * @param {Object} userData - User data
     * @returns {Object} Complete registration result
     */
    async createAndActivateUser(userData) {
        try {
            // Check if user already exists
            const exists = await this.userExists(userData.email);
            if (exists) {
                return {
                    success: false,
                    error: 'User already exists',
                    email: userData.email
                };
            }

            // Register the user
            const registration = await this.registerUser(userData);
            if (!registration.success) {
                return registration;
            }

            // Auto-activate the account
            const activation = await this.activateUser(userData.email);
            if (!activation.success) {
                return {
                    success: false,
                    error: 'User created but activation failed',
                    registrationId: registration.userId,
                    activationError: activation.error
                };
            }

            return {
                success: true,
                message: 'User created and activated successfully',
                userId: registration.userId,
                email: userData.email,
                password: userData.password,
                activated: true
            };
        } catch (error) {
            return {
                success: false,
                error: `Complete registration failed: ${error.message}`
            };
        }
    }

    /**
     * Generate API key for a user
     * Note: This typically needs to be done by the user after login
     *
     * @param {string} userToken - User's authentication token
     * @param {string} appName - Name for the API key
     * @returns {Object} API key information
     */
    async generateApiKey(userToken, appName = 'Default App') {
        try {
            const response = await axios.post(
                `${this.apiUrl}/consumer/app`,
                {
                    name: appName,
                    url: 'http://localhost',
                    scopes: ['consumer', 'api']
                },
                {
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${userToken}`
                    },
                    timeout: this.timeout
                }
            );

            return {
                success: true,
                apiKey: response.data.appKey,
                apiSecret: response.data.appSecret,
                appId: response.data.id
            };
        } catch (error) {
            return {
                success: false,
                error: `API key generation failed: ${error.message}`
            };
        }
    }
}

// Export for use in n8n Code node
module.exports = FusioClient;

// Example usage (for testing)
if (require.main === module) {
    const client = new FusioClient({
        apiUrl: 'http://localhost:8000',
        apiKey: 'test-api-key',
        adminUser: 'admin',
        adminPassword: 'admin-password'
    });

    // Test user registration
    async function testRegistration() {
        try {
            const result = await client.createAndActivateUser({
                name: 'Test User',
                email: 'test@example.com',
                password: 'SecureP@ssw0rd123!'
            });
            console.log('Registration result:', result);
        } catch (error) {
            console.error('Test failed:', error);
        }
    }

    // Uncomment to test
    // testRegistration();
}