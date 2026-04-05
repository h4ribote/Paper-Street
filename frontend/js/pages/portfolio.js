import { loadHeader } from '../components/header.js';
import { requireAuth } from '../core/auth.js';
import { api } from '../core/api.js';
import { state } from '../core/state.js';
import { fmtNumber } from '../core/format.js';

async function initPortfolio() {
    if (!await requireAuth()) return;
    loadHeader('portfolio');

    try {
        const [balances, assets, positions] = await Promise.all([
            api('/api/portfolio/balances'),
            api('/api/portfolio/assets'),
            api('/api/margin/positions')
        ]);

        document.getElementById('portfolio-user-info').textContent = JSON.stringify(state.user, null, 2);

        // Calculate a very rough total ARC (assuming fiat is base for now, accurate calculation requires ticker prices)
        let totalArc = 0;

        const balContainer = document.getElementById('portfolio-balances');
        let balHtml = '';
        if (balances) {
            balances.forEach(b => {
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
        if (assets) {
            assets.forEach(a => {
                balHtml += `
                <tr class="border-t border-dark-border/50 hover:bg-dark-bg/50 transition-colors">
                    <td class="py-4 px-6 flex items-center gap-3 font-sans">
                        <div class="font-bold">Asset #${a.asset_id}</div>
                    </td>
                    <td class="py-4 px-6 text-right">${fmtNumber(a.quantity)}</td>
                </tr>
                `;
            });
        }
        balContainer.innerHTML = balHtml;
        document.getElementById('portfolio-total-arc').textContent = `~ ${fmtNumber(totalArc)}`;

        const posContainer = document.getElementById('portfolio-positions');
        let posHtml = '';
        if (positions) {
            positions.forEach(p => {
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
        posContainer.innerHTML = posHtml;

    } catch (e) {
        console.error("Portfolio init failed", e);
    }
}

document.addEventListener('DOMContentLoaded', initPortfolio);
