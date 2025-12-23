// RegStrava Frontend Application

// API Base URL
const API_BASE = window.location.origin;

// Signup Form Handler
document.addEventListener('DOMContentLoaded', function() {
    const signupForm = document.getElementById('signup-form');

    if (signupForm) {
        signupForm.addEventListener('submit', handleSignup);
    }
});

async function handleSignup(e) {
    e.preventDefault();

    const form = e.target;
    const submitBtn = form.querySelector('button[type="submit"]');
    const originalText = submitBtn.textContent;

    // Show loading state
    submitBtn.disabled = true;
    submitBtn.textContent = 'Creating Account...';

    // Clear any previous errors
    const existingAlert = form.querySelector('.alert');
    if (existingAlert) {
        existingAlert.remove();
    }

    const data = {
        name: form.name.value,
        email: form.email.value,
        company: form.company.value,
        track_fundings: form.track_fundings.checked
    };

    try {
        const response = await fetch(`${API_BASE}/api/v1/funders/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });

        const result = await response.json();

        if (!response.ok) {
            throw new Error(result.error || 'Failed to create account');
        }

        // Show success state with credentials
        showCredentials(result);

        // Store API key in localStorage for dashboard
        localStorage.setItem('regstrava_api_key', result.api_key);
        localStorage.setItem('regstrava_funder_id', result.funder_id);
        localStorage.setItem('regstrava_funder_name', result.name);

    } catch (error) {
        showError(form, error.message);
        submitBtn.disabled = false;
        submitBtn.textContent = originalText;
    }
}

function showCredentials(data) {
    const form = document.getElementById('signup-form');
    const successDiv = document.getElementById('signup-success');

    // Hide form, show success
    form.style.display = 'none';
    successDiv.style.display = 'block';

    // Populate credentials
    document.getElementById('api-key-value').textContent = data.api_key;
    document.getElementById('oauth-client-id').textContent = data.oauth_client_id;
    document.getElementById('oauth-secret').textContent = data.oauth_secret;
}

function showError(form, message) {
    const alert = document.createElement('div');
    alert.className = 'alert alert-error';
    alert.textContent = message;
    form.insertBefore(alert, form.firstChild);
}

function copyToClipboard(elementId) {
    const element = document.getElementById(elementId);
    const text = element.textContent;

    navigator.clipboard.writeText(text).then(() => {
        // Show feedback
        const btn = element.parentElement.querySelector('.copy-btn');
        const originalText = btn.textContent;
        btn.textContent = 'Copied!';
        setTimeout(() => {
            btn.textContent = originalText;
        }, 2000);
    }).catch(err => {
        console.error('Failed to copy:', err);
    });
}

// Dashboard Functions
async function loadDashboard() {
    const apiKey = localStorage.getItem('regstrava_api_key');

    if (!apiKey) {
        window.location.href = '/#signup';
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/api/v1/funders/me`, {
            headers: {
                'X-API-Key': apiKey
            }
        });

        if (!response.ok) {
            throw new Error('Failed to load profile');
        }

        const profile = await response.json();
        displayProfile(profile);

    } catch (error) {
        console.error('Dashboard error:', error);
        showDashboardError(error.message);
    }
}

function displayProfile(profile) {
    const nameEl = document.getElementById('funder-name');
    const idEl = document.getElementById('funder-id');
    const dailyLimitEl = document.getElementById('daily-limit');
    const monthlyLimitEl = document.getElementById('monthly-limit');
    const trackingEl = document.getElementById('tracking-status');

    if (nameEl) nameEl.textContent = profile.name;
    if (idEl) idEl.textContent = profile.id;
    if (dailyLimitEl) dailyLimitEl.textContent = profile.rate_limit_daily.toLocaleString();
    if (monthlyLimitEl) monthlyLimitEl.textContent = profile.rate_limit_monthly.toLocaleString();
    if (trackingEl) trackingEl.textContent = profile.track_fundings ? 'Enabled' : 'Disabled';
}

function showDashboardError(message) {
    const container = document.getElementById('dashboard-content');
    if (container) {
        container.innerHTML = `
            <div class="alert alert-error">
                ${message}. Please <a href="/#signup">sign up</a> or check your API key.
            </div>
        `;
    }
}

async function regenerateApiKey() {
    const apiKey = localStorage.getItem('regstrava_api_key');

    if (!confirm('Are you sure? Your current API key will stop working immediately.')) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/api/v1/funders/me/regenerate-api-key`, {
            method: 'POST',
            headers: {
                'X-API-Key': apiKey
            }
        });

        if (!response.ok) {
            throw new Error('Failed to regenerate API key');
        }

        const result = await response.json();

        // Update stored API key
        localStorage.setItem('regstrava_api_key', result.api_key);

        // Show new API key
        alert(`New API Key: ${result.api_key}\n\nSave this - it won't be shown again!`);

        // Refresh display
        document.getElementById('current-api-key').textContent = result.api_key.substring(0, 20) + '...';

    } catch (error) {
        alert('Error: ' + error.message);
    }
}

function logout() {
    localStorage.removeItem('regstrava_api_key');
    localStorage.removeItem('regstrava_funder_id');
    localStorage.removeItem('regstrava_funder_name');
    window.location.href = '/';
}

// Test API functionality
async function testCheckInvoice() {
    const apiKey = localStorage.getItem('regstrava_api_key');
    const resultDiv = document.getElementById('test-result');

    const testData = {
        invoice_number: 'TEST-' + Date.now(),
        issuer_tax_id: 'TEST123456',
        amount: 1000.00,
        currency: 'EUR'
    };

    try {
        resultDiv.innerHTML = '<p>Checking invoice...</p>';

        const response = await fetch(`${API_BASE}/api/v1/invoices/check-raw`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-API-Key': apiKey
            },
            body: JSON.stringify(testData)
        });

        const result = await response.json();

        resultDiv.innerHTML = `
            <pre>${JSON.stringify(result, null, 2)}</pre>
            <p class="text-success">API is working correctly!</p>
        `;

    } catch (error) {
        resultDiv.innerHTML = `<p class="text-error">Error: ${error.message}</p>`;
    }
}
