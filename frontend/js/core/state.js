export const state = {
    baseUrl: localStorage.getItem('paperstreet.baseUrl') || `${location.protocol}//${location.hostname}:8000`,
    apiKey: localStorage.getItem('paperstreet.apiKey') || '',
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

export function setApiKey(key) {
    state.apiKey = key;
    localStorage.setItem('paperstreet.apiKey', key);
}
