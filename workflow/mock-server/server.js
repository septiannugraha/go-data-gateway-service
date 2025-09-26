/**
 * Mock Fusio Server for Testing n8n Workflow
 *
 * This server simulates the Fusio API endpoints needed for user registration.
 * Use this when the real Fusio instance is not available.
 */

const express = require('express');
const bodyParser = require('body-parser');
const crypto = require('crypto');

const app = express();
app.use(bodyParser.json());

// Store registered users (in-memory for testing)
const users = new Map();
const tokens = new Map();

// Middleware for logging
app.use((req, res, next) => {
  console.log(`[${new Date().toISOString()}] ${req.method} ${req.path}`);
  next();
});

// Health check
app.get('/health', (req, res) => {
  res.json({
    status: 'ok',
    service: 'Mock Fusio API',
    timestamp: new Date().toISOString()
  });
});

// Mock OAuth2 token endpoint
app.post('/authorization/token', (req, res) => {
  const { username, password, grant_type } = req.body;

  console.log('Auth request:', { username, grant_type });

  // Simple validation
  if (grant_type !== 'password' && grant_type !== 'client_credentials') {
    return res.status(400).json({
      error: 'unsupported_grant_type',
      error_description: 'Grant type not supported'
    });
  }

  // Generate mock token
  const token = crypto.randomBytes(32).toString('hex');
  const expiresIn = 3600;

  tokens.set(token, {
    username: username || 'admin',
    expires: Date.now() + (expiresIn * 1000)
  });

  res.json({
    access_token: token,
    token_type: 'Bearer',
    expires_in: expiresIn,
    refresh_token: crypto.randomBytes(32).toString('hex')
  });
});

// Mock user registration endpoint
app.post('/consumer/register', (req, res) => {
  const authHeader = req.headers.authorization;

  // Check API key or Bearer token
  if (!authHeader && !req.headers['x-api-key']) {
    return res.status(401).json({
      success: false,
      message: 'Authentication required'
    });
  }

  const { email, name, password, status = 1, roleId = 3 } = req.body;

  console.log('Registration request:', { email, name, hasPassword: !!password });

  // Validate required fields
  if (!email || !name || !password) {
    return res.status(400).json({
      success: false,
      message: 'Missing required fields: email, name, password'
    });
  }

  // Check if user already exists
  if (users.has(email)) {
    return res.status(409).json({
      success: false,
      message: 'User already exists'
    });
  }

  // Create user
  const userId = crypto.randomUUID();
  const user = {
    id: userId,
    email,
    name,
    password: crypto.createHash('sha256').update(password).digest('hex'),
    status,
    roleId,
    createdAt: new Date().toISOString()
  };

  users.set(email, user);

  // Return success response
  res.status(201).json({
    success: true,
    message: 'User registered successfully',
    data: {
      id: userId,
      email: email,
      name: name,
      status: status,
      roleId: roleId
    }
  });
});

// Mock user list endpoint (for testing)
app.get('/consumer/users', (req, res) => {
  const userList = Array.from(users.values()).map(u => ({
    id: u.id,
    email: u.email,
    name: u.name,
    status: u.status,
    createdAt: u.createdAt
  }));

  res.json({
    success: true,
    totalResults: userList.length,
    itemsPerPage: 16,
    startIndex: 0,
    entry: userList
  });
});

// Mock user detail endpoint
app.get('/consumer/user/:id', (req, res) => {
  const user = Array.from(users.values()).find(u => u.id === req.params.id);

  if (!user) {
    return res.status(404).json({
      success: false,
      message: 'User not found'
    });
  }

  res.json({
    success: true,
    data: {
      id: user.id,
      email: user.email,
      name: user.name,
      status: user.status,
      createdAt: user.createdAt
    }
  });
});

// Mock password reset endpoint
app.post('/consumer/password-reset', (req, res) => {
  const { email } = req.body;

  if (!email) {
    return res.status(400).json({
      success: false,
      message: 'Email is required'
    });
  }

  const user = users.get(email);
  if (!user) {
    // Don't reveal if user exists or not
    return res.json({
      success: true,
      message: 'If the email exists, a reset link has been sent'
    });
  }

  console.log(`Password reset requested for: ${email}`);

  res.json({
    success: true,
    message: 'Password reset link sent to email'
  });
});

// Start server
const PORT = process.env.MOCK_PORT || 8000;
const HOST = '0.0.0.0'; // Listen on all interfaces
app.listen(PORT, HOST, () => {
  console.log(`
=================================================================
Mock Fusio API Server
=================================================================
Server running at: http://${HOST}:${PORT}
Health check: http://${HOST}:${PORT}/health

Available endpoints:
- POST /authorization/token     - Get access token
- POST /consumer/register       - Register new user
- GET  /consumer/users          - List all users
- GET  /consumer/user/:id       - Get user details
- POST /consumer/password-reset - Request password reset

Test registration:
curl -X POST http://localhost:${PORT}/consumer/register \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: test-key" \\
  -d '{"email":"test@example.com","name":"Test User","password":"secure123"}'
=================================================================
  `);
});

// Graceful shutdown
process.on('SIGINT', () => {
  console.log('\nShutting down mock server...');
  process.exit(0);
});