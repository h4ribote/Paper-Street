import { PriceChart } from './chart.js';
import { RealtimeClient } from './ws.js';

const state = {
  baseUrl: localStorage.getItem('paperstreet.baseUrl') || `${location.protocol}//${location.hostname}:8000`,
  apiKey: '',
  user: null,
  assets: [],
  assetsById: new Map(),
  tickers: [],
  selectedAssetId: null,
  orderbook: { bids: [], asks: [] },
  trades: [],
  orders: [],
  balances: [],
  portfolioAssets: [],
  marginPositions: [],
  news: [],
  wsClient: null,
  chart: null,
};

const el = {
  baseUrl: document.getElementById('api-base-url'),
  apiKey: document.getElementById('api-key'),
  botRole: document.getElementById('bot-role'),
  adminPassword: document.getElementById('admin-password'),
  httpStatus: document.getElementById('http-status'),
  wsStatus: document.getElementById('ws-status'),
  selectedAssetLabel: document.getElementById('selected-asset-label'),
  marketWatchBody: document.getElementById('market-watch-body'),
  orderAsset: document.getElementById('order-asset'),
  orderForm: document.getElementById('order-form'),
  orderType: document.getElementById('order-type'),
  orderPrice: document.getElementById('order-price'),
  orderStopPrice: document.getElementById('order-stop-price'),
  orderbookBids: document.getElementById('orderbook-bids'),
  orderbookAsks: document.getElementById('orderbook-asks'),
  tradesBody: document.getElementById('trades-body'),
  ordersBody: document.getElementById('orders-body'),
  userCard: document.getElementById('user-card'),
  balancesList: document.getElementById('balances-list'),
  assetsList: document.getElementById('assets-list'),
  positionsList: document.getElementById('positions-list'),
  newsList: document.getElementById('news-list'),
  logOutput: document.getElementById('log-output'),
  btnLogin: document.getElementById('btn-login'),
  btnConnect: document.getElementById('btn-connect'),
  btnDisconnect: document.getElementById('btn-disconnect'),
  btnRefresh: document.getElementById('btn-refresh'),
};

function log(message, level = 'info') {
  const prefix = new Date().toLocaleTimeString();
  const line = `[${prefix}] ${String(message)}`;
  el.logOutput.textContent = `${line}\n${el.logOutput.textContent}`.slice(0, 10000);
  if (level === 'error') console.error(message);
}

function setHttpStatus(text, cls = '') {
  el.httpStatus.textContent = text;
  el.httpStatus.className = cls;
}

function setWsStatus(text, cls = '') {
  el.wsStatus.textContent = text;
  el.wsStatus.className = cls;
}

function headers(auth = true) {
  const h = { 'Content-Type': 'application/json' };
  if (auth && state.apiKey) h['X-API-Key'] = state.apiKey;
  return h;
}

async function api(path, options = {}, auth = true) {
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

function ensureBaseUrl() {
  let v = (el.baseUrl.value || '').trim();
  if (!v) v = state.baseUrl;
  if (!v) throw new Error('API Base URL is required');
  if (!/^https?:\/\//i.test(v)) v = `http://${v}`;
  state.baseUrl = v;
  localStorage.setItem('paperstreet.baseUrl', v);
  return v;
}

function updateApiKey(value) {
  const trimmed = String(value || '').trim();
  state.apiKey = trimmed;
  el.apiKey.value = trimmed;
}

function fmtNumber(v) {
  if (v == null || Number.isNaN(Number(v))) return '-';
  return Number(v).toLocaleString();
}

function fmtTime(v) {
  if (!v) return '-';
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return '-';
  return d.toLocaleTimeString();
}

function sideByTrade(t) {
  if (!t) return '-';
  if (Number(t.taker_user_id) === Number(state.user?.id || -1)) return 'TAKER';
  if (Number(t.maker_user_id) === Number(state.user?.id || -1)) return 'MAKER';
  return '-';
}

function selectedAsset() {
  return state.assetsById.get(Number(state.selectedAssetId)) || null;
}

function selectedAssetSymbol() {
  const asset = selectedAsset();
  return asset ? `${asset.symbol} (#${asset.id})` : '-';
}

function setSelectedAsset(assetId) {
  state.selectedAssetId = Number(assetId) || null;
  el.orderAsset.value = String(state.selectedAssetId || '');
  el.selectedAssetLabel.textContent = selectedAssetSymbol();
  renderMarketWatch();
}

function appendCell(row, value, className = '') {
  const td = document.createElement('td');
  td.textContent = value == null ? '-' : String(value);
  if (className) td.className = className;
  row.appendChild(td);
}

function makeLi(text) {
  const li = document.createElement('li');
  li.textContent = text;
  return li;
}

function renderMarketWatch() {
  const rows = [];
  for (const ticker of state.tickers) {
    const tr = document.createElement('tr');
    const selected = Number(ticker.asset_id) === Number(state.selectedAssetId);
    if (selected) tr.classList.add('selected');
    tr.addEventListener('click', async () => {
      setSelectedAsset(ticker.asset_id);
      await loadAssetDetail();
      subscribeAssetTopics();
    });

    const changeClass = Number(ticker.change) >= 0 ? 'good' : 'bad';
    appendCell(tr, ticker.symbol);
    appendCell(tr, fmtNumber(ticker.price));
    appendCell(tr, fmtNumber(ticker.change), changeClass);
    appendCell(tr, fmtNumber(ticker.volume));
    rows.push(tr);
  }
  el.marketWatchBody.replaceChildren(...rows);
}

function renderOrderAssetSelect() {
  const nodes = state.assets.map((a) => {
    const option = document.createElement('option');
    option.value = String(a.id);
    option.textContent = `${a.symbol} (#${a.id})`;
    return option;
  });
  el.orderAsset.replaceChildren(...nodes);
  if (!state.selectedAssetId && state.assets.length > 0) {
    setSelectedAsset(state.assets[0].id);
  }
}

function renderOrderbook() {
  const bidRows = (state.orderbook.bids || []).map((lv) => {
    const tr = document.createElement('tr');
    appendCell(tr, fmtNumber(lv.price), 'good');
    appendCell(tr, fmtNumber(lv.quantity));
    return tr;
  });
  const askRows = (state.orderbook.asks || []).map((lv) => {
    const tr = document.createElement('tr');
    appendCell(tr, fmtNumber(lv.price), 'bad');
    appendCell(tr, fmtNumber(lv.quantity));
    return tr;
  });
  el.orderbookBids.replaceChildren(...bidRows);
  el.orderbookAsks.replaceChildren(...askRows);
}

function renderTrades() {
  const rows = (state.trades || []).slice(0, 100).map((trade) => {
    const tr = document.createElement('tr');
    appendCell(tr, fmtTime(trade.occurred_at));
    appendCell(tr, fmtNumber(trade.price));
    appendCell(tr, fmtNumber(trade.quantity));
    appendCell(tr, sideByTrade(trade));
    return tr;
  });
  el.tradesBody.replaceChildren(...rows);
}

function renderOrders() {
  const rows = (state.orders || []).map((o) => {
    const tr = document.createElement('tr');
    appendCell(tr, String(o.id));
    appendCell(tr, String(o.asset_id));
    appendCell(tr, o.side || '-');
    appendCell(tr, o.type || '-');
    appendCell(tr, fmtNumber(o.quantity));
    appendCell(tr, fmtNumber(o.remaining));
    appendCell(tr, fmtNumber(o.price || 0));
    appendCell(tr, o.status || '-');

    const actionTd = document.createElement('td');
    if (o.status === 'OPEN' || o.status === 'PARTIAL') {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'secondary';
      btn.textContent = 'Cancel';
      btn.addEventListener('click', () => cancelOrder(o));
      actionTd.appendChild(btn);
    } else {
      actionTd.textContent = '-';
    }
    tr.appendChild(actionTd);
    return tr;
  });
  el.ordersBody.replaceChildren(...rows);
}

function renderPortfolio() {
  el.userCard.textContent = state.user ? JSON.stringify(state.user, null, 2) : 'Not loaded';

  const balances = (state.balances || []).map((b) => makeLi(`${b.currency}: ${fmtNumber(b.amount)}`));
  el.balancesList.replaceChildren(...balances);

  const assets = (state.portfolioAssets || []).map((pa) => makeLi(`${pa.asset?.symbol || pa.asset?.id || '-'}: ${fmtNumber(pa.quantity)}`));
  el.assetsList.replaceChildren(...assets);

  const positions = (state.marginPositions || []).map((p) => {
    const side = p.side || '-';
    const pnl = -Number(p.unrealized_loss || 0);
    return makeLi(`#${p.id} ${side} ${p.asset_id} x${fmtNumber(p.quantity)} P&L:${fmtNumber(pnl)}`);
  });
  el.positionsList.replaceChildren(...positions);
}

function renderNews() {
  const nodes = (state.news || []).slice(0, 60).map((n) => {
    const li = document.createElement('li');
    const time = fmtTime(n.published_at);
    li.textContent = `${time} | ${n.headline}`;
    return li;
  });
  el.newsList.replaceChildren(...nodes);
}

async function loadBootstrap() {
  setHttpStatus('loading', 'warn');
  const [assets, tickers, news] = await Promise.all([
    api('/assets'), api('/market/ticker'), api('/news?limit=50'),
  ]);
  state.assets = Array.isArray(assets) ? assets : [];
  state.assetsById = new Map(state.assets.map((a) => [Number(a.id), a]));
  state.tickers = Array.isArray(tickers) ? tickers : [];
  state.news = Array.isArray(news) ? news : [];

  renderOrderAssetSelect();
  renderMarketWatch();
  renderNews();
  setHttpStatus('ok', 'good');
}

async function loadUserAndPortfolio() {
  try {
    const [user, balances, assets, positions, orders] = await Promise.all([
      api('/users/me'),
      api('/portfolio/balances'),
      api('/portfolio/assets'),
      api('/margin/positions'),
      api('/orders?status=OPEN&limit=200'),
    ]);
    state.user = user || null;
    state.balances = Array.isArray(balances) ? balances : [];
    state.portfolioAssets = Array.isArray(assets) ? assets : [];
    state.marginPositions = Array.isArray(positions) ? positions : [];
    state.orders = Array.isArray(orders) ? orders : [];
    renderPortfolio();
    renderOrders();
  } catch (error) {
    log(`portfolio load failed: ${error.message}`, 'error');
  }
}

async function loadAssetDetail() {
  if (!state.selectedAssetId) return;
  try {
    const [orderbook, trades, candles] = await Promise.all([
      api(`/market/orderbook/${state.selectedAssetId}?depth=20`),
      api(`/market/trades/${state.selectedAssetId}?limit=100`),
      api(`/market/candles/${state.selectedAssetId}?timeframe=1m&limit=120`),
    ]);

    state.orderbook = orderbook || { bids: [], asks: [] };
    state.trades = Array.isArray(trades) ? trades : [];
    renderOrderbook();
    renderTrades();
    state.chart?.setCandles(Array.isArray(candles) ? candles : []);
  } catch (error) {
    log(`asset detail load failed: ${error.message}`, 'error');
  }
}

async function loginBot() {
  const role = (el.botRole.value || '').trim();
  const adminPassword = (el.adminPassword.value || '').trim();
  if (!role || !adminPassword) {
    log('role and admin password are required for /auth/login');
    return;
  }
  try {
    const payload = await api('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ role, admin_password: adminPassword }),
    }, false);

    if (payload?.api_key) {
      updateApiKey(payload.api_key);
      state.user = payload.user || null;
      renderPortfolio();
      log('bot login succeeded');
    } else {
      log('bot login response does not include api_key', 'error');
    }
  } catch (error) {
    log(`bot login failed: ${error.message}`, 'error');
  }
}

function connectWS() {
  if (!state.apiKey) {
    log('set API key before connecting websocket');
    return;
  }
  log('ブラウザWebSocketではX-API-Keyヘッダーを付与できないため、接続失敗時はサーバープロキシ経由を検討してください。');
  if (state.wsClient) state.wsClient.disconnect();

  state.wsClient = new RealtimeClient({
    baseUrl: ensureBaseUrl(),
    apiKey: state.apiKey,
    onStatus: (status) => {
      const cls = status === 'connected' ? 'good' : status === 'connecting' ? 'warn' : 'bad';
      setWsStatus(status, cls);
      log(`ws ${status}`);
    },
    onMessage: handleWsMessage,
    onError: (error) => log(`ws error: ${error.message}`, 'error'),
  });

  state.wsClient.connect();
  subscribeBaseTopics();
  subscribeAssetTopics();
}

function disconnectWS() {
  if (state.wsClient) {
    state.wsClient.disconnect();
    state.wsClient = null;
  }
}

function subscribeBaseTopics() {
  if (!state.wsClient) return;
  state.wsClient.subscribe(['market.ticker', 'news', 'user.orders', 'user.executions', 'user.portfolio']);
}

function subscribeAssetTopics() {
  if (!state.wsClient || !state.selectedAssetId) return;
  const id = state.selectedAssetId;
  state.wsClient.subscribe([`market.orderbook.${id}`, `market.trade.${id}`, `market.candles.${id}.1m`]);
}

function handleWsMessage(payload) {
  const topic = payload?.topic;
  const data = payload?.data;
  if (!topic) return;

  if (topic === 'market.ticker' && Array.isArray(data)) {
    state.tickers = data;
    renderMarketWatch();
    return;
  }
  if (topic === 'news' && Array.isArray(data)) {
    state.news = data;
    renderNews();
    return;
  }
  if (topic === 'user.orders' && Array.isArray(data)) {
    state.orders = data;
    renderOrders();
    return;
  }
  if (topic === 'user.portfolio' && data) {
    state.balances = Array.isArray(data.balances) ? data.balances : [];
    state.marginPositions = Array.isArray(data.positions) ? data.positions : [];
    state.portfolioAssets = Array.isArray(data.assets) ? data.assets : [];
    renderPortfolio();
    return;
  }
  if (topic.startsWith('market.orderbook.') && data) {
    state.orderbook = {
      bids: Array.isArray(data.bids) ? data.bids : [],
      asks: Array.isArray(data.asks) ? data.asks : [],
    };
    renderOrderbook();
    return;
  }
  if (topic.startsWith('market.trade.') && Array.isArray(data)) {
    state.trades = data;
    renderTrades();
    if (state.trades.length > 0) state.chart?.updateFromTrade(state.trades[0]);
    return;
  }
  if (topic.startsWith('market.candles.') && Array.isArray(data)) {
    state.chart?.setCandles(data);
    return;
  }
}

async function submitOrder(event) {
  event.preventDefault();
  const payload = {
    asset_id: Number(el.orderAsset.value),
    side: document.getElementById('order-side').value,
    type: document.getElementById('order-type').value,
    time_in_force: document.getElementById('order-tif').value,
    quantity: Number(document.getElementById('order-quantity').value),
    leverage: Number(document.getElementById('order-leverage').value || 1),
  };

  const price = Number(el.orderPrice.value);
  const stopPrice = Number(el.orderStopPrice.value);
  if (price > 0) payload.price = price;
  if (stopPrice > 0) payload.stop_price = stopPrice;

  try {
    await api('/orders', { method: 'POST', body: JSON.stringify(payload) });
    log('order submitted');
    await Promise.all([loadAssetDetail(), loadUserAndPortfolio()]);
  } catch (error) {
    log(`order submit failed: ${error.message}`, 'error');
  }
}

async function cancelOrder(order) {
  if (!order?.id || !order?.asset_id) return;
  try {
    await api(`/orders/${order.id}?asset_id=${order.asset_id}`, { method: 'DELETE' });
    log(`order ${order.id} cancelled`);
    await loadUserAndPortfolio();
  } catch (error) {
    log(`cancel failed: ${error.message}`, 'error');
  }
}

function bindEvents() {
  el.baseUrl.value = state.baseUrl;
  el.apiKey.value = state.apiKey;

  el.apiKey.addEventListener('change', () => updateApiKey(el.apiKey.value));
  el.baseUrl.addEventListener('change', () => ensureBaseUrl());
  el.btnLogin.addEventListener('click', loginBot);
  el.btnConnect.addEventListener('click', connectWS);
  el.btnDisconnect.addEventListener('click', disconnectWS);
  el.btnRefresh.addEventListener('click', refreshAll);
  el.orderForm.addEventListener('submit', submitOrder);
  el.orderType.addEventListener('change', () => {
    const type = el.orderType.value;
    el.orderPrice.disabled = !(type === 'LIMIT' || type === 'STOP_LIMIT');
    el.orderStopPrice.disabled = !(type === 'STOP' || type === 'STOP_LIMIT');
  });

  el.orderAsset.addEventListener('change', async () => {
    setSelectedAsset(Number(el.orderAsset.value));
    await loadAssetDetail();
    subscribeAssetTopics();
  });
}

async function refreshAll() {
  try {
    await loadBootstrap();
    await Promise.all([loadUserAndPortfolio(), loadAssetDetail()]);
  } catch (error) {
    setHttpStatus('error', 'bad');
    log(`refresh failed: ${error.message}`, 'error');
  }
}

async function init() {
  bindEvents();

  state.chart = new PriceChart(document.getElementById('chart-container'));
  state.chart.init();

  if (!state.apiKey) {
    setHttpStatus('waiting-api-key', 'warn');
    log('set API key first or login via bot credentials');
    return;
  }

  await refreshAll();
}

init();
