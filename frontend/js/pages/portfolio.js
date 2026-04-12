import { loadHeader } from '../components/header.js';
import { requireAuth } from '../core/auth.js';
import { api } from '../core/api.js';
import { state } from '../core/state.js';
import { fmtNumber } from '../core/format.js';
import { RealtimeClient } from '../core/ws.js';

let wsClient;
let currentBalances = [];
let currentAssets = [];
let currentPositions = [];

function renderPortfolio() {
    document.getElementById('portfolio-user-info').textContent = JSON.stringify(state.user, null, 2);

    // Calculate a very rough total ARC
    let totalArc = 0;

    const balContainer = document.getElementById('portfolio-balances');
    let balHtml = '';
    if (currentBalances) {
        currentBalances.forEach(b => {
            if(b.currency === 'ARC') totalArc += Number(b.amount);
            balHtml += `
            <tr class="hover:bg-dark-bg/50 transition-colors">
                <td class="py-4 px-6 flex items-center gap-3 font-sans">
                    <div class="font-bold">${b.currency}</div>
                </td>
                <td class="py-4 px-6 text-right">${fmtNumber(b.amount)}</td>
            </tr>
            `;
        });
    }
    if (currentAssets) {
        currentAssets.forEach(a => {
            balHtml += `
            <tr class="border-t border-dark-border/50 hover:bg-dark-bg/50 transition-colors">
                <td class="py-4 px-6 flex items-center gap-3 font-sans">
                    <div class="font-bold">Asset #${a.asset ? a.asset.id : '?'}</div>
                </td>
                <td class="py-4 px-6 text-right">${fmtNumber(a.quantity)}</td>
            </tr>
            `;
        });
    }
    if (balContainer) balContainer.innerHTML = balHtml;
    const totalArcEl = document.getElementById('portfolio-total-arc');
    if (totalArcEl) totalArcEl.textContent = `~ ${fmtNumber(totalArc)}`;

    const posContainer = document.getElementById('portfolio-positions');
    let posHtml = '';
    if (currentPositions) {
        currentPositions.forEach(p => {
            const pnl = -Number(p.unrealized_loss || 0);
            const pnlClass = pnl >= 0 ? 'text-trading-up' : 'text-trading-down';
            posHtml += `
            <tr class="hover:bg-dark-bg/50 transition-colors">
                <td class="py-4 px-6">${p.id}</td>
                <td class="py-4 px-6">${p.asset_id}</td>
                <td class="py-4 px-6">${p.side}</td>
                <td class="py-4 px-6 text-right">${fmtNumber(p.quantity)}</td>
                <td class="py-4 px-6 text-right ${pnlClass}">${fmtNumber(pnl)}</td>
            </tr>
            `;
        });
    }
    if (posContainer) posContainer.innerHTML = posHtml;
}

async function initPortfolio() {
    if (!await requireAuth()) return;
    loadHeader('portfolio');

    try {
        const [balances, assets, positions] = await Promise.all([
            api('/api/portfolio/balances'),
            api('/api/portfolio/assets'),
            api('/api/margin/positions')
        ]);

        currentBalances = balances || [];
        currentAssets = assets || [];
        currentPositions = positions || [];

        renderPortfolio();

        wsClient = new RealtimeClient({
            onMessage: (msg) => {
                if (msg.topic === 'user.portfolio') {
                    if (msg.data.balances) currentBalances = msg.data.balances;
                    if (msg.data.assets) currentAssets = msg.data.assets;
                    if (msg.data.positions) currentPositions = msg.data.positions;
                    renderPortfolio();
                }
            }
        });
        wsClient.connect();
        wsClient.subscribe(['user.portfolio']);
    } catch (e) {
        console.error("Portfolio init failed", e);
    }
}

document.addEventListener('DOMContentLoaded', initPortfolio);
