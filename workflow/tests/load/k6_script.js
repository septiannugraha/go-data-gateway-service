/**
 * K6 Load Testing Script for User Onboarding Workflow
 * Tests performance under various load conditions
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { randomString, randomItem } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Custom metrics
const errorRate = new Rate('errors');
const registrationDuration = new Trend('registration_duration');
const webhookResponseTime = new Trend('webhook_response_time');

// Test configuration
export const options = {
  scenarios: {
    // Smoke test
    smoke: {
      executor: 'constant-vus',
      vus: 1,
      duration: '1m',
      tags: { test_type: 'smoke' },
    },
    // Load test
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 10 },  // Ramp up to 10 users
        { duration: '5m', target: 10 },  // Stay at 10 users
        { duration: '2m', target: 20 },  // Ramp up to 20 users
        { duration: '5m', target: 20 },  // Stay at 20 users
        { duration: '2m', target: 0 },   // Ramp down to 0 users
      ],
      gracefulRampDown: '30s',
      tags: { test_type: 'load' },
    },
    // Stress test
    stress: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 50 },   // Ramp up to 50 users
        { duration: '5m', target: 50 },   // Stay at 50 users
        { duration: '2m', target: 100 },  // Ramp up to 100 users
        { duration: '5m', target: 100 },  // Stay at 100 users
        { duration: '2m', target: 150 },  // Ramp up to 150 users
        { duration: '5m', target: 150 },  // Stay at 150 users
        { duration: '5m', target: 0 },    // Ramp down to 0 users
      ],
      gracefulRampDown: '30s',
      tags: { test_type: 'stress' },
    },
    // Spike test
    spike: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 5 },    // Normal load
        { duration: '1m', target: 5 },     // Stay at normal
        { duration: '10s', target: 100 },  // Spike to 100 users
        { duration: '3m', target: 100 },   // Stay at spike
        { duration: '10s', target: 5 },    // Back to normal
        { duration: '3m', target: 5 },     // Stay at normal
        { duration: '10s', target: 0 },    // Ramp down
      ],
      gracefulRampDown: '10s',
      tags: { test_type: 'spike' },
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000'], // 95% of requests should be below 2s
    http_req_failed: ['rate<0.1'],     // Error rate should be below 10%
    errors: ['rate<0.1'],               // Custom error rate below 10%
    registration_duration: ['p(95)<5000'], // 95% of registrations below 5s
  },
};

// Test configuration from environment
const WEBHOOK_URL = __ENV.WEBHOOK_URL || 'http://localhost:5679/webhook/google-forms-webhook';
const PASSPHRASE = __ENV.FORM_PASSPHRASE || 'spse2025';

// Test data generators
function generateTestUser() {
  const firstNames = ['John', 'Jane', 'Bob', 'Alice', 'Charlie', 'Diana', 'Eve', 'Frank'];
  const lastNames = ['Smith', 'Johnson', 'Williams', 'Brown', 'Jones', 'Garcia', 'Miller', 'Davis'];

  return {
    email: `load-test-${randomString(8)}@example.com`,
    name: `${randomItem(firstNames)} ${randomItem(lastNames)}`,
    passphrase: PASSPHRASE,
    timestamp: new Date().toISOString(),
    formId: `load-test-${randomString(6)}`,
  };
}

// Main test scenario
export default function () {
  const testUser = generateTestUser();

  // Start timer for registration duration
  const startTime = Date.now();

  // Prepare request
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '30s',
    tags: {
      name: 'UserRegistration',
    },
  };

  // Send registration request
  const response = http.post(WEBHOOK_URL, JSON.stringify(testUser), params);

  // Record custom metrics
  const duration = Date.now() - startTime;
  registrationDuration.add(duration);
  webhookResponseTime.add(response.timings.duration);

  // Check response
  const success = check(response, {
    'status is 200': (r) => r.status === 200,
    'response has success field': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.hasOwnProperty('success');
      } catch (e) {
        return false;
      }
    },
    'registration successful': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success === true;
      } catch (e) {
        return false;
      }
    },
    'response time < 2s': (r) => r.timings.duration < 2000,
    'response time < 5s': (r) => r.timings.duration < 5000,
  });

  // Record errors
  if (!success || response.status !== 200) {
    errorRate.add(1);
    console.error(`Registration failed for ${testUser.email}: ${response.status} - ${response.body}`);
  } else {
    errorRate.add(0);
  }

  // Think time between requests
  sleep(randomIntBetween(1, 3));
}

// Helper function for random integers
function randomIntBetween(min, max) {
  return Math.floor(Math.random() * (max - min + 1) + min);
}

// Test invalid passphrase scenario
export function testInvalidPassphrase() {
  const testUser = generateTestUser();
  testUser.passphrase = 'wrong_passphrase';

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '10s',
    tags: {
      name: 'InvalidPassphrase',
    },
  };

  const response = http.post(WEBHOOK_URL, JSON.stringify(testUser), params);

  check(response, {
    'status is 401': (r) => r.status === 401,
    'error message present': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.error && body.error.includes('passphrase');
      } catch (e) {
        return false;
      }
    },
  });

  sleep(1);
}

// Test missing fields scenario
export function testMissingFields() {
  const incompleteUser = {
    name: 'Test User',
    passphrase: PASSPHRASE,
    // Missing email
  };

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '10s',
    tags: {
      name: 'MissingFields',
    },
  };

  const response = http.post(WEBHOOK_URL, JSON.stringify(incompleteUser), params);

  check(response, {
    'status is 400 or 500': (r) => r.status === 400 || r.status === 500,
    'error for missing field': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success === false;
      } catch (e) {
        return true; // Error response is expected
      }
    },
  });

  sleep(1);
}

// Test concurrent duplicate registrations
export function testDuplicateRegistrations() {
  const email = `duplicate-test-${randomString(8)}@example.com`;
  const testUser = {
    email: email,
    name: 'Duplicate Test User',
    passphrase: PASSPHRASE,
    timestamp: new Date().toISOString(),
  };

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '10s',
    tags: {
      name: 'DuplicateRegistration',
    },
  };

  // Send two requests in quick succession
  const response1 = http.post(WEBHOOK_URL, JSON.stringify(testUser), params);
  const response2 = http.post(WEBHOOK_URL, JSON.stringify(testUser), params);

  check(response1, {
    'first request successful': (r) => r.status === 200,
  });

  check(response2, {
    'duplicate handled gracefully': (r) => r.status === 200 || r.status === 409,
  });

  sleep(2);
}

// Lifecycle hooks
export function setup() {
  console.log('Starting load test...');
  console.log(`Webhook URL: ${WEBHOOK_URL}`);

  // Test webhook is accessible
  const response = http.options(WEBHOOK_URL);
  if (response.status === 0) {
    throw new Error('Webhook endpoint is not accessible');
  }

  return { startTime: Date.now() };
}

export function teardown(data) {
  const duration = Date.now() - data.startTime;
  console.log(`Load test completed in ${duration}ms`);
}

// Custom summary
export function handleSummary(data) {
  return {
    'test-results/k6-summary.json': JSON.stringify(data),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

// Helper to generate text summary
function textSummary(data, options) {
  const { indent = '', enableColors = false } = options;
  let summary = '';

  summary += `${indent}Test Results Summary\n`;
  summary += `${indent}====================\n\n`;

  // Add metrics summary
  if (data.metrics) {
    summary += `${indent}Response Times:\n`;
    summary += `${indent}  Median: ${data.metrics.http_req_duration.med}ms\n`;
    summary += `${indent}  95th percentile: ${data.metrics.http_req_duration['p(95)']}ms\n`;
    summary += `${indent}  99th percentile: ${data.metrics.http_req_duration['p(99)']}ms\n\n`;

    summary += `${indent}Success Rate: ${(100 - data.metrics.http_req_failed.rate * 100).toFixed(2)}%\n`;
    summary += `${indent}Total Requests: ${data.metrics.http_reqs.count}\n`;
  }

  return summary;
}