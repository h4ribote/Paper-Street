export class RealtimeClient {
  constructor({ baseUrl, apiKey, onStatus, onMessage, onError }) {
    this.baseUrl = baseUrl;
    this.apiKey = apiKey;
    this.onStatus = onStatus || (() => {});
    this.onMessage = onMessage || (() => {});
    this.onError = onError || (() => {});
    this.socket = null;
    this.subscriptions = new Set();
    this.reconnectTimer = null;
    this.reconnectAttempts = 0;
    this.closedManually = false;
  }

  setConnection({ baseUrl, apiKey }) {
    if (baseUrl) this.baseUrl = baseUrl;
    if (apiKey) this.apiKey = apiKey;
  }

  connect() {
    this.closedManually = false;
    this.clearReconnect();
    if (!this.baseUrl || !this.apiKey) {
      this.onError(new Error('base URL and API key are required for WebSocket'));
      return;
    }

    const wsURL = this.toWebSocketURL(this.baseUrl);
    try {
      this.socket = new WebSocket(wsURL, []);
    } catch (error) {
      this.onError(error);
      this.scheduleReconnect();
      return;
    }

    this.onStatus('connecting');

    this.socket.addEventListener('open', () => {
      this.onStatus('connected');
      this.reconnectAttempts = 0;
      this.send({ op: 'ping' });
      this.resubscribeAll();
    });

    this.socket.addEventListener('message', (event) => {
      try {
        const payload = JSON.parse(event.data);
        this.onMessage(payload);
      } catch {
        this.onError(new Error('invalid websocket message payload'));
      }
    });

    this.socket.addEventListener('error', () => this.onStatus('error'));

    this.socket.addEventListener('close', () => {
      this.onStatus('disconnected');
      this.socket = null;
      if (!this.closedManually) this.scheduleReconnect();
    });
  }

  disconnect() {
    this.closedManually = true;
    this.clearReconnect();
    if (this.socket && this.socket.readyState <= WebSocket.OPEN) this.socket.close();
    this.socket = null;
    this.onStatus('disconnected');
  }

  subscribe(topics = []) {
    const normalized = topics.filter(Boolean);
    normalized.forEach((topic) => this.subscriptions.add(topic));
    this.send({ op: 'subscribe', args: normalized });
  }

  unsubscribe(topics = []) {
    const normalized = topics.filter(Boolean);
    normalized.forEach((topic) => this.subscriptions.delete(topic));
    this.send({ op: 'unsubscribe', args: normalized });
  }

  send(payload) {
    if (!payload) return;
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;
    this.socket.send(JSON.stringify(payload));
  }

  resubscribeAll() {
    if (this.subscriptions.size === 0) return;
    this.send({ op: 'subscribe', args: Array.from(this.subscriptions) });
  }

  scheduleReconnect() {
    this.clearReconnect();
    const backoffMs = Math.min(15000, 1000 * Math.pow(2, this.reconnectAttempts));
    this.reconnectAttempts += 1;
    this.reconnectTimer = setTimeout(() => this.connect(), backoffMs);
  }

  clearReconnect() {
    if (!this.reconnectTimer) return;
    clearTimeout(this.reconnectTimer);
    this.reconnectTimer = null;
  }

  toWebSocketURL(baseUrl) {
    const url = new URL(baseUrl);
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    url.pathname = '/ws';
    // Append api_key to the query string as required by the backend
    if (this.apiKey) {
      url.searchParams.set('api_key', this.apiKey);
    }
    return url.toString();
  }
}
