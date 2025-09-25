# Detailed Setup Guide - User Onboarding Workflow

This guide provides step-by-step instructions for setting up the automated user onboarding workflow.

## Prerequisites Checklist

- [ ] Docker installed (version 20.10+)
- [ ] Docker Compose installed (version 1.29+)
- [ ] Google account with access to Forms
- [ ] Gmail account with 2FA enabled
- [ ] Fusio API Gateway installed and running
- [ ] Admin credentials for Fusio
- [ ] Basic understanding of webhooks

## Step 1: Google Forms Configuration

### Create the Registration Form

1. Go to [Google Forms](https://forms.google.com)
2. Click "Blank Form" to create new
3. Set form title: "SPSE Data Gateway Registration"
4. Add the following fields:

   **Field 1: Email Address**
   - Type: Short answer
   - Question: "Email Address"
   - Required: Yes
   - Validation: Is email address

   **Field 2: Full Name**
   - Type: Short answer
   - Question: "Full Name"
   - Required: Yes
   - Validation: Maximum character count = 100

   **Field 3: Organization**
   - Type: Short answer
   - Question: "Organization/Company"
   - Required: No

   **Field 4: Passphrase**
   - Type: Short answer
   - Question: "Registration Passphrase"
   - Required: Yes
   - Description: "Enter the passphrase provided by the administrator"

5. Click Settings (⚙️) and configure:
   - Collect email addresses: OFF (we have our own field)
   - Limit to 1 response: Optional
   - Edit after submit: No

### Install FormLinker Add-on

1. In your Google Form, click the three dots menu (⋮)
2. Select "Add-ons"
3. Search for "FormLinker"
4. Click "Install" and grant permissions
5. After installation, click "Add-ons" → "FormLinker" → "Configure"

## Step 2: Gmail Configuration for Email Sending

### Enable 2-Factor Authentication

1. Go to [Google Account Security](https://myaccount.google.com/security)
2. Click "2-Step Verification"
3. Follow the setup wizard to enable 2FA
4. Add your phone number for verification

### Generate App Password

1. Go to [App Passwords](https://myaccount.google.com/apppasswords)
2. Select app: "Mail"
3. Select device: "Other (Custom name)"
4. Enter name: "n8n Workflow"
5. Click "Generate"
6. **IMPORTANT**: Copy the 16-character password immediately
7. Save it securely - you won't be able to see it again

### Configure Gmail SMTP

The app password will be used in the workflow configuration:
- SMTP Server: smtp.gmail.com
- Port: 587 (TLS) or 465 (SSL)
- Username: your-email@gmail.com
- Password: [16-character app password]

## Step 3: Fusio API Gateway Setup

### Verify Fusio Installation

1. Check Fusio is running:
   ```bash
   curl http://your-fusio-url:8000/
   ```

2. Access Fusio backend:
   ```
   http://your-fusio-url:8000/apps/fusio
   ```

### Create Admin User for API Access

1. Log into Fusio backend as super admin
2. Navigate to "User" section
3. Click "Create"
4. Fill in:
   - Name: workflow-admin
   - Email: workflow@your-domain.com
   - Password: [secure password]
   - Status: Active
   - Scopes: Select all admin scopes

### Generate API Key

1. Navigate to "App" section
2. Click "Create"
3. Fill in:
   - Name: n8n Workflow
   - URL: http://localhost:5678
   - Scopes: consumer, consumer.user, backend.user

4. Save and note the generated:
   - App Key (API Key)
   - App Secret

### Enable Consumer Registration

1. Navigate to "Routes"
2. Find `/consumer/register`
3. Ensure it's active and public
4. Check rate limits if needed

## Step 4: n8n Workflow Setup

### Initial Setup

```bash
# Navigate to workflow directory
cd /home/septiannugraha/code/go-data-gateway/workflow

# Run the setup script
./scripts/setup.sh
```

The setup script will:
- Check prerequisites
- Create .env from template
- Generate encryption key
- Prompt for configuration
- Start Docker containers

### Manual Configuration

If you prefer manual setup:

1. **Copy environment template:**
   ```bash
   cp .env.example .env
   ```

2. **Edit .env file:**
   ```bash
   nano .env
   ```

3. **Set required variables:**
   ```env
   # Fusio Configuration
   FUSIO_API_URL=http://your-fusio-url:8000
   FUSIO_API_KEY=your-api-key-from-fusio
   FUSIO_ADMIN_USER=workflow-admin
   FUSIO_ADMIN_PASSWORD=your-admin-password

   # Gmail Configuration
   GMAIL_USER=your-email@gmail.com
   GMAIL_APP_PASSWORD=your-16-char-app-password

   # Security
   FORM_PASSPHRASE=spse2025
   N8N_ENCRYPTION_KEY=$(openssl rand -hex 32)
   ```

4. **Start services:**
   ```bash
   docker-compose up -d
   ```

### Import the Workflow

Option 1: Using import script
```bash
./scripts/import_workflow.sh
```

Option 2: Manual import via UI
1. Access n8n: http://localhost:5678
2. Click "Workflows" → "Add Workflow"
3. Click menu → "Import from File"
4. Select: `n8n/workflows/user_onboarding.json`

## Step 5: Connect FormLinker to n8n

### Get the Webhook URL

Your webhook URL will be:
```
http://your-server-ip:5678/webhook/google-forms-webhook
```

For local testing:
```
http://localhost:5678/webhook/google-forms-webhook
```

For production (with domain):
```
https://workflow.your-domain.com/webhook/google-forms-webhook
```

### Configure FormLinker

1. Open your Google Form
2. Click "Add-ons" → "FormLinker" → "Configure"
3. In FormLinker settings:
   - Webhook URL: [your webhook URL from above]
   - Method: POST
   - Content Type: application/json
   - Trigger: On Form Submit
4. Click "Save Configuration"
5. Click "Test Webhook" to verify connection

## Step 6: Configure n8n Credentials

### Gmail OAuth2 Setup (Recommended)

1. In n8n UI, go to "Credentials"
2. Click "Add Credential"
3. Search for "Gmail"
4. Select "Gmail OAuth2"
5. Follow the OAuth flow to authorize

### Alternative: SMTP Setup

1. In n8n UI, go to "Credentials"
2. Click "Add Credential"
3. Search for "Send Email"
4. Configure:
   - Host: smtp.gmail.com
   - Port: 587
   - User: your-email@gmail.com
   - Password: [app password]
   - SSL/TLS: Yes

## Step 7: Testing

### Test the Complete Flow

1. **Run the test script:**
   ```bash
   ./scripts/test.sh
   ```

2. **Submit a test form:**
   - Open your Google Form
   - Fill in test data
   - Use correct passphrase
   - Submit

3. **Check n8n executions:**
   - Open n8n UI
   - Go to "Executions"
   - Verify workflow ran successfully

4. **Check email delivery:**
   - Check the test email inbox
   - Verify credentials received

### Troubleshooting Tests

If tests fail, check:

1. **Webhook connectivity:**
   ```bash
   curl -X POST http://localhost:5678/webhook/google-forms-webhook \
     -H "Content-Type: application/json" \
     -d '{"test": true}'
   ```

2. **n8n logs:**
   ```bash
   docker-compose logs -f n8n
   ```

3. **PostgreSQL status:**
   ```bash
   docker exec n8n-postgres pg_isready
   ```

## Step 8: Production Deployment

### SSL/TLS Configuration

1. **Use reverse proxy (nginx):**
   ```nginx
   server {
       listen 443 ssl http2;
       server_name workflow.your-domain.com;

       ssl_certificate /path/to/cert.pem;
       ssl_certificate_key /path/to/key.pem;

       location / {
           proxy_pass http://localhost:5678;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection 'upgrade';
           proxy_set_header Host $host;
       }
   }
   ```

2. **Update webhook URL in FormLinker to HTTPS**

### Security Hardening

1. **Enable n8n authentication:**
   ```env
   N8N_BASIC_AUTH_ACTIVE=true
   N8N_BASIC_AUTH_USER=secure-username
   N8N_BASIC_AUTH_PASSWORD=secure-password
   ```

2. **Set production database:**
   ```env
   DB_PASSWORD=very-secure-password
   ```

3. **Restrict webhook access:**
   - Use webhook authentication
   - Implement IP whitelisting
   - Add rate limiting

### Monitoring

1. **Set up health checks:**
   ```bash
   # Add to monitoring system
   curl http://localhost:5678/healthz
   ```

2. **Configure alerting:**
   ```env
   ERROR_NOTIFICATION_EMAIL=ops-team@your-domain.com
   ```

3. **Enable execution logging:**
   ```env
   EXECUTIONS_DATA_SAVE_ON_ERROR=all
   EXECUTIONS_DATA_SAVE_ON_SUCCESS=all
   ```

## Maintenance

### Backup Procedures

1. **Backup n8n data:**
   ```bash
   docker exec n8n-postgres pg_dump -U n8n n8n > backup.sql
   ```

2. **Backup workflows:**
   ```bash
   cp -r n8n/workflows backups/
   ```

### Updates

1. **Update n8n:**
   ```bash
   docker-compose pull
   docker-compose up -d
   ```

2. **Update workflow:**
   - Export current workflow as backup
   - Import new version
   - Test thoroughly

## Common Issues and Solutions

### Issue: Webhook not receiving data
- Check FormLinker configuration
- Verify n8n is accessible from internet
- Check firewall rules

### Issue: Emails not sending
- Verify Gmail app password
- Check 2FA is enabled
- Review Gmail sending limits

### Issue: Fusio registration fails
- Verify API credentials
- Check user has correct permissions
- Ensure registration endpoint is enabled

### Issue: Password generation fails
- Check Node.js modules in container
- Verify crypto module availability
- Review function node logs

## Support

For additional help:
1. Check logs: `docker-compose logs -f`
2. Review n8n documentation: https://docs.n8n.io
3. Contact administrator: admin@spse-gateway.com