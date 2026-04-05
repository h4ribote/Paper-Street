import { loadHeader } from '../components/header.js';
import { requireAuth } from '../core/auth.js';
import { api } from '../core/api.js';
import { state } from '../core/state.js';
import { fmtNumber, fmtDate } from '../core/format.js';

async function initNews() {
    if (!await requireAuth()) return;
    loadHeader('news');

    const container = document.getElementById('news-container');
    const searchInput = document.getElementById('news-search');
    const refreshBtn = document.getElementById('btn-refresh-news');

    async function loadNews() {
        try {
            // simple fetch, no pagination for now
            const news = await api('/api/news?limit=50');
            renderNews(news);
        } catch (e) {
            console.error(e);
        }
    }

    function renderNews(newsList) {
        let html = '';
        const query = searchInput.value.toLowerCase();

        newsList.filter(n => n.headline.toLowerCase().includes(query) || (n.body && n.body.toLowerCase().includes(query))).forEach(n => {
            html += `
            <div class="group border-b border-dark-border pb-6 last:border-0">
                <div class="flex items-center gap-3 mb-2">
                    <span class="bg-brand-500/20 text-brand-500 text-xs px-2 py-0.5 rounded font-medium border border-brand-500/30">${n.category || 'General'}</span>
                    <span class="text-dark-muted text-xs"><i class="fa-regular fa-clock mr-1"></i>${fmtDate(n.published_at)}</span>
                </div>
                <h4 class="text-lg md:text-xl font-bold text-white mb-2 group-hover:text-brand-500 transition-colors cursor-pointer">${n.headline}</h4>
                <p class="text-sm text-dark-muted leading-relaxed mb-3">${n.body || ''}</p>
            </div>
            `;
        });
        container.innerHTML = html;
    }

    refreshBtn.addEventListener('click', loadNews);
    searchInput.addEventListener('input', async () => {
        const news = await api('/api/news?limit=50'); // ideally cache this
        renderNews(news);
    });

    loadNews();
}

document.addEventListener('DOMContentLoaded', initNews);
