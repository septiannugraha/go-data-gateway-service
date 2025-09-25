/**
 * Local Workflow Test - Simulates n8n workflow without n8n
 * This tests the complete registration flow logic
 */

const PasswordGenerator = require('./services/password_generator');

// Color codes for output
const colors = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m'
};

// Simulated workflow execution
class WorkflowSimulator {
  constructor() {
    this.passphrase = process.env.FORM_PASSPHRASE || 'spse2025';
    this.results = [];
  }

  // Webhook trigger simulation
  async webhookTrigger(data) {
    console.log(`${colors.blue}[Webhook Trigger]${colors.reset} Received:`, JSON.stringify(data, null, 2));
    return data;
  }

  // Validate passphrase node
  async validatePassphrase(data) {
    console.log(`${colors.blue}[Validate Passphrase]${colors.reset} Checking...`);

    if (data.passphrase === this.passphrase) {
      console.log(`${colors.green}✅ Passphrase valid${colors.reset}`);
      return { ...data, passphraseValid: true };
    } else {
      console.log(`${colors.red}❌ Invalid passphrase${colors.reset}`);
      throw new Error('Invalid passphrase');
    }
  }

  // Generate password node
  async generatePassword(data) {
    console.log(`${colors.blue}[Generate Password]${colors.reset} Creating secure password...`);

    const generator = new PasswordGenerator({
      length: 16,
      includeNumbers: true,
      includeSymbols: true,
      includeUppercase: true,
      includeLowercase: true
    });

    const password = generator.generate();
    const strength = PasswordGenerator.checkStrength(password);

    console.log(`${colors.green}✅ Password generated${colors.reset} (strength: ${strength.strength})`);

    return {
      ...data,
      password: password,
      passwordStrength: strength.strength
    };
  }

  // Mock Fusio registration
  async registerFusioUser(data) {
    console.log(`${colors.blue}[Register Fusio User]${colors.reset} Calling API...`);

    // Simulate API call
    await this.delay(500);

    // Mock response
    const response = {
      success: true,
      userId: `user-${Date.now()}`,
      message: 'User registered successfully'
    };

    console.log(`${colors.green}✅ User registered${colors.reset} (ID: ${response.userId})`);

    return {
      ...data,
      fusioResponse: response
    };
  }

  // Send email simulation
  async sendEmail(data) {
    console.log(`${colors.blue}[Send Email]${colors.reset} Sending credentials...`);

    // Simulate email sending
    await this.delay(300);

    const emailContent = `
    To: ${data.email}
    Subject: Welcome to SPSE Data Gateway

    Dear ${data.name},

    Your account has been created successfully!

    Credentials:
    - Email: ${data.email}
    - Password: ${data.password}

    Please login at: http://fusio-gateway.com/login
    `;

    console.log(`${colors.green}✅ Email sent${colors.reset}`);
    console.log(`${colors.yellow}Email Preview:${colors.reset}`);
    console.log(emailContent);

    return {
      ...data,
      emailSent: true
    };
  }

  // Helper to simulate delay
  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // Execute complete workflow
  async executeWorkflow(testData) {
    console.log(`\n${colors.blue}${'='.repeat(60)}${colors.reset}`);
    console.log(`${colors.blue}WORKFLOW EXECUTION START${colors.reset}`);
    console.log(`${colors.blue}${'='.repeat(60)}${colors.reset}\n`);

    const startTime = Date.now();

    try {
      // Step 1: Webhook receives data
      let data = await this.webhookTrigger(testData);

      // Step 2: Validate passphrase
      data = await this.validatePassphrase(data);

      // Step 3: Generate password
      data = await this.generatePassword(data);

      // Step 4: Register user in Fusio
      data = await this.registerFusioUser(data);

      // Step 5: Send email
      data = await this.sendEmail(data);

      const duration = Date.now() - startTime;

      console.log(`\n${colors.green}${'='.repeat(60)}${colors.reset}`);
      console.log(`${colors.green}✅ WORKFLOW COMPLETED SUCCESSFULLY${colors.reset}`);
      console.log(`Duration: ${duration}ms`);
      console.log(`${colors.green}${'='.repeat(60)}${colors.reset}`);

      return {
        success: true,
        duration: duration,
        data: data
      };

    } catch (error) {
      const duration = Date.now() - startTime;

      console.log(`\n${colors.red}${'='.repeat(60)}${colors.reset}`);
      console.log(`${colors.red}❌ WORKFLOW FAILED${colors.reset}`);
      console.log(`Error: ${error.message}`);
      console.log(`Duration: ${duration}ms`);
      console.log(`${colors.red}${'='.repeat(60)}${colors.reset}`);

      return {
        success: false,
        error: error.message,
        duration: duration
      };
    }
  }
}

// Test runner
async function runTests() {
  const simulator = new WorkflowSimulator();
  const testCases = [
    {
      name: 'Valid Registration',
      data: {
        email: 'john.doe@example.com',
        name: 'John Doe',
        passphrase: 'spse2025',
        timestamp: new Date().toISOString()
      },
      shouldSucceed: true
    },
    {
      name: 'Invalid Passphrase',
      data: {
        email: 'jane.doe@example.com',
        name: 'Jane Doe',
        passphrase: 'wrong_pass',
        timestamp: new Date().toISOString()
      },
      shouldSucceed: false
    },
    {
      name: 'Special Characters in Name',
      data: {
        email: 'test@example.com',
        name: "O'Connor-Smith",
        passphrase: 'spse2025',
        timestamp: new Date().toISOString()
      },
      shouldSucceed: true
    }
  ];

  console.log(`${colors.yellow}🧪 n8n Workflow Logic Test${colors.reset}`);
  console.log(`${colors.yellow}Testing without n8n running${colors.reset}\n`);

  const results = [];

  for (const testCase of testCases) {
    console.log(`\n${colors.yellow}TEST: ${testCase.name}${colors.reset}`);

    const result = await simulator.executeWorkflow(testCase.data);

    const passed = result.success === testCase.shouldSucceed;
    results.push({
      name: testCase.name,
      passed: passed,
      duration: result.duration
    });

    if (passed) {
      console.log(`${colors.green}✅ Test passed as expected${colors.reset}`);
    } else {
      console.log(`${colors.red}❌ Test failed unexpectedly${colors.reset}`);
    }
  }

  // Summary
  console.log(`\n${colors.blue}${'='.repeat(60)}${colors.reset}`);
  console.log(`${colors.blue}TEST SUMMARY${colors.reset}`);
  console.log(`${colors.blue}${'='.repeat(60)}${colors.reset}\n`);

  const passed = results.filter(r => r.passed).length;
  const total = results.length;

  results.forEach(r => {
    const status = r.passed ? `${colors.green}✅ PASS${colors.reset}` : `${colors.red}❌ FAIL${colors.reset}`;
    console.log(`${status} - ${r.name} (${r.duration}ms)`);
  });

  console.log(`\nResults: ${passed}/${total} tests passed`);

  if (passed === total) {
    console.log(`${colors.green}🎉 All workflow tests passed!${colors.reset}`);
  } else {
    console.log(`${colors.red}⚠️ Some tests failed${colors.reset}`);
  }

  // Instructions for real n8n
  console.log(`\n${colors.yellow}📝 To test with real n8n:${colors.reset}`);
  console.log('1. Install n8n: npm install -g n8n');
  console.log('2. Start n8n: n8n start');
  console.log('3. Import workflow: n8n/workflows/user_onboarding.json');
  console.log('4. Access at: http://localhost:5678');
  console.log('5. Test webhook: curl -X POST http://localhost:5678/webhook/google-forms-webhook');
}

// Run tests
runTests().catch(console.error);