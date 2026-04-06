import { state } from '../core/state.js';
import { logout } from '../core/auth.js';

export function loadHeader(activeTab) {
    const headerHtml = `
    <nav class="h-14 flex items-center justify-between px-4 border-b border-dark-border bg-dark-panel shrink-0 z-10">
        <div class="flex items-center gap-8">
            <a href="/index.html" class="flex items-center gap-2 font-bold text-xl tracking-tighter text-white">
                <i class="fa-solid fa-layer-group text-brand-500"></i>
                <span>PAPER STREET</span>
            </a>
            <!-- Navigation Links -->
            <div class="hidden md:flex gap-6 font-medium">
                <a href="/dashboard.html" class="nav-link ${activeTab === 'dashboard' ? 'text-white border-brand-500' : 'text-dark-muted border-transparent hover:text-white'} transition-colors pb-4 mt-4 border-b-2">ダッシュボード</a>
                <a href="/trade.html" class="nav-link ${activeTab === 'trade' ? 'text-white border-brand-500' : 'text-dark-muted border-transparent hover:text-white'} transition-colors pb-4 mt-4 border-b-2">現物取引</a>
                <a href="/margin.html" class="nav-link ${activeTab === 'margin' ? 'text-white border-brand-500' : 'text-dark-muted border-transparent hover:text-white'} transition-colors pb-4 mt-4 border-b-2">信用取引</a>
                <a href="/news.html" class="nav-link ${activeTab === 'news' ? 'text-white border-brand-500' : 'text-dark-muted border-transparent hover:text-white'} transition-colors pb-4 mt-4 border-b-2">ニュース</a>
                <a href="/portfolio.html" class="nav-link ${activeTab === 'portfolio' ? 'text-white border-brand-500' : 'text-dark-muted border-transparent hover:text-white'} transition-colors pb-4 mt-4 border-b-2">ポートフォリオ</a>
            </div>
        </div>
        <div class="flex items-center gap-4">
            ${state.user ? `
            <div class="flex items-center gap-3 mr-4">
                <div class="text-right hidden sm:block">
                    <div class="text-xs text-dark-muted">User ID</div>
                    <div class="font-mono font-bold">${state.user.id}</div>
                </div>
            </div>
            <button id="btn-logout" class="text-dark-muted hover:text-white"><i class="fa-solid fa-sign-out-alt"></i></button>
            <div class="w-8 h-8 rounded-full bg-brand-600 flex items-center justify-center font-bold text-white cursor-pointer hover:bg-brand-500">
                ${state.user.id}
            </div>
            ` : `
            <a href="/login.html" class="text-sm font-bold text-white bg-brand-600 hover:bg-brand-500 px-4 py-1.5 rounded transition-colors">ログイン</a>
            `}
        </div>
    </nav>
    `;

    document.getElementById('main-header').innerHTML = headerHtml;

    const logoutBtn = document.getElementById('btn-logout');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', logout);
    }
}
