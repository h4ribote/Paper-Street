import { state } from '../core/state.js';
import { fmtNumber } from '../core/format.js';
import { api } from '../core/api.js';

export async function renderIndexMarketWatch() {
    try {
        const tickers = await api('/api/market/ticker', {}, false); // no auth
        if (!tickers) return;

        const container = document.getElementById('index-market-watch');
        if (!container) return;

        let html = '';
        tickers.slice(0, 4).forEach(t => {
            const isUp = Number(t.change) >= 0;
            const colorClass = isUp ? 'text-trading-up' : 'text-trading-down';
            const sign = isUp ? '+' : '';
            html += `
            <div class="panel p-4 flex flex-col gap-2 hover:border-brand-500 border border-transparent transition-colors cursor-pointer bg-dark-panel rounded">
                <div class="flex justify-between items-center"><span class="text-dark-muted font-bold font-mono">${t.symbol}</span><span class="${colorClass} text-xs font-bold">${sign}${fmtNumber(t.change)}%</span></div>
                <div class="text-2xl font-mono text-white font-bold">${fmtNumber(t.price)}</div>
                <div class="text-xs text-dark-muted font-mono">Vol: ${fmtNumber(t.volume)}</div>
            </div>
            `;
        });
        container.innerHTML = html;
    } catch (e) {
        console.error("Failed to load tickers for index", e);
    }
}
