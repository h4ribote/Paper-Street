import { loadHeader } from '../components/header.js';
import { requireAuth } from '../core/auth.js';
import { api } from '../core/api.js';
import { fmtNumber, fmtDate } from '../core/format.js';
import { RealtimeClient } from '../core/ws.js';

let wsClient;
let currentTickers = [];
let currentNews = [];

function renderTickers() {
    if (!currentTickers || currentTickers.length === 0) return;
    
    // Ticker Cards
    const cardsContainer = document.getElementById('dashboard-ticker-cards');
    let cardsHtml = '';
    currentTickers.slice(0, 4).forEach(t => {
        const isUp = Number(t.change) >= 0;
        const colorClass = isUp ? 'text-trading-up' : 'text-trading-down';
        const sign = isUp ? '+' : '';
        cardsHtml += `
        <div class="panel p-4 flex flex-col gap-2 hover:border-brand-500 border border-transparent transition-colors cursor-pointer bg-dark-panel rounded" onclick="window.location.href='/trade.html?asset=${t.asset_id}'">
            <div class="flex justify-between items-center"><span class="text-dark-muted font-bold font-mono">${t.symbol}</span><span class="${colorClass} text-xs font-bold">${sign}${fmtNumber(t.change)}%</span></div>
            <div class="text-2xl font-mono text-white font-bold">${fmtNumber(t.price)}</div>
            <div class="text-xs text-dark-muted font-mono">Vol: ${fmtNumber(t.volume)}</div>
        </div>
        `;
    });
    if (cardsContainer) cardsContainer.innerHTML = cardsHtml;

    // Top Gainers
    const gainersContainer = document.getElementById('dashboard-top-gainers');
    const sorted = [...currentTickers].sort((a, b) => Number(b.change) - Number(a.change));
    let gainersHtml = '';
    sorted.slice(0, 5).forEach(t => {
        const isUp = Number(t.change) >= 0;
        const colorClass = isUp ? 'text-trading-up' : 'text-trading-down';
        const sign = isUp ? '+' : '';
        gainersHtml += `
        <tr class="border-b border-dark-border/50 hover:bg-dark-bg cursor-pointer" onclick="window.location.href='/trade.html?asset=${t.asset_id}'">
            <td class="py-3 px-4 flex items-center gap-2"><span class="font-bold">${t.symbol}</span></td>
            <td class="py-3 px-4 text-right">${fmtNumber(t.price)}</td>
            <td class="py-3 px-4 text-right ${colorClass}">${sign}${fmtNumber(t.change)}%</td>
            <td class="py-3 px-4 text-right hidden sm:table-cell">${fmtNumber(t.volume)}</td>
            <td class="py-3 px-4 text-right text-brand-500 font-sans hover:underline">取引</td>
        </tr>
        `;
    });
    if (gainersContainer) gainersContainer.innerHTML = gainersHtml;
}

function renderNews() {
    if (!currentNews || currentNews.length === 0) return;
    const newsContainer = document.getElementById('dashboard-mini-news');
    let newsHtml = '';
    currentNews.slice(0, 5).forEach(n => {
        newsHtml += `
        <div class="group cursor-pointer" onclick="window.location.href='/news.html'">
            <div class="flex items-center justify-between mb-1">
                <span class="text-xxs text-brand-500 border border-brand-500/30 px-1 rounded">${n.category || 'News'}</span>
                <span class="text-xxs text-dark-muted">${fmtDate(n.published_at)}</span>
            </div>
            <h4 class="text-sm font-bold text-white group-hover:text-brand-500 transition-colors line-clamp-2">${n.headline}</h4>
        </div>
        <div class="w-full h-px bg-dark-border last:hidden"></div>
        `;
    });
    if (newsContainer) newsContainer.innerHTML = newsHtml;
}

async function initDashboard() {
    if (!await requireAuth()) return;
    loadHeader('dashboard');

    try {
        const [tickers, news] = await Promise.all([
            api('/api/market/ticker'),
            api('/api/news?limit=5')
        ]);

        if (tickers) {
            currentTickers = tickers;
            renderTickers();
        }

        if (news) {
            currentNews = news;
            renderNews();
        }

        wsClient = new RealtimeClient({
            onMessage: (msg) => {
                if (msg.topic === 'market.ticker') {
                    // Update tickers
                    // We expect message payload could be an array of all tickers
                    if (Array.isArray(msg.data)) {
                        currentTickers = msg.data;
                        renderTickers();
                    }
                } else if (msg.topic === 'news') {
                    currentNews.unshift(msg.data);
                    renderNews();
                }
            }
        });
        wsClient.connect();
        wsClient.subscribe(['market.ticker', 'news']);
    } catch (e) {
        console.error("Dashboard init failed", e);
    }
}

document.addEventListener('DOMContentLoaded', initDashboard);
