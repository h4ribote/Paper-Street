import { state } from '../core/state.js';
import { fmtNumber } from '../core/format.js';

export function renderOrderbook(containerBids, containerAsks, midPriceEl, arrowEl) {
    if (!containerBids || !containerAsks) return;

    let askHtml = '';
    let askTotal = 0;
    const asks = [...state.orderbook.asks].reverse(); // Highest ask first down to lowest ask

    asks.forEach(ask => {
        askTotal += Number(ask.quantity);
        const depthRatio = Math.min((askTotal / 15) * 100, 100); // 15 is arbitrary max for visual
        askHtml += `
            <div class="flex justify-between px-3 py-[2px] font-mono text-xs relative cursor-pointer hover:bg-dark-border">
                <div class="absolute top-0 right-0 bottom-0 bg-trading-down opacity-10" style="width: ${depthRatio}%"></div>
                <span class="text-trading-down z-10">${fmtNumber(ask.price, 2)}</span>
                <span class="text-white z-10">${fmtNumber(ask.quantity, 4)}</span>
                <span class="text-dark-muted z-10">${fmtNumber(askTotal, 4)}</span>
            </div>`;
    });
    containerAsks.innerHTML = askHtml;

    let bidHtml = '';
    let bidTotal = 0;
    const bids = state.orderbook.bids; // Highest bid first down to lowest bid

    bids.forEach(bid => {
        bidTotal += Number(bid.quantity);
        const depthRatio = Math.min((bidTotal / 15) * 100, 100);
        bidHtml += `
            <div class="flex justify-between px-3 py-[2px] font-mono text-xs relative cursor-pointer hover:bg-dark-border">
                <div class="absolute top-0 right-0 bottom-0 bg-trading-up opacity-10" style="width: ${depthRatio}%"></div>
                <span class="text-trading-up z-10">${fmtNumber(bid.price, 2)}</span>
                <span class="text-white z-10">${fmtNumber(bid.quantity, 4)}</span>
                <span class="text-dark-muted z-10">${fmtNumber(bidTotal, 4)}</span>
            </div>`;
    });
    containerBids.innerHTML = bidHtml;

    if (midPriceEl) {
        if (asks.length > 0 && bids.length > 0) {
            const mid = (Number(asks[asks.length-1].price) + Number(bids[0].price)) / 2;
            const currentText = midPriceEl.textContent;
            midPriceEl.textContent = fmtNumber(mid, 2);

            // basic color logic
            if (currentText !== '--') {
                const prevMid = parseFloat(currentText.replace(/,/g, ''));
                if (mid > prevMid) {
                    midPriceEl.className = 'font-mono font-bold text-lg text-trading-up';
                    if (arrowEl) arrowEl.className = 'fa-solid fa-arrow-up text-xs text-trading-up';
                } else if (mid < prevMid) {
                    midPriceEl.className = 'font-mono font-bold text-lg text-trading-down';
                    if (arrowEl) arrowEl.className = 'fa-solid fa-arrow-down text-xs text-trading-down';
                }
            }
        }
    }
}
