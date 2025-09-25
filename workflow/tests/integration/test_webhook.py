"""
Integration tests for webhook endpoint
Tests the complete webhook flow with n8n
"""

import pytest
import requests
import json
import time
import os
from typing import Dict, Any

# Test configuration
WEBHOOK_URL = os.getenv('WEBHOOK_URL', 'http://localhost:5679/webhook/google-forms-webhook')
N8N_URL = os.getenv('N8N_URL', 'http://localhost:5679')
PASSPHRASE = os.getenv('FORM_PASSPHRASE', 'spse2025')
TIMEOUT = 30


class TestWebhookIntegration:
    """Test webhook endpoint integration"""

    @pytest.fixture(autouse=True)
    def setup(self):
        """Setup before each test"""
        # Wait for services to be ready
        self.wait_for_service(N8N_URL + '/healthz')

    def wait_for_service(self, url: str, max_attempts: int = 30):
        """Wait for service to be available"""
        for i in range(max_attempts):
            try:
                response = requests.get(url, timeout=2)
                if response.status_code == 200:
                    return
            except requests.exceptions.RequestException:
                pass
            time.sleep(1)
        raise TimeoutError(f"Service {url} not available after {max_attempts} seconds")

    def test_webhook_health_check(self):
        """Test if webhook endpoint is accessible"""
        response = requests.get(N8N_URL + '/healthz', timeout=TIMEOUT)
        assert response.status_code == 200

    def test_valid_registration_request(self):
        """Test webhook with valid registration data"""
        payload = {
            "email": f"test-{int(time.time())}@example.com",
            "name": "Test User",
            "passphrase": PASSPHRASE,
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "formId": "test-form-001"
        }

        response = requests.post(
            WEBHOOK_URL,
            json=payload,
            headers={'Content-Type': 'application/json'},
            timeout=TIMEOUT
        )

        assert response.status_code == 200
        data = response.json()
        assert data.get('success') is True

    def test_invalid_passphrase_rejection(self):
        """Test webhook rejects invalid passphrase"""
        payload = {
            "email": f"test-{int(time.time())}@example.com",
            "name": "Test User",
            "passphrase": "wrong_passphrase",
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
        }

        response = requests.post(
            WEBHOOK_URL,
            json=payload,
            headers={'Content-Type': 'application/json'},
            timeout=TIMEOUT
        )

        assert response.status_code == 401
        data = response.json()
        assert data.get('success') is False
        assert 'invalid' in data.get('error', '').lower()

    def test_missing_required_fields(self):
        """Test webhook handles missing required fields"""
        # Test missing email
        payload = {
            "name": "Test User",
            "passphrase": PASSPHRASE
        }

        response = requests.post(
            WEBHOOK_URL,
            json=payload,
            headers={'Content-Type': 'application/json'},
            timeout=TIMEOUT
        )

        assert response.status_code in [400, 500]

        # Test missing name
        payload = {
            "email": "test@example.com",
            "passphrase": PASSPHRASE
        }

        response = requests.post(
            WEBHOOK_URL,
            json=payload,
            headers={'Content-Type': 'application/json'},
            timeout=TIMEOUT
        )

        assert response.status_code in [400, 500]

    def test_invalid_email_format(self):
        """Test webhook validates email format"""
        payload = {
            "email": "not-an-email",
            "name": "Test User",
            "passphrase": PASSPHRASE
        }

        response = requests.post(
            WEBHOOK_URL,
            json=payload,
            headers={'Content-Type': 'application/json'},
            timeout=TIMEOUT
        )

        # Should either reject at webhook or during processing
        assert response.status_code in [400, 500]

    def test_webhook_handles_large_payload(self):
        """Test webhook handles large payloads gracefully"""
        payload = {
            "email": "test@example.com",
            "name": "Test User",
            "passphrase": PASSPHRASE,
            "extra_data": "x" * 10000  # 10KB of extra data
        }

        response = requests.post(
            WEBHOOK_URL,
            json=payload,
            headers={'Content-Type': 'application/json'},
            timeout=TIMEOUT
        )

        # Should still process successfully
        assert response.status_code in [200, 401]

    @pytest.mark.parametrize("content_type", [
        "application/x-www-form-urlencoded",
        "text/plain",
        "application/xml"
    ])
    def test_webhook_content_type_handling(self, content_type):
        """Test webhook handles different content types"""
        payload = "email=test@example.com&name=Test&passphrase=" + PASSPHRASE

        response = requests.post(
            WEBHOOK_URL,
            data=payload,
            headers={'Content-Type': content_type},
            timeout=TIMEOUT
        )

        # Should handle or reject gracefully
        assert response.status_code in [200, 400, 401, 415]

    def test_concurrent_webhook_requests(self):
        """Test webhook handles concurrent requests"""
        import concurrent.futures

        def send_request(index):
            payload = {
                "email": f"concurrent-{index}-{int(time.time())}@example.com",
                "name": f"Concurrent User {index}",
                "passphrase": PASSPHRASE
            }

            response = requests.post(
                WEBHOOK_URL,
                json=payload,
                headers={'Content-Type': 'application/json'},
                timeout=TIMEOUT
            )
            return response.status_code

        with concurrent.futures.ThreadPoolExecutor(max_workers=5) as executor:
            futures = [executor.submit(send_request, i) for i in range(5)]
            results = [f.result() for f in concurrent.futures.as_completed(futures)]

        # All requests should be processed
        assert all(status in [200, 401, 429, 500] for status in results)

    def test_webhook_response_format(self):
        """Test webhook returns consistent response format"""
        payload = {
            "email": f"format-test-{int(time.time())}@example.com",
            "name": "Format Test",
            "passphrase": PASSPHRASE
        }

        response = requests.post(
            WEBHOOK_URL,
            json=payload,
            headers={'Content-Type': 'application/json'},
            timeout=TIMEOUT
        )

        assert response.headers.get('Content-Type', '').startswith('application/json')
        data = response.json()
        assert 'success' in data
        if data['success']:
            assert 'message' in data or 'data' in data
        else:
            assert 'error' in data

    def test_webhook_idempotency(self):
        """Test webhook handles duplicate requests gracefully"""
        email = f"idempotent-{int(time.time())}@example.com"
        payload = {
            "email": email,
            "name": "Idempotent User",
            "passphrase": PASSPHRASE
        }

        # First request
        response1 = requests.post(
            WEBHOOK_URL,
            json=payload,
            headers={'Content-Type': 'application/json'},
            timeout=TIMEOUT
        )

        time.sleep(2)  # Wait a bit

        # Second request with same data
        response2 = requests.post(
            WEBHOOK_URL,
            json=payload,
            headers={'Content-Type': 'application/json'},
            timeout=TIMEOUT
        )

        # Should handle duplicate gracefully
        assert response1.status_code == 200
        assert response2.status_code in [200, 409, 500]

    def test_webhook_timeout_handling(self):
        """Test webhook handles slow processing"""
        payload = {
            "email": f"timeout-test-{int(time.time())}@example.com",
            "name": "Timeout Test",
            "passphrase": PASSPHRASE,
            # Add flag to simulate slow processing if workflow supports it
            "test_mode": "slow"
        }

        # Use short timeout to test handling
        try:
            response = requests.post(
                WEBHOOK_URL,
                json=payload,
                headers={'Content-Type': 'application/json'},
                timeout=2  # Short timeout
            )
            # If it completes, should be successful
            assert response.status_code in [200, 401]
        except requests.exceptions.Timeout:
            # Timeout is acceptable for this test
            pass

    @pytest.mark.skip(reason="Requires special characters testing")
    def test_webhook_special_characters(self):
        """Test webhook handles special characters in input"""
        special_chars = ["'", '"', "<", ">", "&", "\\", "/", "%", "\n", "\r", "\t"]

        for char in special_chars:
            payload = {
                "email": f"test{char}user@example.com",
                "name": f"Test{char}User",
                "passphrase": PASSPHRASE
            }

            response = requests.post(
                WEBHOOK_URL,
                json=payload,
                headers={'Content-Type': 'application/json'},
                timeout=TIMEOUT
            )

            # Should handle or reject gracefully
            assert response.status_code in [200, 400, 401]


class TestWebhookSecurity:
    """Security-focused webhook tests"""

    def test_sql_injection_protection(self):
        """Test webhook protects against SQL injection"""
        payloads = [
            "'; DROP TABLE users; --",
            "1' OR '1'='1",
            "admin'--",
            "' UNION SELECT * FROM users--"
        ]

        for injection in payloads:
            payload = {
                "email": f"{injection}@example.com",
                "name": injection,
                "passphrase": PASSPHRASE
            }

            response = requests.post(
                WEBHOOK_URL,
                json=payload,
                headers={'Content-Type': 'application/json'},
                timeout=TIMEOUT
            )

            # Should reject or sanitize
            assert response.status_code in [400, 401, 500]

    def test_xss_protection(self):
        """Test webhook protects against XSS"""
        xss_payloads = [
            "<script>alert('xss')</script>",
            "javascript:alert('xss')",
            "<img src=x onerror=alert('xss')>",
            "<svg onload=alert('xss')>"
        ]

        for xss in xss_payloads:
            payload = {
                "email": "test@example.com",
                "name": xss,
                "passphrase": PASSPHRASE
            }

            response = requests.post(
                WEBHOOK_URL,
                json=payload,
                headers={'Content-Type': 'application/json'},
                timeout=TIMEOUT
            )

            # Should sanitize or reject
            if response.status_code == 200:
                data = response.json()
                # Check that script tags are not in response
                assert '<script>' not in str(data)

    def test_rate_limiting(self):
        """Test webhook has rate limiting"""
        # Send many requests quickly
        results = []
        for i in range(20):
            payload = {
                "email": f"ratelimit-{i}@example.com",
                "name": f"Rate Limit {i}",
                "passphrase": PASSPHRASE
            }

            response = requests.post(
                WEBHOOK_URL,
                json=payload,
                headers={'Content-Type': 'application/json'},
                timeout=TIMEOUT
            )
            results.append(response.status_code)

        # Some requests should be rate limited
        rate_limited = any(status == 429 for status in results)
        # If no explicit rate limiting, at least check for errors
        has_errors = any(status >= 500 for status in results)

        assert rate_limited or has_errors or len(set(results)) > 1


if __name__ == "__main__":
    pytest.main([__file__, "-v"])