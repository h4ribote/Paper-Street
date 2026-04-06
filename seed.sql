-- ==========================================
-- Paper Street: Seed Data SQL
-- docs/design/tradable_assets.md / docs/world_setting.md 準拠
-- ==========================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- --------------------------------------------------------
-- 1) Regions / Countries / Currencies
-- --------------------------------------------------------

INSERT INTO regions (region_id, name, description) VALUES
    (1, 'Northern Alliance', 'Advanced markets with high-tech leadership and aging demographics.'),
    (2, 'Eastern Coalition', 'Industrial powerhouse with state-led growth and export-driven policy.'),
    (3, 'Southern Resource Pact', 'Resource-rich bloc with high commodity exposure and political risk.'),
    (4, 'Oceanic Tech Arch', 'Financial hubs and tax havens fueling volatile innovation.')
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    description = VALUES(description);

INSERT INTO countries (country_id, region_id, name)
SELECT 1, 1, 'Arcadia'
WHERE NOT EXISTS (SELECT 1 FROM countries WHERE country_id = 1 OR name = 'Arcadia');

INSERT INTO countries (country_id, region_id, name)
SELECT 2, 2, 'Boros Federation'
WHERE NOT EXISTS (SELECT 1 FROM countries WHERE country_id = 2 OR name = 'Boros Federation');

INSERT INTO countries (country_id, region_id, name)
SELECT 3, 3, 'El Dorado'
WHERE NOT EXISTS (SELECT 1 FROM countries WHERE country_id = 3 OR name = 'El Dorado');

INSERT INTO countries (country_id, region_id, name)
SELECT 4, 4, 'Neo Venice'
WHERE NOT EXISTS (SELECT 1 FROM countries WHERE country_id = 4 OR name = 'Neo Venice');

INSERT INTO countries (country_id, region_id, name)
SELECT 5, 3, 'San Verde'
WHERE NOT EXISTS (SELECT 1 FROM countries WHERE country_id = 5 OR name = 'San Verde');

INSERT INTO countries (country_id, region_id, name)
SELECT 6, 1, 'Novaya Zemlya'
WHERE NOT EXISTS (SELECT 1 FROM countries WHERE country_id = 6 OR name = 'Novaya Zemlya');

INSERT INTO countries (country_id, region_id, name)
SELECT 7, 2, 'Pearl River Zone'
WHERE NOT EXISTS (SELECT 1 FROM countries WHERE country_id = 7 OR name = 'Pearl River Zone');

INSERT INTO currencies (currency_id, country_id, code, name) VALUES
    (1, 1, 'ARC', 'Arcadian Credit'),
    (2, 2, 'BRB', 'Boros Ruble'),
    (3, 3, 'DRL', 'Dorado Real'),
    (4, 4, 'VND', 'Venice Dollar'),
    (5, 5, 'VDP', 'Verde Peso'),
    (6, 6, 'ZMR', 'Zemlya Ruble'),
    (7, 7, 'RVD', 'River Dollar')
ON DUPLICATE KEY UPDATE
    country_id = VALUES(country_id),
    code = VALUES(code),
    name = VALUES(name);

-- --------------------------------------------------------
-- 2) Sectors
-- --------------------------------------------------------

INSERT INTO sectors (sector_id, code, name) VALUES
    (1, 'TECH', 'TECH'),
    (2, 'ENERGY', 'ENERGY'),
    (3, 'FIN', 'FIN'),
    (4, 'BIO', 'BIO'),
    (5, 'CONS', 'CONS'),
    (6, 'DEF', 'DEF'),
    (7, 'LOG', 'LOG'),
    (8, 'METAL', 'METAL'),
    (9, 'FOOD', 'FOOD')
ON DUPLICATE KEY UPDATE
    code = VALUES(code),
    name = VALUES(name);

-- --------------------------------------------------------
-- 3) Companies (Stocks)
-- NOTE: last_capex_at = 0 は「未実施」を示す規定仕様。
-- --------------------------------------------------------

INSERT INTO companies (
    company_id, country_id, sector_id, name, ticker_symbol, description, user_id,
    max_production_capacity, current_inventory, last_capex_at,
    shares_issued, shares_outstanding, treasury_stock
) VALUES
    (101, 1, 1, 'OmniCorp', 'OMNI', 'World-leading AI and infrastructure company.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (102, 2, 2, 'Nyx Energy', 'NYX', 'Legacy seed-compatible energy issuer.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (104, 1, 1, 'Quantum Dynamics', 'QDY', 'Quantum computing hardware manufacturer.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (105, 4, 1, 'CyberLife', 'CYB', 'Cybernetics and implant leader.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (106, 7, 1, 'Silicon Dragon', 'SLD', 'Global semiconductor champion.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (107, 2, 2, 'Titan Energy', 'TTN', 'Fossil + transition energy supplier.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (108, 6, 2, 'Nebula Mining', 'NEB', 'Space mining venture.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (109, 3, 2, 'Helios Solar', 'SOL', 'Large-scale solar generation.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (110, 6, 2, 'Atomos Energy', 'ATM', 'Next-gen nuclear and geothermal.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (111, 1, 3, 'Goliath Bank', 'GLT', 'Global investment bank.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (112, 4, 3, 'Shadow Fund', 'SHD', 'Opaque algorithmic hedge fund.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (113, 1, 4, 'Chimera Genetics', 'CHM', 'Gene-editing pioneer.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (114, 5, 4, 'Verde Pharma', 'VPH', 'Low-cost pharma maker.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (115, 3, 4, 'Panacea Corp', 'PAN', 'High-value specialty therapeutics.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (116, 4, 5, 'Stardust Luxury', 'SDL', 'Ultra-luxury consumer brand.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (117, 5, 5, 'Red Ox Food', 'ROX', 'Global food producer and trader.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (118, 4, 5, 'Global News Network', 'GNN', 'Media conglomerate.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (119, 2, 6, 'Iron Fist Armaments', 'IFA', 'State-linked defense manufacturer.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (120, 1, 6, 'Aegis Systems', 'AEG', 'Autonomous defense systems.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (121, 4, 7, 'Trans-Oceanic', 'TRN', 'Maritime logistics operator.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (122, 1, 7, 'Void Cargo', 'VDC', 'Orbital transport venture.', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (123, 1, 7, 'Horizon Logistics', 'HRZ', 'Drone logistics network.', NULL, 10000, 0, 0, 1000000, 500000, 500000)
ON DUPLICATE KEY UPDATE
    country_id = VALUES(country_id),
    sector_id = VALUES(sector_id),
    name = VALUES(name),
    ticker_symbol = VALUES(ticker_symbol),
    description = VALUES(description),
    max_production_capacity = VALUES(max_production_capacity),
    current_inventory = VALUES(current_inventory),
    last_capex_at = VALUES(last_capex_at),
    shares_issued = VALUES(shares_issued),
    shares_outstanding = VALUES(shares_outstanding),
    treasury_stock = VALUES(treasury_stock);

-- --------------------------------------------------------
-- 4) Resources / Commodities
-- --------------------------------------------------------

INSERT INTO resources (resource_id, name, type, description) VALUES
    (1,  'Crude Oil',         'ENERGY', 'Core fossil fuel.'),
    (2,  'Natural Gas',       'ENERGY', 'Power and heating fuel.'),
    (3,  'Hydrogen',          'ENERGY', 'Clean energy carrier.'),
    (4,  'Uranium',           'ENERGY', 'Nuclear fuel.'),
    (5,  'Energy Credits',    'ENERGY', 'Power certificates.'),
    (6,  'Rare Earth Metals', 'METAL',  'Critical minerals.'),
    (7,  'Lithium',           'METAL',  'Battery material.'),
    (8,  'Steel',             'METAL',  'Industrial base material.'),
    (9,  'Water',             'BASIC',  'Industrial and potable water.'),
    (10, 'Grain',             'FOOD',   'Staple agriculture output.'),
    (11, 'Synthetic Meat',    'FOOD',   'Cultured protein.'),
    (12, 'Semiconductors',    'TECH',   'Electronic core component.'),
    (13, 'Cyber-Implants',    'TECH',   'Cybernetic components.'),
    (14, 'Bio-Gel',           'TECH',   'Regenerative medical material.')
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    type = VALUES(type),
    description = VALUES(description);

-- --------------------------------------------------------
-- 5) Assets (Stocks / Bonds / Indices / Commodities)
-- NOTE: created_at = 0 は「未設定/未初期化」の番兵値として利用する。1970-01-01を意味する業務時刻としては扱わない。実際の生成時刻はアプリ側永続化で現在時刻(ms)が設定される。
-- --------------------------------------------------------

INSERT INTO assets (asset_id, ticker, company_id, resource_id, type, base_price, lot_size, is_tradable, created_at) VALUES
    -- Stocks
    (101, 'OMNI', 101, NULL, 'STOCK', 15250, 1, TRUE, 0),
    (102, 'NYX',  102, NULL, 'STOCK',  9825, 1, TRUE, 0),
    (104, 'QDY',  104, NULL, 'STOCK', 12600, 1, TRUE, 0),
    (105, 'CYB',  105, NULL, 'STOCK', 11800, 1, TRUE, 0),
    (106, 'SLD',  106, NULL, 'STOCK', 11000, 1, TRUE, 0),
    (107, 'TTN',  107, NULL, 'STOCK',  9600, 1, TRUE, 0),
    (108, 'NEB',  108, NULL, 'STOCK', 10100, 1, TRUE, 0),
    (109, 'SOL',  109, NULL, 'STOCK',  9300, 1, TRUE, 0),
    (110, 'ATM',  110, NULL, 'STOCK',  9900, 1, TRUE, 0),
    (111, 'GLT',  111, NULL, 'STOCK', 13500, 1, TRUE, 0),
    (112, 'SHD',  112, NULL, 'STOCK', 12200, 1, TRUE, 0),
    (113, 'CHM',  113, NULL, 'STOCK', 11600, 1, TRUE, 0),
    (114, 'VPH',  114, NULL, 'STOCK',  8700, 1, TRUE, 0),
    (115, 'PAN',  115, NULL, 'STOCK', 10800, 1, TRUE, 0),
    (116, 'SDL',  116, NULL, 'STOCK', 10400, 1, TRUE, 0),
    (117, 'ROX',  117, NULL, 'STOCK',  9200, 1, TRUE, 0),
    (118, 'GNN',  118, NULL, 'STOCK',  8800, 1, TRUE, 0),
    (119, 'IFA',  119, NULL, 'STOCK', 12500, 1, TRUE, 0),
    (120, 'AEG',  120, NULL, 'STOCK', 13000, 1, TRUE, 0),
    (121, 'TRN',  121, NULL, 'STOCK',  9100, 1, TRUE, 0),
    (122, 'VDC',  122, NULL, 'STOCK',  9500, 1, TRUE, 0),
    (123, 'HRZ',  123, NULL, 'STOCK',  9050, 1, TRUE, 0),

    -- Commodities (legacy compatibility + docs set)
    (103, 'AUR',  NULL, 6,  'COMMODITY', 18750, 1, TRUE, 0),
    (601, 'OIL',  NULL, 1,  'COMMODITY',  7200, 1, TRUE, 0),
    (602, 'NGS',  NULL, 2,  'COMMODITY',  5100, 1, TRUE, 0),
    (603, 'H2',   NULL, 3,  'COMMODITY',  8400, 1, TRUE, 0),
    (604, 'URN',  NULL, 4,  'COMMODITY',  9800, 1, TRUE, 0),
    (605, 'ECR',  NULL, 5,  'COMMODITY',  7600, 1, TRUE, 0),
    (606, 'REM',  NULL, 6,  'COMMODITY', 11200, 1, TRUE, 0),
    (607, 'LTH',  NULL, 7,  'COMMODITY', 10900, 1, TRUE, 0),
    (608, 'STL',  NULL, 8,  'COMMODITY',  6800, 1, TRUE, 0),
    (609, 'WTR',  NULL, 9,  'COMMODITY',  4700, 1, TRUE, 0),
    (610, 'GRN',  NULL, 10, 'COMMODITY',  5300, 1, TRUE, 0),
    (611, 'SMT',  NULL, 11, 'COMMODITY',  7600, 1, TRUE, 0),
    (612, 'SEM',  NULL, 12, 'COMMODITY', 11800, 1, TRUE, 0),
    (613, 'CYI',  NULL, 13, 'COMMODITY', 12400, 1, TRUE, 0),
    (614, 'BGL',  NULL, 14, 'COMMODITY', 11300, 1, TRUE, 0),

    -- Indices
    (201, 'TRI',   NULL, NULL, 'INDEX', 14600, 1, TRUE, 0),
    (401, 'PSI10', NULL, NULL, 'INDEX', 14000, 1, TRUE, 0),
    (402, 'TCH',   NULL, NULL, 'INDEX', 14500, 1, TRUE, 0),
    (403, 'EGY',   NULL, NULL, 'INDEX', 13200, 1, TRUE, 0),
    (404, 'BIO',   NULL, NULL, 'INDEX', 12800, 1, TRUE, 0),
    (405, 'RSC',   NULL, NULL, 'INDEX', 12000, 1, TRUE, 0),

    -- Perpetual bonds
    (301, 'ARCB',  NULL, NULL, 'BOND', 10000, 1, TRUE, 0),
    (302, 'BRSB',  NULL, NULL, 'BOND', 10000, 1, TRUE, 0),
    (303, 'SVDB',  NULL, NULL, 'BOND', 10000, 1, TRUE, 0)
ON DUPLICATE KEY UPDATE
    ticker = VALUES(ticker),
    company_id = VALUES(company_id),
    resource_id = VALUES(resource_id),
    type = VALUES(type),
    base_price = VALUES(base_price),
    lot_size = VALUES(lot_size),
    is_tradable = VALUES(is_tradable);

-- --------------------------------------------------------
-- 6) Perpetual Bonds
-- --------------------------------------------------------

INSERT INTO perpetual_bonds (asset_id, issuer_country_id, base_coupon, payment_frequency) VALUES
    (301, 1,  250, 'WEEKLY'),
    (302, 2,  500, 'WEEKLY'),
    (303, 5, 1000, 'WEEKLY')
ON DUPLICATE KEY UPDATE
    issuer_country_id = VALUES(issuer_country_id),
    base_coupon = VALUES(base_coupon),
    payment_frequency = VALUES(payment_frequency);

-- --------------------------------------------------------
-- 7) Index Constituents
-- --------------------------------------------------------

INSERT IGNORE INTO index_constituents (index_asset_id, component_asset_id) VALUES
    -- Legacy TriCore
    (201, 101), (201, 102), (201, 103),
    -- PSI10
    (401, 101), (401, 104), (401, 105), (401, 106), (401, 107),
    (401, 108), (401, 109), (401, 110), (401, 111), (401, 112),
    -- TCH
    (402, 101), (402, 104), (402, 105), (402, 106),
    -- EGY
    (403, 102), (403, 107), (403, 108), (403, 109), (403, 110),
    -- BIO
    (404, 113), (404, 114), (404, 115),
    -- RSC
    (405, 601), (405, 602), (405, 603), (405, 606), (405, 607), (405, 608), (405, 610), (405, 611);

-- --------------------------------------------------------
-- 8) World Events / Season baseline
-- --------------------------------------------------------

-- NOTE: seasons.start_at/end_at と world_events.starts_at/ends_at の 0 は
-- 「未スケジュール/未初期化」を示す番兵値。UNIX epoch時刻としては扱わない。

INSERT INTO seasons (season_id, name, theme_code, start_at, end_at, is_active)
SELECT 1, 'Season 1: The Great Resurgence', 'RECOVERY', 0, 0, TRUE
WHERE NOT EXISTS (SELECT 1 FROM seasons WHERE season_id = 1);

INSERT INTO world_events (event_id, name, description, starts_at, ends_at)
SELECT 2, 'Tech Bubble Burst', 'Accounting irregularities trigger a broad selloff in tech names.', 0, 0
WHERE NOT EXISTS (SELECT 1 FROM world_events WHERE event_id = 2 OR name = 'Tech Bubble Burst');

INSERT INTO world_events (event_id, name, description, starts_at, ends_at)
SELECT 3, 'Resource War', 'El Dorado export restrictions trigger global supply shock.', 0, 0
WHERE NOT EXISTS (SELECT 1 FROM world_events WHERE event_id = 3 OR name = 'Resource War');

INSERT INTO world_events (event_id, name, description, starts_at, ends_at)
SELECT 4, 'Digital Currency Crisis', 'Major exchange hack causes risk-off and liquidity freeze.', 0, 0
WHERE NOT EXISTS (SELECT 1 FROM world_events WHERE event_id = 4 OR name = 'Digital Currency Crisis');

INSERT INTO world_events (event_id, name, description, starts_at, ends_at)
SELECT 5, 'Boros Election', 'Election outcome shifts defense spending and trade policy outlook.', 0, 0
WHERE NOT EXISTS (SELECT 1 FROM world_events WHERE event_id = 5 OR name = 'Boros Election');

INSERT INTO world_events (event_id, name, description, starts_at, ends_at)
SELECT 6, 'Arcadia Privacy Act', 'Strict privacy regulation pressures data-driven business models.', 0, 0
WHERE NOT EXISTS (SELECT 1 FROM world_events WHERE event_id = 6 OR name = 'Arcadia Privacy Act');

INSERT INTO world_events (event_id, name, description, starts_at, ends_at)
SELECT 7, 'El Dorado Succession', 'Succession tensions raise civil unrest and currency volatility risks.', 0, 0
WHERE NOT EXISTS (SELECT 1 FROM world_events WHERE event_id = 7 OR name = 'El Dorado Succession');


-- --------------------------------------------------------
-- 9) Bot auth roles (users / api_keys)
-- --------------------------------------------------------
-- DBモードではサーバーがrole/api_keyを自動生成しないため、
-- /auth/bot 利用ボットのロールをseedで事前投入する。
-- users.rank_id=1 は既定のエントリーランクを示し、created_at=0 は
-- 未設定/初期状態の番兵値（業務時刻としてのUNIX epochではない）。

INSERT INTO users (user_id, username, rank_id, created_at) VALUES
    (1000, 'Market Maker', 1, 0),
    (1001, 'Liquidity Provider', 1, 0),
    (1002, 'Whale Northern', 1, 0),
    (1003, 'Whale Oceanic', 1, 0),
    (1004, 'Whale Energy', 1, 0),
    (1005, 'National AI Arcadia', 1, 0),
    (1006, 'National AI Boros', 1, 0),
    (1007, 'National AI El Dorado', 1, 0),
    (1008, 'National AI Neo Venice', 1, 0),
    (1009, 'National AI San Verde', 1, 0),
    (1010, 'National AI Novaya', 1, 0),
    (1011, 'National AI Pearl River', 1, 0),
    (1012, 'momentum chaser a', 1, 0),
    (1013, 'momentum chaser b', 1, 0),
    (1014, 'momentum chaser c', 1, 0),
    (1015, 'dip buyer a', 1, 0),
    (1016, 'dip buyer b', 1, 0),
    (1017, 'dip buyer c', 1, 0),
    (1018, 'reversal sniper a', 1, 0),
    (1019, 'reversal sniper b', 1, 0),
    (1020, 'reversal sniper c', 1, 0),
    (1021, 'grid trader a', 1, 0),
    (1022, 'grid trader b', 1, 0),
    (1023, 'grid trader c', 1, 0),
    (1024, 'news reactor', 1, 0),
    (1025, 'arbitrageur', 1, 0),
    (1026, 'yield hunter', 1, 0),
    (1027, 'public consumer', 1, 0),
    (101, 'Corporate AI OMNI', 1, 0),
    (102, 'Corporate AI NYX', 1, 0),
    (104, 'Corporate AI QDY', 1, 0),
    (105, 'Corporate AI CYB', 1, 0),
    (106, 'Corporate AI SLD', 1, 0),
    (107, 'Corporate AI TTN', 1, 0),
    (108, 'Corporate AI NEB', 1, 0),
    (109, 'Corporate AI SOL', 1, 0),
    (110, 'Corporate AI ATM', 1, 0),
    (111, 'Corporate AI GLT', 1, 0),
    (112, 'Corporate AI SHD', 1, 0),
    (113, 'Corporate AI CHM', 1, 0),
    (114, 'Corporate AI VPH', 1, 0),
    (115, 'Corporate AI PAN', 1, 0),
    (116, 'Corporate AI SDL', 1, 0),
    (117, 'Corporate AI ROX', 1, 0),
    (118, 'Corporate AI GNN', 1, 0),
    (119, 'Corporate AI IFA', 1, 0),
    (120, 'Corporate AI AEG', 1, 0),
    (121, 'Corporate AI TRN', 1, 0),
    (122, 'Corporate AI VDC', 1, 0),
    (123, 'Corporate AI HRZ', 1, 0)
ON DUPLICATE KEY UPDATE
    username = VALUES(username),
    rank_id = VALUES(rank_id),
    created_at = VALUES(created_at);

INSERT INTO api_keys (api_key, user_id, role, created_at) VALUES
    (SHA1('role:market_maker'), 1000, 'market_maker', 0),
    (SHA1('role:liquidity_provider'), 1001, 'liquidity_provider', 0),
    (SHA1('role:whale_northern'), 1002, 'whale_northern', 0),
    (SHA1('role:whale_oceanic'), 1003, 'whale_oceanic', 0),
    (SHA1('role:whale_energy'), 1004, 'whale_energy', 0),
    (SHA1('role:national_ai_arcadia'), 1005, 'national_ai_arcadia', 0),
    (SHA1('role:national_ai_boros'), 1006, 'national_ai_boros', 0),
    (SHA1('role:national_ai_el_dorado'), 1007, 'national_ai_el_dorado', 0),
    (SHA1('role:national_ai_neo_venice'), 1008, 'national_ai_neo_venice', 0),
    (SHA1('role:national_ai_san_verde'), 1009, 'national_ai_san_verde', 0),
    (SHA1('role:national_ai_novaya_zemlya'), 1010, 'national_ai_novaya_zemlya', 0),
    (SHA1('role:national_ai_pearl_river'), 1011, 'national_ai_pearl_river', 0),
    (SHA1('role:momentum_chaser_a'), 1012, 'momentum_chaser_a', 0),
    (SHA1('role:momentum_chaser_b'), 1013, 'momentum_chaser_b', 0),
    (SHA1('role:momentum_chaser_c'), 1014, 'momentum_chaser_c', 0),
    (SHA1('role:dip_buyer_a'), 1015, 'dip_buyer_a', 0),
    (SHA1('role:dip_buyer_b'), 1016, 'dip_buyer_b', 0),
    (SHA1('role:dip_buyer_c'), 1017, 'dip_buyer_c', 0),
    (SHA1('role:reversal_sniper_a'), 1018, 'reversal_sniper_a', 0),
    (SHA1('role:reversal_sniper_b'), 1019, 'reversal_sniper_b', 0),
    (SHA1('role:reversal_sniper_c'), 1020, 'reversal_sniper_c', 0),
    (SHA1('role:grid_trader_a'), 1021, 'grid_trader_a', 0),
    (SHA1('role:grid_trader_b'), 1022, 'grid_trader_b', 0),
    (SHA1('role:grid_trader_c'), 1023, 'grid_trader_c', 0),
    (SHA1('role:news_reactor'), 1024, 'news_reactor', 0),
    (SHA1('role:arbitrageur'), 1025, 'arbitrageur', 0),
    (SHA1('role:yield_hunter'), 1026, 'yield_hunter', 0),
    (SHA1('role:public_consumer'), 1027, 'public_consumer', 0),
    (SHA1('role:corporate_ai_omni'), 101, 'corporate_ai_omni', 0),
    (SHA1('role:corporate_ai_nyx'), 102, 'corporate_ai_nyx', 0),
    (SHA1('role:corporate_ai_qdy'), 104, 'corporate_ai_qdy', 0),
    (SHA1('role:corporate_ai_cyb'), 105, 'corporate_ai_cyb', 0),
    (SHA1('role:corporate_ai_sld'), 106, 'corporate_ai_sld', 0),
    (SHA1('role:corporate_ai_ttn'), 107, 'corporate_ai_ttn', 0),
    (SHA1('role:corporate_ai_neb'), 108, 'corporate_ai_neb', 0),
    (SHA1('role:corporate_ai_sol'), 109, 'corporate_ai_sol', 0),
    (SHA1('role:corporate_ai_atm'), 110, 'corporate_ai_atm', 0),
    (SHA1('role:corporate_ai_glt'), 111, 'corporate_ai_glt', 0),
    (SHA1('role:corporate_ai_shd'), 112, 'corporate_ai_shd', 0),
    (SHA1('role:corporate_ai_chm'), 113, 'corporate_ai_chm', 0),
    (SHA1('role:corporate_ai_vph'), 114, 'corporate_ai_vph', 0),
    (SHA1('role:corporate_ai_pan'), 115, 'corporate_ai_pan', 0),
    (SHA1('role:corporate_ai_sdl'), 116, 'corporate_ai_sdl', 0),
    (SHA1('role:corporate_ai_rox'), 117, 'corporate_ai_rox', 0),
    (SHA1('role:corporate_ai_gnn'), 118, 'corporate_ai_gnn', 0),
    (SHA1('role:corporate_ai_ifa'), 119, 'corporate_ai_ifa', 0),
    (SHA1('role:corporate_ai_aeg'), 120, 'corporate_ai_aeg', 0),
    (SHA1('role:corporate_ai_trn'), 121, 'corporate_ai_trn', 0),
    (SHA1('role:corporate_ai_vdc'), 122, 'corporate_ai_vdc', 0),
    (SHA1('role:corporate_ai_hrz'), 123, 'corporate_ai_hrz', 0)
ON DUPLICATE KEY UPDATE
    api_key = VALUES(api_key),
    user_id = VALUES(user_id),
    created_at = VALUES(created_at);

SET FOREIGN_KEY_CHECKS = 1;
