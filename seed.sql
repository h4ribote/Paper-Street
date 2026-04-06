-- ==========================================
-- Paper Street: Initial Seed Data
-- ==========================================

SET NAMES utf8mb4;

-- --------------------------------------------------------
-- 1. Regions / Countries / Currencies
-- --------------------------------------------------------

INSERT INTO regions (region_id, name, description) VALUES
    (1, 'Northern Alliance', 'Advanced markets with high-tech leadership and aging demographics.'),
    (2, 'Eastern Coalition', 'Industrial powerhouse with state-led growth and export-driven policy.'),
    (3, 'Southern Resource Pact', 'Resource-rich bloc with high commodity exposure and political risk.'),
    (4, 'Oceanic Tech Arch', 'Financial hubs and tax havens fueling volatile innovation.')
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    description = VALUES(description);

INSERT INTO countries (country_id, region_id, name) VALUES
    (1, 1, 'Arcadia'),
    (2, 2, 'Boros Federation'),
    (3, 3, 'El Dorado'),
    (4, 4, 'Neo Venice'),
    (5, 3, 'San Verde'),
    (6, 1, 'Novaya Zemlya'),
    (7, 2, 'Pearl River Zone')
ON DUPLICATE KEY UPDATE
    region_id = VALUES(region_id),
    name = VALUES(name);

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
-- 2. Sectors
-- --------------------------------------------------------

INSERT INTO sectors (sector_id, code, name) VALUES
    (1, 'TECH', 'TECH'),
    (2, 'ENERGY', 'ENERGY'),
    (3, 'FIN', 'FIN'),
    (4, 'BIO', 'BIO'),
    (5, 'CONS', 'CONS'),
    (6, 'DEF', 'DEF'),
    (7, 'LOG', 'LOG')
ON DUPLICATE KEY UPDATE
    code = VALUES(code),
    name = VALUES(name);

-- --------------------------------------------------------
-- 3. Companies (Stocks)
-- --------------------------------------------------------

INSERT INTO companies (
    company_id, country_id, sector_id, name, ticker_symbol, description, user_id,
    max_production_capacity, current_inventory, last_capex_at,
    shares_issued, shares_outstanding, treasury_stock
) VALUES
    (101, 1, 1, 'OmniCorp', 'OMNI', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (102, 2, 2, 'Titan Energy', 'TTN', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (104, 1, 1, 'Quantum Dynamics', 'QDY', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (105, 4, 1, 'CyberLife', 'CYB', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (106, 7, 1, 'Silicon Dragon', 'SLD', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (107, 6, 2, 'Nebula Mining', 'NEB', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (108, 3, 2, 'Helios Solar', 'SOL', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (109, 6, 2, 'Atomos Energy', 'ATM', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (110, 1, 3, 'Goliath Bank', 'GLT', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (111, 4, 3, 'Shadow Fund', 'SHD', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (112, 1, 4, 'Chimera Genetics', 'CHM', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (113, 5, 4, 'Verde Pharma', 'VPH', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (114, 3, 4, 'Panacea Corp', 'PAN', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (115, 4, 5, 'Stardust Luxury', 'SDL', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (116, 5, 5, 'Red Ox Food', 'ROX', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (117, 4, 5, 'Global News Network', 'GNN', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (118, 2, 6, 'Iron Fist Armaments', 'IFA', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (119, 1, 6, 'Aegis Systems', 'AEG', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (120, 4, 7, 'Trans-Oceanic', 'TRN', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (121, 1, 7, 'Void Cargo', 'VDC', '', NULL, 10000, 0, 0, 1000000, 500000, 500000),
    (122, 1, 7, 'Horizon Logistics', 'HRZ', '', NULL, 10000, 0, 0, 1000000, 500000, 500000)
ON DUPLICATE KEY UPDATE
    country_id = VALUES(country_id),
    sector_id = VALUES(sector_id),
    name = VALUES(name),
    ticker_symbol = VALUES(ticker_symbol),
    max_production_capacity = VALUES(max_production_capacity),
    current_inventory = VALUES(current_inventory),
    last_capex_at = VALUES(last_capex_at),
    shares_issued = VALUES(shares_issued),
    shares_outstanding = VALUES(shares_outstanding),
    treasury_stock = VALUES(treasury_stock);

-- --------------------------------------------------------
-- 4. Assets (Stocks + contract compatibility commodity)
-- --------------------------------------------------------

INSERT INTO assets (asset_id, ticker, company_id, resource_id, type, base_price, lot_size, is_tradable, created_at) VALUES
    (101, 'OMNI', 101, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (102, 'TTN', 102, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (103, 'REM', NULL, NULL, 'COMMODITY', 10000, 1, TRUE, 0),
    (104, 'QDY', 104, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (105, 'CYB', 105, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (106, 'SLD', 106, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (107, 'NEB', 107, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (108, 'SOL', 108, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (109, 'ATM', 109, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (110, 'GLT', 110, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (111, 'SHD', 111, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (112, 'CHM', 112, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (113, 'VPH', 113, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (114, 'PAN', 114, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (115, 'SDL', 115, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (116, 'ROX', 116, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (117, 'GNN', 117, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (118, 'IFA', 118, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (119, 'AEG', 119, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (120, 'TRN', 120, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (121, 'VDC', 121, NULL, 'STOCK', 10000, 1, TRUE, 0),
    (122, 'HRZ', 122, NULL, 'STOCK', 10000, 1, TRUE, 0)
ON DUPLICATE KEY UPDATE
    ticker = VALUES(ticker),
    company_id = VALUES(company_id),
    resource_id = VALUES(resource_id),
    type = VALUES(type),
    base_price = VALUES(base_price),
    lot_size = VALUES(lot_size),
    is_tradable = VALUES(is_tradable);
