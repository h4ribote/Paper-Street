import { state } from './state.js';

export function headers(auth = true) {
    const h = { 'Content-Type': 'application/json' };
    if (auth && state.apiKey) h['X-API-Key'] = state.apiKey;
    return h;
}

export function ensureBaseUrl() {
    let v = state.baseUrl || `${location.protocol}//${location.hostname}:8000`;
    if (!/^https?:\/\//i.test(v)) v = `http://${v}`;
    state.baseUrl = v;
    localStorage.setItem('paperstreet.baseUrl', v);
    return v;
}

export async function api(path, options = {}, auth = true) {
    const url = new URL(path, ensureBaseUrl());
    const response = await fetch(url.toString(), {
        ...options,
        headers: { ...headers(auth), ...(options.headers || {}) },
    });

    if (!response.ok) {
        let msg = `${response.status} ${response.statusText}`;
        try {
            const err = await response.json();
            if (err?.error) msg = err.error;
        } catch {}
        throw new Error(msg);
    }

    const contentType = response.headers.get('content-type') || '';
    if (!contentType.includes('application/json')) return null;
    return response.json();
}
