-- ==========================================
-- Paper Street: Initialization SQL
-- ==========================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 通貨・価格は固定小数点の整数で保存します (1.000000 = 1000000)

-- --------------------------------------------------------
-- 1. Game Meta & World Setting (マスタデータ)
-- --------------------------------------------------------

-- シーズン管理 (Season Cycle)
CREATE TABLE IF NOT EXISTS seasons (
    season_id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT 'e.g., Season 1: The Great Depression',
    theme_code VARCHAR(50) COMMENT 'gameplay modifier code',
    start_at BIGINT COMMENT 'Unix Timestamp (ms)',
    end_at BIGINT COMMENT 'Unix Timestamp (ms)',
    is_active BOOLEAN DEFAULT TRUE
);

-- 地域 (Geopolitical Regions)
CREATE TABLE IF NOT EXISTS regions (
    region_id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT
);

-- 国家 (Countries / Origins)
CREATE TABLE IF NOT EXISTS countries (
    country_id INT AUTO_INCREMENT PRIMARY KEY,
    region_id INT,
    name VARCHAR(100) NOT NULL, -- e.g., Neo Venice, Arcadia
    FOREIGN KEY (region_id) REFERENCES regions(region_id)
);

-- 通貨マスタ (Currencies)
CREATE TABLE IF NOT EXISTS currencies (
    currency_id INT AUTO_INCREMENT PRIMARY KEY,
    country_id INT,
    code VARCHAR(5) UNIQUE NOT NULL, -- VND, BRB, DRL...
    name VARCHAR(50) NOT NULL,
    FOREIGN KEY (country_id) REFERENCES countries(country_id)
);

-- マクロ経済指標 (Macro Economic Indicators)
-- 国ごとの経済健全性を示す指標
CREATE TABLE IF NOT EXISTS macro_indicators (
    indicator_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    country_id INT NOT NULL,
    type ENUM('GDP_GROWTH', 'CPI', 'INTEREST_RATE', 'UNEMPLOYMENT') NOT NULL,
    value BIGINT NOT NULL COMMENT 'Scaled value: 550 = 5.50%',
    published_at BIGINT NOT NULL,
    FOREIGN KEY (country_id) REFERENCES countries(country_id),
    INDEX idx_macro_country_date (country_id, published_at)
);

-- 産業セクター (Sectors)
CREATE TABLE IF NOT EXISTS sectors (
    sector_id INT AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(20) UNIQUE NOT NULL, -- TECH, ENERGY, FIN...
    name VARCHAR(50) NOT NULL
);

-- 企業・発行体 (Companies / Issuers)
CREATE TABLE IF NOT EXISTS companies (
    company_id INT AUTO_INCREMENT PRIMARY KEY,
    country_id INT,
    sector_id INT,
    name VARCHAR(100) NOT NULL,
    ticker_symbol VARCHAR(10) UNIQUE NOT NULL,
    description TEXT,

    user_id BIGINT UNIQUE COMMENT 'Bot Account ID',

    -- Economic Simulation State
    max_production_capacity BIGINT DEFAULT 10000 COMMENT 'Max units per quarter',
    current_inventory BIGINT DEFAULT 0 COMMENT 'Current units in stock',
    last_capex_at BIGINT DEFAULT 0 COMMENT 'Timestamp of last expansion',

    -- Equity Financing State
    shares_issued BIGINT DEFAULT 1000000 COMMENT 'Total shares issued',
    shares_outstanding BIGINT DEFAULT 500000 COMMENT 'Shares outstanding',
    treasury_stock BIGINT DEFAULT 500000 COMMENT 'Treasury shares',

    FOREIGN KEY (country_id) REFERENCES countries(country_id),
    FOREIGN KEY (sector_id) REFERENCES sectors(sector_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);

-- 資源・コモディティ (Resources / Commodities)
CREATE TABLE IF NOT EXISTS resources (
    resource_id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    type VARCHAR(20) NOT NULL, -- ENERGY, METAL, FOOD, TECH, BASIC
    description TEXT
);

-- 企業ファンダメンタルズ (Financial Reports)
-- 四半期ごとの決算データ
CREATE TABLE IF NOT EXISTS financial_reports (
    report_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    company_id INT NOT NULL,
    fiscal_year INT NOT NULL,
    fiscal_quarter INT NOT NULL COMMENT '1, 2, 3, 4',
    
    revenue BIGINT DEFAULT 0 COMMENT 'Scaled currency (1.000000 = 1000000)',
    net_income BIGINT DEFAULT 0 COMMENT 'Scaled currency (1.000000 = 1000000)',
    eps BIGINT DEFAULT 0 COMMENT 'Earnings Per Share (1.000000 = 1000000)',
    
    -- Economic Metrics
    capex BIGINT DEFAULT 0 COMMENT 'Capital Expenditure this quarter (1.000000 = 1000000)',
    utilization_rate BIGINT DEFAULT 0 COMMENT 'Scaled: 10000 = 100.00%',
    inventory_level BIGINT DEFAULT 0 COMMENT 'Inventory at quarter end',

    guidance TEXT COMMENT 'Management guidance/outlook',
    published_at BIGINT NOT NULL,
    
    FOREIGN KEY (company_id) REFERENCES companies(company_id),
    UNIQUE(company_id, fiscal_year, fiscal_quarter)
);

-- 企業配当実績 (Company Dividend Records)
CREATE TABLE IF NOT EXISTS company_dividends (
    dividend_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    company_id INT NOT NULL,
    asset_id INT NOT NULL,
    fiscal_year INT NOT NULL,
    fiscal_quarter INT NOT NULL COMMENT '1, 2, 3, 4',
    net_income BIGINT DEFAULT 0,
    payout_ratio_bps BIGINT DEFAULT 0,
    dividend_per_share BIGINT DEFAULT 0,
    company_payout BIGINT DEFAULT 0,
    pool_payout BIGINT DEFAULT 0,
    spot_payout BIGINT DEFAULT 0,
    margin_long_payout BIGINT DEFAULT 0,
    margin_short_charge BIGINT DEFAULT 0,
    eligible_spot_shares BIGINT DEFAULT 0,
    eligible_long_shares BIGINT DEFAULT 0,
    pool_shares BIGINT DEFAULT 0,
    created_at BIGINT NOT NULL,

    FOREIGN KEY (company_id) REFERENCES companies(company_id),
    UNIQUE(company_id, fiscal_year, fiscal_quarter)
);

-- 資産マスタ (Tradable Assets: Stock, Bond, Index, etc.)
CREATE TABLE IF NOT EXISTS assets (
    asset_id INT AUTO_INCREMENT PRIMARY KEY,
    ticker VARCHAR(10) UNIQUE NOT NULL,
    company_id INT NULL,
    resource_id INT NULL,
    type ENUM('STOCK', 'BOND', 'INDEX', 'COMMODITY') NOT NULL,
    base_price BIGINT NOT NULL COMMENT 'Integer scaled price (1.000000 = 1000000)',
    lot_size INT DEFAULT 1,
    is_tradable BOOLEAN DEFAULT TRUE,
    created_at BIGINT DEFAULT 0,
    FOREIGN KEY (company_id) REFERENCES companies(company_id),
    FOREIGN KEY (resource_id) REFERENCES resources(resource_id)
);

-- 永久債パラメータ (Perpetual Bonds)
CREATE TABLE IF NOT EXISTS perpetual_bonds (
    asset_id INT PRIMARY KEY,
    issuer_country_id INT NOT NULL COMMENT '発行国',
    base_coupon BIGINT NOT NULL COMMENT '1単位あたりの固定利息額 (1.000000 = 1000000)',
    payment_frequency ENUM('DAILY', 'WEEKLY') DEFAULT 'WEEKLY' COMMENT '利払い頻度',
    
    FOREIGN KEY (asset_id) REFERENCES assets(asset_id),
    FOREIGN KEY (issuer_country_id) REFERENCES countries(country_id)
);

-- --------------------------------------------------------
-- 2. User & Accounts (プレイヤーデータ)
-- --------------------------------------------------------

-- ユーザー
-- user_id < 10000 はBotとして予約
CREATE TABLE IF NOT EXISTS users (
    user_id BIGINT PRIMARY KEY, -- Discord Snowflake ID (BIGINT)
    username VARCHAR(50) NOT NULL,
    rank ENUM('Shrimp', 'Fish', 'Shark', 'Whale', 'Leviathan') DEFAULT 'Shrimp',
    created_at BIGINT DEFAULT 0
);

-- APIキー管理
CREATE TABLE IF NOT EXISTS api_keys (
    api_key VARCHAR(40) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    role VARCHAR(50) NOT NULL,
    created_at BIGINT DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    UNIQUE(role)
);

-- 資産管理: 通貨残高 (Currency Balances)
CREATE TABLE IF NOT EXISTS currency_balances (
    balance_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    currency_id INT NOT NULL,
    amount BIGINT DEFAULT 0,
    updated_at BIGINT DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (currency_id) REFERENCES currencies(currency_id),
    UNIQUE(user_id, currency_id)
);

-- 入出金・資産変動ログ (Transaction Audit Logs)
-- 主にCash Flowを記録
CREATE TABLE IF NOT EXISTS transaction_logs (
    log_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    currency_id INT NOT NULL,
    amount BIGINT NOT NULL COMMENT 'Signed integer: +Deposit, -Withdrawal',
    balance_after BIGINT NOT NULL COMMENT 'Snapshot of balance after tx',
    
    type ENUM('DEPOSIT', 'WITHDRAW', 'TRADE_BUY', 'TRADE_SELL', 'FEE', 'DIVIDEND', 'INTEREST', 'TRANSFER', 'INSURANCE_PAYOUT') NOT NULL,
    reference_id VARCHAR(50) COMMENT 'Order ID or External Tx ID',
    description TEXT,
    
    created_at BIGINT DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (currency_id) REFERENCES currencies(currency_id),
    INDEX idx_user_logs (user_id, created_at)
);

-- 資産管理: 資産残高 (Asset Balances)
-- Stock, Bond, Index 等の保有量
CREATE TABLE IF NOT EXISTS asset_balances (
    balance_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    asset_id INT NOT NULL,
    quantity BIGINT DEFAULT 0,
    average_price BIGINT DEFAULT 0,
    average_acquired_at BIGINT DEFAULT 0 COMMENT 'Weighted average timestamp for dividend boost',
    updated_at BIGINT DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (asset_id) REFERENCES assets(asset_id),
    UNIQUE(user_id, asset_id)
);

-- ポジション (Leveraged/Margin Positions)
CREATE TABLE IF NOT EXISTS positions (
    position_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    season_id INT NOT NULL,
    asset_id INT NOT NULL,
    side ENUM('LONG', 'SHORT') NOT NULL,
    quantity BIGINT NOT NULL,
    entry_price BIGINT NOT NULL,
    current_price BIGINT NOT NULL COMMENT 'Last marked price',
    leverage BIGINT DEFAULT 100 COMMENT '100 = 1.00x',
    margin_used BIGINT DEFAULT 0,
    unrealized_pl BIGINT DEFAULT 0,
    created_at BIGINT DEFAULT 0,
    updated_at BIGINT DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (asset_id) REFERENCES assets(asset_id),
    FOREIGN KEY (season_id) REFERENCES seasons(season_id)
);

-- --------------------------------------------------------
-- 3. Trading System (取引エンジン)
-- --------------------------------------------------------

-- 注文 (Active & Historical Orders)
CREATE TABLE IF NOT EXISTS orders (
    order_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    asset_id INT NOT NULL,
    side ENUM('BUY', 'SELL') NOT NULL,
    type ENUM('MARKET', 'LIMIT', 'STOP', 'STOP_LIMIT') NOT NULL,
    time_in_force ENUM('GTC', 'IOC', 'FOK') DEFAULT 'GTC',
    
    quantity BIGINT NOT NULL,
    price BIGINT, -- Limit price (scaled)
    stop_price BIGINT, -- Stop trigger price (scaled)
    
    filled_quantity BIGINT DEFAULT 0,
    average_fill_price BIGINT DEFAULT 0,
    
    status ENUM('OPEN', 'PARTIAL', 'FILLED', 'CANCELLED', 'REJECTED') DEFAULT 'OPEN',
    
    created_at BIGINT DEFAULT 0,
    updated_at BIGINT DEFAULT 0,
    
    INDEX idx_order_book (asset_id, status, side, price),
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (asset_id) REFERENCES assets(asset_id)
);

-- 約定履歴 (Executions / Trade Tape)
CREATE TABLE IF NOT EXISTS executions (
    execution_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    buy_order_id BIGINT NOT NULL,
    sell_order_id BIGINT NOT NULL,
    asset_id INT NOT NULL,
    price BIGINT NOT NULL,
    quantity BIGINT NOT NULL,
    executed_at BIGINT DEFAULT 0,
    is_taker_buyer BOOLEAN,
    
    FOREIGN KEY (buy_order_id) REFERENCES orders(order_id),
    FOREIGN KEY (sell_order_id) REFERENCES orders(order_id)
);

-- ロウソク足データ (Market Candles)
CREATE TABLE IF NOT EXISTS market_candles (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    asset_id INT NOT NULL,
    timeframe ENUM('1M', '5M', '15M', '1H', '4H', '1D') NOT NULL,
    open_time BIGINT NOT NULL,
    open BIGINT NOT NULL,
    high BIGINT NOT NULL,
    low BIGINT NOT NULL,
    close BIGINT NOT NULL,
    volume BIGINT DEFAULT 0,
    
    UNIQUE(asset_id, timeframe, open_time),
    FOREIGN KEY (asset_id) REFERENCES assets(asset_id)
);

-- --------------------------------------------------------
-- 4. Events & News (イベント)
-- --------------------------------------------------------

-- ニュースフィード
CREATE TABLE IF NOT EXISTS news_feed (
    news_id INT AUTO_INCREMENT PRIMARY KEY,
    headline VARCHAR(255) NOT NULL,
    body TEXT,
    published_at BIGINT DEFAULT 0,
    source VARCHAR(50) DEFAULT 'Paper Street Wire',
    
    sentiment_score BIGINT DEFAULT 0 COMMENT 'Scaled: 100 = 1.00',
    category VARCHAR(50) DEFAULT '',
    impact VARCHAR(20) DEFAULT '',
    impact_scope TEXT,
    related_asset_id INT,
    related_sector_id INT,
    related_country_id INT
);

-- --------------------------------------------------------
-- 5. Liquidity Pools (FX Market)
-- --------------------------------------------------------

-- 流動性プール (Liquidity Pools for FX)
-- ARCを基軸通貨とし、各通貨とのペアを管理
CREATE TABLE IF NOT EXISTS liquidity_pools (
    pool_id INT AUTO_INCREMENT PRIMARY KEY,
    currency_id INT NOT NULL COMMENT 'The other currency paired with ARC',
    fee_tier_bp INT NOT NULL DEFAULT 20 COMMENT 'Fee Tier in basis points (e.g., 20 = 0.20%, 4 = 0.04%)',
    current_tick INT NOT NULL DEFAULT 0,
    tick_spacing INT NOT NULL DEFAULT 1,
    liquidity BIGINT DEFAULT 0,

    -- Fee tracking (Global)
    fee_growth_global_0 BIGINT DEFAULT 0,
    fee_growth_global_1 BIGINT DEFAULT 0,

    created_at BIGINT DEFAULT 0,

    FOREIGN KEY (currency_id) REFERENCES currencies(currency_id),
    UNIQUE(currency_id, fee_tier_bp)
);

-- 流動性ポジション / 指値注文 (Liquidity Positions / Limit Orders)
CREATE TABLE IF NOT EXISTS liquidity_positions (
    position_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    pool_id INT NOT NULL,
    user_id BIGINT NOT NULL,

    tick_lower INT NOT NULL,
    tick_upper INT NOT NULL,

    liquidity BIGINT DEFAULT 0,

    -- Fee tracking (Inside)
    fee_growth_inside_0_last BIGINT DEFAULT 0,
    fee_growth_inside_1_last BIGINT DEFAULT 0,
    tokens_owed_0 BIGINT DEFAULT 0,
    tokens_owed_1 BIGINT DEFAULT 0,

    -- Limit Order Specifics
    is_limit_order BOOLEAN DEFAULT FALSE,
    status ENUM('ACTIVE', 'FILLED', 'CLOSED', 'WITHDRAWN') DEFAULT 'ACTIVE',

    created_at BIGINT DEFAULT 0,
    updated_at BIGINT DEFAULT 0,

    FOREIGN KEY (pool_id) REFERENCES liquidity_pools(pool_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);

-- --------------------------------------------------------
-- 6. Margin Trading Pools (Dual Liquidity Inventory)
-- --------------------------------------------------------

-- 信用取引流動性プール (Margin Pools)
-- 各銘柄(Asset)ごとの現金在庫と現物在庫を管理
CREATE TABLE IF NOT EXISTS margin_pools (
    pool_id INT AUTO_INCREMENT PRIMARY KEY,
    asset_id INT NOT NULL COMMENT 'The asset being traded',
    currency_id INT NOT NULL COMMENT 'The quote currency (e.g. ARC)',
    
    -- Cash Vault (Currency Inventory)
    total_cash BIGINT DEFAULT 0 COMMENT 'Total cash liquidity available',
    borrowed_cash BIGINT DEFAULT 0 COMMENT 'Cash borrowed by long positions',
    
    -- Asset Vault (Asset Inventory)
    total_assets BIGINT DEFAULT 0 COMMENT 'Total asset liquidity available',
    borrowed_assets BIGINT DEFAULT 0 COMMENT 'Assets borrowed by short positions',
    
    -- Interest Rates (Snapshot/Current)
    borrow_rate BIGINT DEFAULT 0 COMMENT 'Long interest rate (Cost to borrow cash)',
    short_fee BIGINT DEFAULT 0 COMMENT 'Short fee rate (Cost to borrow asset)',
    
    -- Share Tokens (Lending System)
    total_cash_shares BIGINT DEFAULT 0 COMMENT 'Total shares issued to cash lenders',
    total_asset_shares BIGINT DEFAULT 0 COMMENT 'Total shares issued to asset lenders',

    updated_at BIGINT DEFAULT 0,
    
    FOREIGN KEY (asset_id) REFERENCES assets(asset_id),
    FOREIGN KEY (currency_id) REFERENCES currencies(currency_id),
    UNIQUE(asset_id, currency_id)
);

-- 流動性提供者 (Liquidity Providers for Margin Pools)
CREATE TABLE IF NOT EXISTS margin_pool_providers (
    provider_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    pool_id INT NOT NULL,
    user_id BIGINT NOT NULL,

    cash_shares BIGINT DEFAULT 0,
    asset_shares BIGINT DEFAULT 0,

    updated_at BIGINT DEFAULT 0,

    FOREIGN KEY (pool_id) REFERENCES margin_pools(pool_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    UNIQUE(pool_id, user_id)
);

-- --------------------------------------------------------
-- 7. Indices (Simple Physical Basket)
-- --------------------------------------------------------

-- インデックス構成銘柄 (Index Constituents)
CREATE TABLE IF NOT EXISTS index_constituents (
    id INT AUTO_INCREMENT PRIMARY KEY,
    index_asset_id INT NOT NULL,
    component_asset_id INT NOT NULL,
    -- weight removed: implicitly 1 unit each
    FOREIGN KEY (index_asset_id) REFERENCES assets(asset_id),
    FOREIGN KEY (component_asset_id) REFERENCES assets(asset_id),
    UNIQUE(index_asset_id, component_asset_id)
);

-- --------------------------------------------------------
-- 7.5. Production & Supply Chain
-- --------------------------------------------------------

-- 生産レシピ (Production Recipes)
CREATE TABLE IF NOT EXISTS production_recipes (
    recipe_id INT AUTO_INCREMENT PRIMARY KEY,
    company_id INT NOT NULL,
    output_asset_id INT NOT NULL, -- 生産されるコモディティ
    output_quantity BIGINT NOT NULL DEFAULT 1,

    FOREIGN KEY (company_id) REFERENCES companies(company_id),
    FOREIGN KEY (output_asset_id) REFERENCES assets(asset_id)
);

-- レシピの材料 (Production Inputs)
CREATE TABLE IF NOT EXISTS production_inputs (
    input_id INT AUTO_INCREMENT PRIMARY KEY,
    recipe_id INT NOT NULL,
    input_asset_id INT NOT NULL, -- 原材料となるコモディティ
    input_quantity BIGINT NOT NULL, -- output 1単位を作るのに必要な量

    FOREIGN KEY (recipe_id) REFERENCES production_recipes(recipe_id),
    FOREIGN KEY (input_asset_id) REFERENCES assets(asset_id)
);

-- --------------------------------------------------------
-- 7.6. Corporate Contracts (Supply Chain Quests)
-- --------------------------------------------------------

-- コントラクト定義 (Contract Definitions)
CREATE TABLE IF NOT EXISTS contracts (
    contract_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    
    -- Issuer (Corporate or Government)
    company_id INT NULL,
    country_id INT NULL,
    
    title VARCHAR(255) NOT NULL,
    description TEXT,
    
    target_asset_id INT NOT NULL,
    total_required_quantity BIGINT NOT NULL,
    current_delivered_quantity BIGINT DEFAULT 0,
    
    unit_price BIGINT NOT NULL COMMENT 'Fixed buying price per unit',
    xp_reward_per_unit INT NOT NULL DEFAULT 0,
    min_rank_required ENUM('Shrimp', 'Fish', 'Shark', 'Whale', 'Leviathan') DEFAULT 'Shark',
    
    status ENUM('ACTIVE', 'COMPLETED', 'EXPIRED') DEFAULT 'ACTIVE',
    
    start_at BIGINT NOT NULL,
    expires_at BIGINT NOT NULL,
    
    FOREIGN KEY (company_id) REFERENCES companies(company_id),
    FOREIGN KEY (country_id) REFERENCES countries(country_id),
    FOREIGN KEY (target_asset_id) REFERENCES assets(asset_id)
);

-- 納品履歴 (Contract Deliveries)
CREATE TABLE IF NOT EXISTS contract_deliveries (
    delivery_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    contract_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    
    quantity BIGINT NOT NULL,
    payout_amount BIGINT NOT NULL,
    xp_gained INT NOT NULL,
    
    delivered_at BIGINT DEFAULT 0,
    
    FOREIGN KEY (contract_id) REFERENCES contracts(contract_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);

SET FOREIGN_KEY_CHECKS = 1;

-- --------------------------------------------------------
-- 8. Initial System Data
-- --------------------------------------------------------

-- System Accounts
-- user_id=1: Insurance Fund (Receives fees, Covers bankruptcies)
INSERT INTO users (user_id, username, rank, created_at) VALUES (1, 'Paper Street Insurance Fund', 'Leviathan', 0);
