import { loadHeader } from '../components/header.js';
import { requireAuth, loginWithApiKey, loginBot } from '../core/auth.js';

document.addEventListener('DOMContentLoaded', () => {
    // Note: login page doesn't strictly require auth to view, but handles redirect if already auth'd
    const discordBtn = document.getElementById('btn-discord-login');
    const apiLoginForm = document.getElementById('api-login-form');
    const errorEl = document.getElementById('login-error');

    discordBtn.addEventListener('click', () => {
        errorEl.textContent = 'Discord login is currently disabled.';
        errorEl.classList.remove('hidden');
    });

    apiLoginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        errorEl.classList.add('hidden');

        const apiKey = document.getElementById('api-key').value;
        const adminPw = document.getElementById('admin-password').value;
        const botRole = document.getElementById('bot-role').value;

        let success = false;

        if (adminPw && botRole) {
            success = await loginBot(botRole, adminPw);
        } else if (apiKey) {
            success = await loginWithApiKey(apiKey);
        } else {
            errorEl.textContent = 'Please provide API Key or Bot credentials.';
            errorEl.classList.remove('hidden');
            return;
        }

        if (success) {
            window.location.href = '/dashboard.html';
        } else {
            errorEl.textContent = 'Login failed. Check credentials.';
            errorEl.classList.remove('hidden');
        }
    });
});
