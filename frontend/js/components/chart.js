export class PriceChart {
  constructor(container) {
    this.container = container;
    this.chart = null;
    this.series = null;
    this.fallback = null;
    this.data = [];
  }

  init() {
    if (!this.container) return;
    if (window.LightweightCharts?.createChart) {
      this.chart = window.LightweightCharts.createChart(this.container, {
        layout: { background: { color: '#0a1119' }, textColor: '#9fb3ce' },
        grid: { vertLines: { color: '#1a2738' }, horzLines: { color: '#1a2738' } },
        rightPriceScale: { borderColor: '#2a3a4f' },
        timeScale: { borderColor: '#2a3a4f', timeVisible: true, secondsVisible: false },
      });
      this.series = this.chart.addCandlestickSeries({
        upColor: '#45d483', downColor: '#ff6b81', borderVisible: false,
        wickUpColor: '#45d483', wickDownColor: '#ff6b81',
      });
      this.resize();
      window.addEventListener('resize', () => this.resize());
      return;
    }

    this.fallback = document.createElement('div');
    this.fallback.className = 'mono muted';
    this.fallback.style.padding = '10px';
    this.container.appendChild(this.fallback);
    this.renderFallback();
  }

  resize() {
    if (!this.chart || !this.container) return;
    const rect = this.container.getBoundingClientRect();
    this.chart.applyOptions({ width: Math.max(300, Math.floor(rect.width)), height: Math.max(250, Math.floor(rect.height)) });
    this.chart.timeScale().fitContent();
  }

  setCandles(candles = [], fitContent = false) {
    const normalized = candles.filter((c) => c && Number.isFinite(c.timestamp)).map((c) => ({
      time: Math.floor(Number(c.timestamp) / 1000),
      open: Number(c.open), high: Number(c.high), low: Number(c.low), close: Number(c.close),
    })).sort((a, b) => a.time - b.time);

    const isFirstLoad = this.data.length === 0 && normalized.length > 0;
    this.data = normalized;

    if (this.series) {
      this.series.setData(normalized);
      if (fitContent || isFirstLoad) {
        this.chart?.timeScale()?.fitContent();
      }
    }
    this.renderFallback();
  }

  updateFromTrade(trade, timeframe = '1m') {
    if (!trade || !Number.isFinite(Number(trade.price))) return;
    const price = Number(trade.price);
    const epochSec = Math.floor(new Date(trade.occurred_at || Date.now()).getTime() / 1000);

    if (this.data.length === 0) {
      this.setCandles([{ timestamp: epochSec * 1000, open: price, high: price, low: price, close: price }]);
      return;
    }

    // Convert timeframe to seconds
    let bucketSize = 60; // 1m default
    if (timeframe === '1m') bucketSize = 60;
    else if (timeframe === '5m') bucketSize = 300;
    else if (timeframe === '15m') bucketSize = 900;
    else if (timeframe === '1h') bucketSize = 3600;
    else if (timeframe === '4h') bucketSize = 14400;
    else if (timeframe === '1d') bucketSize = 86400;

    const last = this.data[this.data.length - 1];
    const bucket = epochSec - (epochSec % bucketSize);
    if (last.time === bucket) {
      last.close = price;
      last.high = Math.max(last.high, price);
      last.low = Math.min(last.low, price);
      if (this.series) this.series.update(last);
    } else {
      const next = { time: bucket, open: last.close, high: price, low: price, close: price };
      this.data.push(next);
      if (this.series) this.series.update(next);
    }
    this.renderFallback();
  }

  renderFallback() {
    if (!this.fallback) return;
    if (this.data.length === 0) {
      this.fallback.textContent = 'Chart data unavailable';
      return;
    }
    const last = this.data[this.data.length - 1];
    this.fallback.textContent = `Last close: ${last.close} (candles: ${this.data.length})`;
  }
}
