# Troubleshooting Guide - User Onboarding Workflow

Common issues and their solutions for the n8n user onboarding workflow.

## 🔍 Diagnostic Commands

### Quick Health Check
```bash
# Check all services status
docker-compose ps

# Check n8n health
curl http://localhost:5678/healthz

# Check PostgreSQL
docker exec n8n-postgres pg_isready

# View recent logs
docker-compose logs --tail=50

# Follow live logs
docker-compose logs -f n8n
```

## 🚨 Common Issues

### 1. n8n Won't Start

**Symptoms:**
- Container exits immediately
- Port 5678 not accessible
- Health check fails

**Solutions:**

1. **Check port conflicts:**
   ```bash
   # Check if port is in use
   sudo lsof -i :5678

   # Kill process using port
   sudo kill -9 $(sudo lsof -t -i:5678)
   ```

2. **Check Docker resources:**
   ```bash
   # Check Docker status
   docker system df

   # Clean up if needed
   docker system prune -a
   ```

3. **Reset n8n:**
   ```bash
   docker-compose down -v
   docker-compose up -d
   ```

### 2. Webhook Not Receiving Data

**Symptoms:**
- Form submissions not triggering workflow
- No executions in n8n
- FormLinker shows errors

**Solutions:**

1. **Test webhook directly:**
   ```bash
   curl -X POST http://localhost:5678/webhook/google-forms-webhook \
     -H "Content-Type: application/json" \
     -d '{"email":"test@example.com","name":"Test","passphrase":"spse2025"}'
   ```

2. **Check FormLinker configuration:**
   - Verify webhook URL is correct
   - Ensure "On Form Submit" is selected
   - Test webhook from FormLinker interface

3. **Network accessibility:**
   ```bash
   # If using local development
   # Consider using ngrok for public URL
   ngrok http 5678
   ```

4. **Check n8n webhook logs:**
   ```bash
   docker logs n8n 2>&1 | grep webhook
   ```

### 3. Email Not Sending

**Symptoms:**
- Workflow executes but email not received
- Gmail node shows error
- Authentication failures

**Solutions:**

1. **Verify Gmail configuration:**
   ```bash
   # Check environment variables
   grep GMAIL .env
   ```

2. **Test Gmail authentication:**
   ```python
   # Test SMTP connection
   import smtplib
   server = smtplib.SMTP('smtp.gmail.com', 587)
   server.starttls()
   server.login('your-email@gmail.com', 'app-password')
   server.quit()
   print("Success!")
   ```

3. **Common Gmail issues:**
   - App password expired - regenerate
   - 2FA not enabled - enable it first
   - Daily sending limit reached (500/day)
   - Less secure apps blocked - use app password

4. **Check Gmail credentials in n8n:**
   - Go to Credentials in n8n UI
   - Test the Gmail credential
   - Re-authenticate if needed

### 4. Fusio Registration Fails

**Symptoms:**
- HTTP 401/403 errors
- User not created in Fusio
- "Invalid credentials" error

**Solutions:**

1. **Test Fusio API directly:**
   ```bash
   # Test authentication
   curl -X POST http://fusio-url:8000/consumer/login \
     -H "Content-Type: application/json" \
     -d '{"username":"admin","password":"admin-password"}'
   ```

2. **Verify API configuration:**
   ```bash
   # Check environment
   grep FUSIO .env
   ```

3. **Check Fusio permissions:**
   - Login to Fusio backend
   - Verify user has consumer.register scope
   - Check API key is active

4. **Test registration endpoint:**
   ```bash
   curl -X POST http://fusio-url:8000/consumer/register \
     -H "Content-Type: application/json" \
     -H "X-API-Key: your-api-key" \
     -d '{"name":"Test","email":"test@example.com","password":"Test123!"}'
   ```

### 5. Password Generation Fails

**Symptoms:**
- Function node error
- "Cannot find module" error
- Empty password field

**Solutions:**

1. **Check module availability:**
   ```bash
   # Enter n8n container
   docker exec -it n8n sh

   # Check if crypto is available
   node -e "console.log(require('crypto').randomBytes(16).toString('hex'))"
   ```

2. **Copy custom modules:**
   ```bash
   # Copy password generator to n8n
   docker cp services/password_generator.js n8n:/home/node/.n8n/custom/
   ```

3. **Alternative: Use Code node instead:**
   ```javascript
   // Inline password generation
   const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*';
   let password = '';
   for (let i = 0; i < 16; i++) {
     password += chars.charAt(Math.floor(Math.random() * chars.length));
   }
   return [{json: {password: password}}];
   ```

### 6. Workflow Not Activating

**Symptoms:**
- Workflow imported but not running
- Shows as inactive in UI
- Webhook returns 404

**Solutions:**

1. **Manually activate:**
   - Open n8n UI
   - Go to Workflows
   - Toggle the activation switch

2. **Activate via API:**
   ```bash
   # Get workflow ID
   curl http://localhost:5678/rest/workflows

   # Activate workflow
   curl -X PATCH http://localhost:5678/rest/workflows/[workflow-id] \
     -H "Content-Type: application/json" \
     -d '{"active": true}'
   ```

### 7. Database Connection Issues

**Symptoms:**
- n8n crashes on startup
- "Connection refused" errors
- Data not persisting

**Solutions:**

1. **Check PostgreSQL:**
   ```bash
   # Check if running
   docker-compose ps n8n-postgres

   # Check logs
   docker-compose logs n8n-postgres

   # Test connection
   docker exec -it n8n-postgres psql -U n8n -c "SELECT 1"
   ```

2. **Reset database:**
   ```bash
   # Backup first if needed
   docker exec n8n-postgres pg_dump -U n8n n8n > backup.sql

   # Reset
   docker-compose down -v
   docker-compose up -d
   ```

### 8. Performance Issues

**Symptoms:**
- Slow workflow execution
- Timeouts
- High memory usage

**Solutions:**

1. **Check resource usage:**
   ```bash
   # Container stats
   docker stats

   # n8n specific
   docker exec n8n top
   ```

2. **Optimize workflow:**
   - Reduce node complexity
   - Use webhook response node early
   - Implement pagination for large datasets

3. **Scale resources:**
   ```yaml
   # In docker-compose.yml
   services:
     n8n:
       deploy:
         resources:
           limits:
             cpus: '2'
             memory: 2G
   ```

## 📊 Monitoring & Logging

### Enable Detailed Logging

1. **Set log level:**
   ```env
   # In .env
   N8N_LOG_LEVEL=debug
   ```

2. **View execution details:**
   ```bash
   # Get execution IDs
   curl http://localhost:5678/rest/executions

   # Get specific execution
   curl http://localhost:5678/rest/executions/[id]
   ```

### Set Up Monitoring

1. **Health check endpoint:**
   ```bash
   # Add to monitoring tool
   curl http://localhost:5678/healthz
   ```

2. **Metrics collection:**
   ```bash
   # Execution stats
   docker exec n8n-postgres psql -U n8n -c \
     "SELECT COUNT(*), success FROM execution_entity GROUP BY success"
   ```

## 🔧 Advanced Debugging

### Enable n8n Debug Mode

```env
# In .env
N8N_LOG_LEVEL=debug
N8N_LOG_OUTPUT=console
```

### Inspect Webhook Payload

```javascript
// Add debug node in workflow
console.log('Received data:', JSON.stringify($input.all(), null, 2));
return $input.all();
```

### Database Queries

```sql
-- Check recent executions
SELECT id, workflow_id, finished, mode, status
FROM execution_entity
ORDER BY started_at DESC
LIMIT 10;

-- Check workflow status
SELECT id, name, active, created_at
FROM workflow_entity;

-- Check credentials
SELECT id, name, type
FROM credentials_entity;
```

## 🆘 Getting Help

### Collect Diagnostic Information

```bash
# Generate diagnostic report
cat > diagnostic.txt << EOF
=== System Information ===
$(uname -a)
$(docker --version)
$(docker-compose --version)

=== Container Status ===
$(docker-compose ps)

=== Recent Logs ===
$(docker-compose logs --tail=100)

=== Environment Check ===
$(grep -E "FUSIO|GMAIL|N8N" .env | sed 's/PASSWORD=.*/PASSWORD=***/')

=== Network Test ===
$(curl -s http://localhost:5678/healthz)
EOF
```

### Resources

- **n8n Community**: https://community.n8n.io
- **n8n Documentation**: https://docs.n8n.io
- **Docker Logs**: `docker-compose logs -f`
- **Support Email**: admin@spse-gateway.com

## 🔄 Recovery Procedures

### Complete Reset

```bash
# Backup data
docker exec n8n-postgres pg_dump -U n8n n8n > backup.sql

# Stop everything
docker-compose down

# Clean up
docker system prune -a
rm -rf n8n_data n8n_postgres_data

# Restart
docker-compose up -d

# Restore if needed
docker exec -i n8n-postgres psql -U n8n n8n < backup.sql
```

### Workflow Recovery

```bash
# Export workflow
curl http://localhost:5678/rest/workflows/[id] > workflow-backup.json

# Import workflow
curl -X POST http://localhost:5678/rest/workflows \
  -H "Content-Type: application/json" \
  -d @workflow-backup.json
```

## 🎯 Prevention Tips

1. **Regular backups:**
   ```bash
   # Add to crontab
   0 2 * * * docker exec n8n-postgres pg_dump -U n8n n8n > /backup/n8n-$(date +\%Y\%m\%d).sql
   ```

2. **Monitor disk space:**
   ```bash
   df -h /var/lib/docker
   ```

3. **Update regularly:**
   ```bash
   docker-compose pull
   docker-compose up -d
   ```

4. **Test changes in staging first**

5. **Keep credentials secure and rotated**

Remember: Most issues can be resolved by checking logs first!