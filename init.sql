-- ==========================================
-- Paper Street: Initialization SQL
-- ==========================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

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
    currency_code VARCHAR(3) NOT NULL, -- e.g., VND, ARC
    FOREIGN KEY (region_id) REFERENCES regions(region_id)
);

-- マクロ経済指標 (Macro Economic Indicators)
-- 国ごとの経済健全性を示す指標
CREATE TABLE IF NOT EXISTS macro_indicators (
    indicator_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    country_id INT NOT NULL,
    type ENUM('GDP_GROWTH', 'CPI', 'INTEREST_RATE', 'UNEMPLOYMENT') NOT NULL,
    value DECIMAL(21, 0) NOT NULL COMMENT 'Scaled value: 550 = 5.50%',
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
    FOREIGN KEY (country_id) REFERENCES countries(country_id),
    FOREIGN KEY (sector_id) REFERENCES sectors(sector_id)
);

-- 企業ファンダメンタルズ (Financial Reports)
-- 四半期ごとの決算データ
CREATE TABLE IF NOT EXISTS financial_reports (
    report_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    company_id INT NOT NULL,
    fiscal_year INT NOT NULL,
    fiscal_quarter INT NOT NULL COMMENT '1, 2, 3, 4',
    
    revenue DECIMAL(21, 0) DEFAULT 0 COMMENT 'Scaled currency',
    net_income DECIMAL(21, 0) DEFAULT 0 COMMENT 'Scaled currency',
    eps DECIMAL(21, 0) DEFAULT 0 COMMENT 'Earnings Per Share (Scaled)',
    
    guidance TEXT COMMENT 'Management guidance/outlook',
    published_at BIGINT NOT NULL,
    
    FOREIGN KEY (company_id) REFERENCES companies(company_id),
    UNIQUE(company_id, fiscal_year, fiscal_quarter)
);

-- 金融商品マスタ (Tradable Instruments)
CREATE TABLE IF NOT EXISTS instruments (
    instrument_id INT AUTO_INCREMENT PRIMARY KEY,
    ticker VARCHAR(10) UNIQUE NOT NULL,
    company_id INT NULL,
    type ENUM('STOCK', 'BOND', 'CURRENCY', 'CRYPTO', 'INDEX') NOT NULL,
    base_price DECIMAL(21, 0) NOT NULL COMMENT 'Integer scaled price',
    lot_size INT DEFAULT 1,
    is_tradable BOOLEAN DEFAULT TRUE,
    created_at BIGINT DEFAULT 0,
    FOREIGN KEY (company_id) REFERENCES companies(company_id)
);

-- --------------------------------------------------------
-- 2. User & Accounts (プレイヤーデータ)
-- --------------------------------------------------------

-- ユーザー (Human & Bots)
-- user_id < 10000 はBotとして予約
CREATE TABLE IF NOT EXISTS users (
    user_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    country_id INT,
    fee_tier ENUM('STANDARD', 'PREMIUM', 'VIP') DEFAULT 'STANDARD' COMMENT 'Commission rate tier',
    created_at BIGINT DEFAULT 0,
    FOREIGN KEY (country_id) REFERENCES countries(country_id)
) AUTO_INCREMENT = 10001; -- Human users start from 10001

-- 資産管理: 通貨 (Physical Currency Assets)
-- 法定通貨の数量: データ上 1000 = 表示 1.000
CREATE TABLE IF NOT EXISTS assets_currency (
    asset_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    currency_code VARCHAR(3) NOT NULL, -- USD, JPY, VND...
    amount DECIMAL(21, 0) DEFAULT 0,
    updated_at BIGINT DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    UNIQUE(user_id, currency_code)
);

-- 入出金・資産変動ログ (Transaction Audit Logs)
-- ユーザーの資産が増減したすべての理由を記録
CREATE TABLE IF NOT EXISTS transaction_logs (
    log_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    currency_code VARCHAR(3) NOT NULL,
    amount DECIMAL(21, 0) NOT NULL COMMENT 'Signed integer: +Deposit, -Withdrawal',
    balance_after DECIMAL(21, 0) NOT NULL COMMENT 'Snapshot of balance after tx',
    
    type ENUM('DEPOSIT', 'WITHDRAW', 'TRADE_BUY', 'TRADE_SELL', 'FEE', 'TAX', 'DIVIDEND', 'INTEREST') NOT NULL,
    reference_id VARCHAR(50) COMMENT 'Order ID or External Tx ID',
    description TEXT,
    
    created_at BIGINT DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    INDEX idx_user_logs (user_id, created_at)
);

-- 資産管理: 株式 (Stock Assets - Spot)
CREATE TABLE IF NOT EXISTS assets_stock (
    asset_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    instrument_id INT NOT NULL,
    quantity DECIMAL(21, 0) DEFAULT 0,
    average_price DECIMAL(21, 0) DEFAULT 0,
    updated_at BIGINT DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (instrument_id) REFERENCES instruments(instrument_id),
    UNIQUE(user_id, instrument_id)
);

-- 資産管理: 債券 (Bond Assets - Spot)
CREATE TABLE IF NOT EXISTS assets_bond (
    asset_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    instrument_id INT NOT NULL,
    quantity DECIMAL(21, 0) DEFAULT 0,
    average_price DECIMAL(21, 0) DEFAULT 0,
    updated_at BIGINT DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (instrument_id) REFERENCES instruments(instrument_id),
    UNIQUE(user_id, instrument_id)
);

-- 資産管理: 暗号資産などその他 (Crypto/Index Assets - Spot)
CREATE TABLE IF NOT EXISTS assets_crypto (
    asset_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    instrument_id INT NOT NULL,
    quantity DECIMAL(21, 0) DEFAULT 0,
    average_price DECIMAL(21, 0) DEFAULT 0,
    updated_at BIGINT DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (instrument_id) REFERENCES instruments(instrument_id),
    UNIQUE(user_id, instrument_id)
);

-- ポジション (Leveraged/Margin Positions)
CREATE TABLE IF NOT EXISTS positions (
    position_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    season_id INT NOT NULL,
    instrument_id INT NOT NULL,
    side ENUM('LONG', 'SHORT') NOT NULL,
    quantity DECIMAL(21, 0) NOT NULL,
    entry_price DECIMAL(21, 0) NOT NULL,
    current_price DECIMAL(21, 0) NOT NULL COMMENT 'Last marked price',
    leverage DECIMAL(21, 0) DEFAULT 100 COMMENT '100 = 1.00x',
    margin_used DECIMAL(21, 0) DEFAULT 0,
    unrealized_pl DECIMAL(21, 0) DEFAULT 0,
    created_at BIGINT DEFAULT 0,
    updated_at BIGINT DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (instrument_id) REFERENCES instruments(instrument_id),
    FOREIGN KEY (season_id) REFERENCES seasons(season_id)
);

-- --------------------------------------------------------
-- 3. Trading System (取引エンジン)
-- --------------------------------------------------------

-- 注文 (Active & Historical Orders)
CREATE TABLE IF NOT EXISTS orders (
    order_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    instrument_id INT NOT NULL,
    side ENUM('BUY', 'SELL') NOT NULL,
    type ENUM('MARKET', 'LIMIT', 'STOP', 'STOP_LIMIT') NOT NULL,
    
    quantity DECIMAL(21, 0) NOT NULL,
    price DECIMAL(21, 0), -- Limit price (scaled)
    
    filled_quantity DECIMAL(21, 0) DEFAULT 0,
    average_fill_price DECIMAL(21, 0) DEFAULT 0,
    
    status ENUM('OPEN', 'PARTIAL', 'FILLED', 'CANCELLED', 'REJECTED') DEFAULT 'OPEN',
    
    created_at BIGINT DEFAULT 0,
    updated_at BIGINT DEFAULT 0,
    
    INDEX idx_order_book (instrument_id, status, side, price),
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);

-- 約定履歴 (Executions / Trade Tape)
CREATE TABLE IF NOT EXISTS executions (
    execution_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    buy_order_id BIGINT NOT NULL,
    sell_order_id BIGINT NOT NULL,
    instrument_id INT NOT NULL,
    price DECIMAL(21, 0) NOT NULL,
    quantity DECIMAL(21, 0) NOT NULL,
    executed_at BIGINT DEFAULT 0,
    is_taker_buyer BOOLEAN,
    
    FOREIGN KEY (buy_order_id) REFERENCES orders(order_id),
    FOREIGN KEY (sell_order_id) REFERENCES orders(order_id)
);

-- ロウソク足データ (Market Candles)
CREATE TABLE IF NOT EXISTS market_candles (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    instrument_id INT NOT NULL,
    timeframe ENUM('1M', '5M', '15M', '1H', '4H', '1D') NOT NULL,
    open_time BIGINT NOT NULL,
    open DECIMAL(21, 0) NOT NULL,
    high DECIMAL(21, 0) NOT NULL,
    low DECIMAL(21, 0) NOT NULL,
    close DECIMAL(21, 0) NOT NULL,
    volume DECIMAL(21, 0) DEFAULT 0,
    
    UNIQUE(instrument_id, timeframe, open_time)
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
    
    sentiment_score DECIMAL(21, 0) DEFAULT 0 COMMENT 'Scaled: 100 = 1.00',
    related_instrument_id INT,
    related_sector_id INT,
    related_country_id INT
);

SET FOREIGN_KEY_CHECKS = 1;
