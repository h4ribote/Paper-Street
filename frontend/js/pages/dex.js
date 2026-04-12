import { loadHeader } from '../components/header.js';
import { api } from '../core/api.js';
import { state } from '../core/state.js';
import { checkAuth } from '../core/auth.js';
import { formatNumber } from '../core/format.js';

let pools = [];
let balances = { cash: 0, assets: {} };
let availableAssets = [];

async function init() {
    if (!checkAuth()) return;
    loadHeader('dex');

    setupTabs();
    setupSwapForm();
    setupLiquidityForm();

    await fetchData();
    renderUI();
}

function setupTabs() {
    const btns = document.querySelectorAll('.tab-btn');
    btns.forEach(btn => {
        btn.addEventListener('click', (e) => {
            btns.forEach(b => {
                b.classList.remove('text-white', 'border-brand-500');
                b.classList.add('text-dark-muted', 'border-transparent');
            });
            const t = e.target;
            t.classList.remove('text-dark-muted', 'border-transparent');
            t.classList.add('text-white', 'border-brand-500');

            document.querySelectorAll('.tab-content').forEach(c => c.classList.add('hidden'));
            document.getElementById(t.dataset.target).classList.remove('hidden');
        });
    });
}

async function fetchData() {
    try {
        const [poolData, balanceData, assetData, allAssets] = await Promise.all([
            api('/api/pools'),
            api('/api/portfolio/balances'),
            api('/api/portfolio/assets'),
            api('/api/assets')
        ]);
        
        pools = poolData || [];
        balances.cash = balanceData?.balance || 0;
        
        const amap = {};
        (assetData || []).forEach(a => {
            amap[a.asset_id] = a.quantity;
        });
        balances.assets = amap;
        availableAssets = allAssets || [];

        await fetchLpPositions();
    } catch (e) {
        console.error('Failed to fetch DEX data:', e);
        alert('データの取得に失敗しました。');
    }
}

async function fetchLpPositions() {
    try {
        const posBody = document.getElementById('lp-positions-body');
        const lpPos = await api('/api/pools/positions');
        if (!lpPos || lpPos.length === 0) {
            posBody.innerHTML = `<tr><td colspan="4" class="text-center py-4 text-dark-muted">ポジションはありません</td></tr>`;
            return;
        }

        posBody.innerHTML = '';
        lpPos.forEach(p => {
            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td class="py-3 px-4 text-white font-bold">Pool #${p.pool_id}</td>
                <td class="py-3 px-4 text-right text-brand-500">${formatNumber(p.liquidity, 2)}</td>
                <td class="py-3 px-4 text-right text-dark-muted">[${p.lower_tick} - ${p.upper_tick}]</td>
                <td class="py-3 px-4 text-right">
                    <button class="text-trading-down hover:text-white border border-trading-down hover:bg-trading-down rounded px-2 py-1 transition-colors text-xs" data-id="${p.id}" onclick="removeLiquidity('${p.id}')">
                        解除 (Remove)
                    </button>
                </td>
            `;
            posBody.appendChild(tr);
        });
    } catch(e) {
        document.getElementById('lp-positions-body').innerHTML = `<tr><td colspan="4" class="text-center py-4 text-trading-down">取得エラー</td></tr>`;
    }
}

window.removeLiquidity = async function(id) {
    if (!confirm('流動性を解除しますか？')) return;
    try {
        await api(`/api/pools/positions/${id}`, { method: 'DELETE' });
        alert('流動性を解除し、資金を回収しました。');
        await fetchData();
        renderUI();
    } catch (e) {
        alert('エラー: ' + e.message);
    }
}

function renderUI() {
    // Populate select boxes
    const selFrom = document.getElementById('swap-from-asset');
    const selTo = document.getElementById('swap-to-asset');
    const selPool = document.getElementById('liquidity-pool');
    
    selFrom.innerHTML = `<option value="USD">USD</option>`;
    selTo.innerHTML = `<option value="USD">USD</option>`;
    
    availableAssets.forEach(a => {
        selFrom.innerHTML += `<option value="${a.id}">${a.symbol} (${a.name})</option>`;
        selTo.innerHTML += `<option value="${a.id}">${a.symbol} (${a.name})</option>`;
    });

    selPool.innerHTML = '<option value="">(自動選択または指定ルーター)</option>';
    pools.forEach(p => {
        selPool.innerHTML += `<option value="${p.id}">Pool #${p.id} (${p.base_asset} / ${p.quote_asset}) [手数料: ${p.fee_tier}%]</option>`;
    });

    updateSwapBalances();
}

function updateSwapBalances() {
    const from = document.getElementById('swap-from-asset').value;
    const to = document.getElementById('swap-to-asset').value;
    
    const balFrom = from === 'USD' ? balances.cash : (balances.assets[from] || 0);
    const balTo = to === 'USD' ? balances.cash : (balances.assets[to] || 0);
    
    document.getElementById('swap-from-bal').textContent = `残高: ${formatNumber(balFrom, 4)}`;
    document.getElementById('swap-to-bal').textContent = `残高: ${formatNumber(balTo, 4)}`;
}

function setupSwapForm() {
    document.getElementById('swap-from-asset').addEventListener('change', updateSwapBalances);
    document.getElementById('swap-to-asset').addEventListener('change', updateSwapBalances);

    document.getElementById('btn-swap-flip').addEventListener('click', () => {
        const from = document.getElementById('swap-from-asset');
        const to = document.getElementById('swap-to-asset');
        const temp = from.value;
        from.value = to.value;
        to.value = temp;
        updateSwapBalances();
    });

    document.getElementById('btn-submit-swap').addEventListener('click', async (e) => {
        e.preventDefault();
        const from = document.getElementById('swap-from-asset').value;
        const to = document.getElementById('swap-to-asset').value;
        const amt = parseFloat(document.getElementById('swap-amount').value);
        
        if (from === to) return alert('同じ通貨ペアです。');
        if (!amt || amt <= 0) return alert('数量を入力してください。');

        try {
            await api('/api/pools/0/swap', {
                method: 'POST',
                body: JSON.stringify({
                    from_currency: from,
                    to_currency: to,
                    amount: amt
                })
            });
            alert('スワップが完了しました。');
            document.getElementById('swap-amount').value = '';
            await fetchData();
            renderUI();
        } catch (e) {
            alert('スワップエラー: ' + e.message);
        }
    });
}

function setupLiquidityForm() {
    document.getElementById('liquidity-pool').addEventListener('change', (e) => {
        const val = e.target.value;
        if (!val) return;
        const p = pools.find(x => x.id == val);
        if (p) {
            document.getElementById('lp-base-label').textContent = `${p.base_asset} Amount`;
            document.getElementById('lp-quote-label').textContent = `${p.quote_asset} Amount`;
            
            // auto-suggest tick boundaries for demo based on current tick if available
            if (p.current_tick) {
                document.getElementById('lp-lower-tick').value = p.current_tick - 1000;
                document.getElementById('lp-upper-tick').value = p.current_tick + 1000;
            } else {
                document.getElementById('lp-lower-tick').value = -10000;
                document.getElementById('lp-upper-tick').value = 10000;
            }
        }
    });

    document.getElementById('btn-add-liquidity').addEventListener('click', async(e) => {
        e.preventDefault();
        const poolId = document.getElementById('liquidity-pool').value;
        const baseAmt = parseFloat(document.getElementById('lp-base-amount').value);
        const quoteAmt = parseFloat(document.getElementById('lp-quote-amount').value);
        const lTick = parseInt(document.getElementById('lp-lower-tick').value);
        const uTick = parseInt(document.getElementById('lp-upper-tick').value);

        if (!poolId) return alert('プールを選択してください。');
        
        try {
            await api(`/api/pools/${poolId}/positions`, {
                method: 'POST',
                body: JSON.stringify({
                    base_amount: baseAmt,
                    quote_amount: quoteAmt,
                    lower_tick: lTick,
                    upper_tick: uTick
                })
            });
            alert('流動性を提供しました。');
            document.getElementById('lp-base-amount').value = '';
            document.getElementById('lp-quote-amount').value = '';
            await fetchData();
            renderUI();
        } catch (e) {
            alert('エラー: ' + e.message);
        }
    });
}

document.addEventListener('DOMContentLoaded', init);