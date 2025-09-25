#!/bin/bash

# n8n Status Check Script
echo "======================================"
echo "🚀 n8n Status Report"
echo "======================================"
echo ""

# Check Docker container
echo "📦 Docker Container Status:"
docker ps | grep n8n && echo "✅ Container running" || echo "❌ Container not running"
echo ""

# Check health endpoint
echo "💓 Health Check:"
if curl -s http://localhost:5678/healthz | grep -q "ok"; then
    echo "✅ n8n is healthy"
else
    echo "❌ n8n health check failed"
fi
echo ""

# Check UI accessibility
echo "🌐 Web Interface:"
if curl -s -o /dev/null -w "%{http_code}" http://localhost:5678 | grep -q "200"; then
    echo "✅ UI accessible at: http://localhost:5678"
else
    echo "❌ UI not accessible"
fi
echo ""

# Test webhook endpoint (will fail if no webhook exists)
echo "🔗 Webhook Test:"
response=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:5678/webhook/google-forms-webhook \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}')

if [ "$response" = "404" ]; then
    echo "⚠️  No webhook configured yet (expected)"
    echo "   You need to import the workflow in n8n UI"
elif [ "$response" = "200" ]; then
    echo "✅ Webhook is configured and working!"
else
    echo "❔ Webhook returned status: $response"
fi
echo ""

# Instructions
echo "======================================"
echo "📝 Next Steps:"
echo "======================================"
echo "1. Open browser to: http://localhost:5678"
echo "2. Click 'New Workflow' or import existing"
echo "3. To import our workflow:"
echo "   - Click menu (⋮) → 'Import from File'"
echo "   - Select: workflow/n8n/workflows/user_onboarding.json"
echo "4. Activate the workflow (toggle switch)"
echo "5. Test with:"
echo "   curl -X POST http://localhost:5678/webhook/google-forms-webhook \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"email\":\"test@example.com\",\"name\":\"Test\",\"passphrase\":\"spse2025\"}'"
echo ""
echo "======================================"
echo "✨ n8n is ready for workflow automation!"
echo "======================================"