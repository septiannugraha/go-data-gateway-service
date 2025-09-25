# n8n Setup & Testing Guide

## 🚀 Quick Start Options

Since you're experiencing Docker credential issues, here are multiple ways to get n8n running:

## Option 1: n8n Desktop App (Easiest)

```bash
# Download n8n Desktop for your OS
# Windows/Mac/Linux versions available at:
https://n8n.io/download

# Or use npm to run locally (no Docker needed!)
npm install -g n8n
n8n start
```

Access at: http://localhost:5678

## Option 2: Fix Docker & Run n8n

### Fix Docker Credentials Issue

```bash
# Remove the problematic credential helper
rm ~/.docker/config.json

# Or edit it and remove the "credsStore" line
nano ~/.docker/config.json
# Remove: "credsStore": "desktop.exe"

# Then login to Docker Hub (optional)
docker login

# Now start n8n
cd /home/septiannugraha/code/go-data-gateway/workflow
docker run -it --rm \
  --name n8n \
  -p 5678:5678 \
  -v ~/.n8n:/home/node/.n8n \
  n8nio/n8n
```

## Option 3: Docker Compose (Fixed Version)

```bash
# Remove version warning
sed -i '1d' docker-compose.yml

# Start n8n and PostgreSQL
docker-compose up -d

# Check logs
docker-compose logs -f n8n
```

## Option 4: Use Existing Docker Containers

Since you have other containers running, let's add n8n to the same network:

```bash
# Run n8n connected to your existing network
docker run -d \
  --name n8n \
  --network host \
  -v ~/.n8n:/home/node/.n8n \
  -v $(pwd)/n8n/workflows:/home/node/.n8n/workflows \
  -e N8N_BASIC_AUTH_ACTIVE=false \
  -e N8N_HOST=localhost \
  -e N8N_PORT=5678 \
  -e N8N_PROTOCOL=http \
  -e WEBHOOK_URL=http://localhost:5678 \
  -e EXECUTIONS_DATA_SAVE_ON_ERROR=all \
  -e EXECUTIONS_DATA_SAVE_ON_SUCCESS=all \
  -e FUSIO_API_URL=http://localhost:8000 \
  -e FORM_PASSPHRASE=spse2025 \
  n8nio/n8n
```

## 🧪 Testing n8n Workflow

### Step 1: Access n8n

Open browser to: http://localhost:5678

### Step 2: Import the Workflow

1. Click **"Workflows"** → **"Add Workflow"**
2. Click menu (⋮) → **"Import from File"**
3. Select: `/home/septiannugraha/code/go-data-gateway/workflow/n8n/workflows/user_onboarding.json`

### Step 3: Configure Credentials

#### For Testing (No Email Required)

1. Replace Gmail node with a simple logging node
2. Or use MailHog for testing:

```bash
# Run MailHog for email testing
docker run -d \
  --name mailhog \
  -p 1025:1025 \
  -p 8025:8025 \
  mailhog/mailhog

# Access MailHog UI at: http://localhost:8025
```

### Step 4: Test the Webhook

```bash
# Test with valid data
curl -X POST http://localhost:5678/webhook/google-forms-webhook \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "name": "Test User",
    "passphrase": "spse2025"
  }'

# Test with invalid passphrase
curl -X POST http://localhost:5678/webhook/google-forms-webhook \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "name": "Test User",
    "passphrase": "wrong"
  }'
```

## 📝 Manual Workflow Testing in n8n

### Without Docker - Using n8n UI

1. **Install n8n globally**:
   ```bash
   npm install -g n8n
   n8n start
   ```

2. **Create Test Workflow**:
   - Add **Webhook** node
   - Add **IF** node (check passphrase)
   - Add **Code** node (generate password)
   - Add **HTTP Request** node (mock Fusio)
   - Add **Set** node (format response)

3. **Test Execution**:
   - Click **"Execute Workflow"**
   - Use the test webhook URL provided
   - Send test requests

## 🔧 Troubleshooting n8n

### Issue: Docker credential error

```bash
# Solution 1: Remove Docker config
rm ~/.docker/config.json

# Solution 2: Use podman instead
podman run -d -p 5678:5678 n8nio/n8n

# Solution 3: Run without Docker
npm install -g n8n && n8n start
```

### Issue: Port 5678 in use

```bash
# Check what's using it
sudo lsof -i :5678

# Use different port
docker run -p 5679:5678 n8nio/n8n
# Access at: http://localhost:5679
```

### Issue: Workflow not triggering

```bash
# Check webhook URL
curl http://localhost:5678/webhook/google-forms-webhook

# Check n8n logs
docker logs n8n

# Verify workflow is active (toggle in UI)
```

## 🎯 Testing Without Full Infrastructure

### Mock Testing Setup

```javascript
// Save as test-workflow.js
const axios = require('axios');

async function testWorkflow() {
  const webhookUrl = 'http://localhost:5678/webhook/google-forms-webhook';

  // Test cases
  const tests = [
    {
      name: 'Valid registration',
      data: {
        email: 'test@example.com',
        name: 'Test User',
        passphrase: 'spse2025'
      },
      expectedStatus: 200
    },
    {
      name: 'Invalid passphrase',
      data: {
        email: 'test@example.com',
        name: 'Test User',
        passphrase: 'wrong'
      },
      expectedStatus: 401
    }
  ];

  for (const test of tests) {
    try {
      console.log(`Running: ${test.name}`);
      const response = await axios.post(webhookUrl, test.data);
      console.log(`✓ Status: ${response.status}`);
      console.log(`  Response:`, response.data);
    } catch (error) {
      console.log(`✓ Status: ${error.response?.status || 'Error'}`);
      console.log(`  Message: ${error.message}`);
    }
    console.log('---');
  }
}

// Run if n8n is running
testWorkflow();
```

Run with: `node test-workflow.js`

## 🌐 n8n Cloud Alternative

If local setup is problematic, use n8n Cloud (free tier):

1. Sign up at: https://n8n.io/cloud
2. Import workflow JSON
3. Get cloud webhook URL
4. Test from anywhere!

## ✅ Quick Validation Checklist

- [ ] n8n accessible at http://localhost:5678
- [ ] Workflow imported successfully
- [ ] Webhook URL responding
- [ ] Test request returns response
- [ ] Execution visible in n8n UI

## 🚦 Simple Test Script

```bash
#!/bin/bash
# save as test-n8n.sh

echo "Testing n8n setup..."

# Check if n8n is running
if curl -s http://localhost:5678/healthz > /dev/null; then
  echo "✅ n8n is running"
else
  echo "❌ n8n not accessible"
  echo "Try: npm install -g n8n && n8n start"
  exit 1
fi

# Test webhook
echo "Testing webhook..."
response=$(curl -s -w "\n%{http_code}" -X POST \
  http://localhost:5678/webhook/google-forms-webhook \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","name":"Test","passphrase":"spse2025"}')

status=$(echo "$response" | tail -n1)

if [ "$status" = "200" ]; then
  echo "✅ Webhook working!"
else
  echo "⚠️  Webhook returned status: $status"
fi

echo "Check n8n UI at: http://localhost:5678"
```

## 💡 Recommended Approach

Given your Docker issues, I recommend:

1. **Use npm to run n8n locally** (easiest)
2. **Import the workflow**
3. **Test with curl commands**
4. **Use MailHog for email testing**

This avoids Docker complexity while still testing the workflow!