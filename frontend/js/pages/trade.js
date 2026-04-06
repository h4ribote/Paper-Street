import { loadHeader } from '../components/header.js';
import { requireAuth } from '../core/auth.js';
import { api } from '../core/api.js';
import { state } from '../core/state.js';
import { fmtNumber, fmtTime } from '../core/format.js';
import { renderOrderbook } from '../components/orderbook.js';
import { PriceChart } from '../components/chart.js';
import { RealtimeClient } from '../core/ws.js';

let chart;
let wsClient;

async function initTrade() {
    if (!await requireAuth()) return;
    loadHeader('trade');

    // Parse URL params for asset
    const urlParams = new URLSearchParams(window.location.search);
    let selectedAssetId = urlParams.get('asset');

    // Load static data
    const assets = await api('/api/assets');
    state.assets = assets || [];
    if (state.assets.length > 0 && !selectedAssetId) {
        selectedAssetId = state.assets[0].id;
    }
    state.selectedAssetId = selectedAssetId;

    // UI elements
    const marketList = document.getElementById('market-list');
    const tickerHeader = document.getElementById('ticker-header');

    // Setup Chart
    chart = new PriceChart(document.getElementById('tvchart'));
    chart.init();

    // Event Listeners for Order Form
    setupOrderForm();

    // Event Listeners for Tabs
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active', 'text-white', 'border-brand-500'));
            e.target.classList.add('active', 'text-white', 'border-brand-500');

            document.querySelectorAll('.tab-content').forEach(c => c.classList.add('hidden'));
            document.getElementById(e.target.dataset.target).classList.remove('hidden');
        });
    });

    document.querySelector('.tab-btn[data-target="open-orders"]').click();

    // Connect WS
    connectWS();

    // Initial Load
    await loadMarketList();
    await loadAssetData();
    await loadUserData();
}

async function loadMarketList() {
    const tickers = await api('/api/market/ticker');
    let html = '';
    if (tickers) {
        tickers.forEach(t => {
            const isUp = Number(t.change) >= 0;
            const colorClass = isUp ? 'text-trading-up' : 'text-trading-down';
            html += `
                <div class="flex justify-between items-center px-3 py-1.5 hover:bg-dark-bg cursor-pointer group" onclick="window.location.href='/trade.html?asset=${t.asset_id}'">
                    <div class="flex items-center gap-2">
                        <span class="font-bold font-mono text-xs ${t.asset_id == state.selectedAssetId ? 'text-brand-500' : 'text-white'}">${t.symbol}</span>
                    </div>
                    <div class="flex gap-3 text-right font-mono text-xs">
                        <span class="${colorClass}">${fmtNumber(t.price)}</span>
                    </div>
                </div>`;
        });
    }
    document.getElementById('market-list').innerHTML = html;
}

async function loadAssetData() {
    if (!state.selectedAssetId) return;
    const id = state.selectedAssetId;

    // Update Header
    const asset = state.assets.find(a => a.id == id) || { symbol: `Asset #${id}` };
    const tickers = await api('/api/market/ticker');
    const ticker = tickers?.find(t => t.asset_id == id) || { price: 0, change: 0, volume: 0 };

    document.getElementById('ticker-header').innerHTML = `
        <div class="flex items-center gap-3 shrink-0">
            <h1 class="text-2xl font-bold font-mono text-white">${asset.symbol}</h1>
        </div>
        <div class="flex flex-col shrink-0">
            <span class="font-mono font-bold text-lg text-white">${fmtNumber(ticker.price)}</span>
        </div>
        <div class="flex flex-col shrink-0 ml-4">
            <span class="text-dark-muted text-xs">24h 変動</span>
            <span class="${Number(ticker.change) >= 0 ? 'text-trading-up' : 'text-trading-down'} font-mono text-sm">${fmtNumber(ticker.change)}%</span>
        </div>
        <div class="flex flex-col shrink-0 ml-4">
            <span class="text-dark-muted text-xs">24h 取引高</span>
            <span class="font-mono text-sm text-white">${fmtNumber(ticker.volume)}</span>
        </div>
    `;

    // Load REST data
    const [ob, trades, candles] = await Promise.all([
        api(`/api/market/orderbook/${id}?depth=20`),
        api(`/api/market/trades/${id}?limit=50`),
        api(`/api/market/candles/${id}?timeframe=1m&limit=100`)
    ]);

    if (ob) {
        state.orderbook = { bids: ob.bids || [], asks: ob.asks || [] };
        updateOrderbookUI();
    }
    if (trades) {
        state.trades = trades;
        updateTradesUI();
    }
    if (candles && chart) {
        chart.setCandles(candles);
    }
}

async function loadUserData() {
    const [balances, orders, positions] = await Promise.all([
        api('/api/portfolio/balances'),
        api('/api/orders?status=OPEN&limit=100'),
        api('/api/margin/positions')
    ]);

    if (balances) {
        const arc = balances.find(b => b.currency === 'ARC');
        document.getElementById('available-balance').textContent = arc ? `${fmtNumber(arc.amount)} ARC` : '0.00 ARC';
    }

    if (orders) {
        state.orders = orders;
        updateOrdersUI();
    }

    if (positions) {
        state.marginPositions = positions;
        updatePositionsUI();
    }
}

function updateOrderbookUI() {
    renderOrderbook(
        document.getElementById('orderbook-bids'),
        document.getElementById('orderbook-asks'),
        document.getElementById('ob-mid-price'),
        document.getElementById('ob-mid-arrow')
    );
}

function updateTradesUI() {
    const tbody = document.getElementById('trades-body');
    let html = '';
    (state.trades || []).slice(0, 50).forEach(t => {
        const isBuyer = t.taker_user_id === state.user?.id; // simplified
        html += `
            <tr class="border-b border-dark-border/50 hover:bg-dark-bg/50">
                <td class="py-2 px-4">${fmtTime(t.occurred_at)}</td>
                <td class="py-2 px-4">${state.selectedAssetId}</td>
                <td class="py-2 px-4 ${isBuyer ? 'text-trading-up' : 'text-trading-down'}">${fmtNumber(t.price)}</td>
                <td class="py-2 px-4">${fmtNumber(t.quantity)}</td>
                <td class="py-2 px-4">${isBuyer ? 'TAKER' : 'MAKER'}</td>
            </tr>
        `;
    });
    tbody.innerHTML = html;
}

function updateOrdersUI() {
    const tbody = document.getElementById('orders-body');
    let html = '';
    (state.orders || []).forEach(o => {
        html += `
            <tr class="border-b border-dark-border/50 hover:bg-dark-bg/50">
                <td class="py-2 px-4">${o.id}</td>
                <td class="py-2 px-4">${fmtTime(o.created_at)}</td>
                <td class="py-2 px-4">${o.asset_id}</td>
                <td class="py-2 px-4">${o.type}</td>
                <td class="py-2 px-4 ${o.side === 'BUY' ? 'text-trading-up' : 'text-trading-down'}">${o.side}</td>
                <td class="py-2 px-4">${fmtNumber(o.price)}</td>
                <td class="py-2 px-4">${fmtNumber(o.quantity)}</td>
                <td class="py-2 px-4 text-right">
                    <button class="text-dark-muted hover:text-white border border-dark-border px-2 py-1 rounded text-xxs" onclick="window.cancelOrder(${o.id}, ${o.asset_id})">Cancel</button>
                </td>
            </tr>
        `;
    });
    tbody.innerHTML = html;
}

window.cancelOrder = async function(orderId, assetId) {
    try {
        await api(`/api/orders/${orderId}?asset_id=${assetId}`, { method: 'DELETE' });
        loadUserData();
    } catch (e) {
        console.error("Cancel failed", e);
    }
}

function updatePositionsUI() {
    const tbody = document.getElementById('positions-body');
    let html = '';
    (state.marginPositions || []).forEach(p => {
        const pnl = -Number(p.unrealized_loss || 0);
        html += `
            <tr class="border-b border-dark-border/50 hover:bg-dark-bg/50">
                <td class="py-2 px-4">${p.asset_id}</td>
                <td class="py-2 px-4 ${p.side === 'LONG' ? 'text-trading-up' : 'text-trading-down'}">${fmtNumber(p.quantity)}</td>
                <td class="py-2 px-4">${fmtNumber(p.entry_price || 0)}</td>
                <td class="py-2 px-4 text-right ${pnl >= 0 ? 'text-trading-up' : 'text-trading-down'}">${fmtNumber(pnl)}</td>
            </tr>
        `;
    });
    tbody.innerHTML = html;
}

function setupOrderForm() {
    const typeInputs = ['btn-type-limit', 'btn-type-market', 'btn-type-stop'];
    const priceGroup = document.getElementById('price-input-group');
    const stopGroup = document.getElementById('stop-price-input-group');
    const typeInput = document.getElementById('order-type');

    document.getElementById('btn-type-limit').addEventListener('click', () => {
        typeInput.value = 'LIMIT';
        priceGroup.classList.remove('hidden');
        stopGroup.classList.add('hidden');
        updateTypeBtns('btn-type-limit');
    });
    document.getElementById('btn-type-market').addEventListener('click', () => {
        typeInput.value = 'MARKET';
        priceGroup.classList.add('hidden');
        stopGroup.classList.add('hidden');
        updateTypeBtns('btn-type-market');
    });
    document.getElementById('btn-type-stop').addEventListener('click', () => {
        typeInput.value = 'STOP';
        priceGroup.classList.add('hidden');
        stopGroup.classList.remove('hidden');
        updateTypeBtns('btn-type-stop');
    });

    function updateTypeBtns(activeId) {
        typeInputs.forEach(id => {
            const el = document.getElementById(id);
            if (id === activeId) {
                el.className = 'flex-1 py-1.5 rounded bg-dark-panel text-white shadow';
            } else {
                el.className = 'flex-1 py-1.5 rounded text-dark-muted hover:text-white';
            }
        });
    }

    document.getElementById('btn-buy').addEventListener('click', () => submitOrder('BUY'));
    document.getElementById('btn-sell').addEventListener('click', () => submitOrder('SELL'));
}

async function submitOrder(side) {
    const payload = {
        asset_id: Number(state.selectedAssetId),
        side: side,
        type: document.getElementById('order-type').value,
        time_in_force: 'GTC',
        quantity: Number(document.getElementById('order-quantity').value),
        leverage: Number(document.getElementById('order-leverage').value || 1),
    };

    if (payload.type === 'LIMIT') {
        payload.price = Number(document.getElementById('order-price').value);
    } else if (payload.type === 'STOP') {
        payload.stop_price = Number(document.getElementById('order-stop-price').value);
    }

    try {
        await api('/api/orders', { method: 'POST', body: JSON.stringify(payload) });
        // alert('Order submitted successfully'); // Optional feedback
        loadUserData(); // refresh orders/balances
    } catch (e) {
        alert(`Order failed: ${e.message}`);
    }
}

function connectWS() {
    if (!state.apiKey || !state.selectedAssetId) return;

    wsClient = new RealtimeClient({
        baseUrl: state.baseUrl,
        apiKey: state.apiKey,
        onMessage: (msg) => {
            if (!msg.topic) return;
            const topic = msg.topic;
            const data = msg.data;

            if (topic.startsWith('market.orderbook.')) {
                state.orderbook = { bids: data.bids || [], asks: data.asks || [] };
                updateOrderbookUI();
            } else if (topic.startsWith('market.order.')) {
                const order = data;
                if (order.type !== 'LIMIT' && order.type !== 'STOP_LIMIT') return;

                if (!state.activeOrders) state.activeOrders = new Map();

                const prevOrder = state.activeOrders.get(order.id);
                const isNew = !prevOrder && order.created_at === order.updated_at;

                if (!prevOrder && !isNew) {
                    // We missed the previous state of this order. Resync orderbook.
                    wsClient.subscribe([`market.orderbook.${state.selectedAssetId}`]);
                    return;
                }

                const isBuy = order.side === 'BUY';
                const price = Number(order.price);
                const book = isBuy ? state.orderbook.bids : state.orderbook.asks;

                const prevRemaining = prevOrder ? Number(prevOrder.remaining) : 0;

                let newRemaining = Number(order.remaining);
                if (order.status === 'FILLED' || order.status === 'CANCELLED' || order.status === 'REJECTED') {
                    newRemaining = 0;
                }

                const diff = newRemaining - prevRemaining;

                if (diff !== 0) {
                    let level = book.find(l => Number(l.price) === price);
                    if (!level) {
                        level = { price: price, quantity: 0 };
                        book.push(level);
                    }
                    level.quantity = Number(level.quantity) + diff;
                    if (level.quantity <= 0) {
                        const idx = book.indexOf(level);
                        if (idx > -1) book.splice(idx, 1);
                    }

                    if (isBuy) {
                        book.sort((a, b) => Number(b.price) - Number(a.price));
                    } else {
                        book.sort((a, b) => Number(a.price) - Number(b.price));
                    }
                    updateOrderbookUI();
                }

                if (newRemaining === 0) {
                    state.activeOrders.delete(order.id);
                } else {
                    state.activeOrders.set(order.id, order);
                }
            } else if (topic.startsWith('market.trade.')) {
                state.trades = data;
                updateTradesUI();
                if (chart && data.length > 0) chart.updateFromTrade(data[0]);
            } else if (topic.startsWith('market.candles.')) {
                if (chart) chart.setCandles(data);
            } else if (topic === 'user.orders') {
                state.orders = data;
                updateOrdersUI();
            } else if (topic === 'user.portfolio') {
                if (data.positions) {
                    state.marginPositions = data.positions;
                    updatePositionsUI();
                }
            }
        }
    });

    wsClient.connect();
    const id = state.selectedAssetId;
    wsClient.subscribe([
        `market.orderbook.${id}`,
        `market.trade.${id}`,
        `market.order.${id}`,
        `market.candles.${id}.1m`,
        'user.orders',
        'user.portfolio'
    ]);
}

document.addEventListener('DOMContentLoaded', initTrade);
