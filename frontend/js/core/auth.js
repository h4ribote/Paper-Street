import { api } from './api.js';
import { state, setApiKey } from './state.js';

export async function loginWithApiKey(apiKey) {
    setApiKey(apiKey);
    try {
        const user = await api('/api/users/me');
        if (user) {
            state.user = user;
            return true;
        }
        return false;
    } catch (e) {
        console.error("Login failed:", e);
        setApiKey('');
        return false;
    }
}

export async function loginBot(role, adminPassword) {
    try {
        const payload = await api('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ role, admin_password: adminPassword }),
        }, false);

        if (payload?.api_key) {
            setApiKey(payload.api_key);
            state.user = payload.user || null;
            return true;
        }
        return false;
    } catch (e) {
        console.error("Bot login failed:", e);
        return false;
    }
}

export function logout() {
    setApiKey('');
    state.user = null;
    window.location.href = '/index.html';
}

export async function requireAuth() {
    if (!state.apiKey) {
        window.location.href = '/login.html';
        return false;
    }
    if (!state.user) {
        try {
            state.user = await api('/api/users/me');
        } catch {
            window.location.href = '/login.html';
            return false;
        }
    }
    return true;
}
