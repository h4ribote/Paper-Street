-- ==========================================
-- Paper Street: Seed Data SQL
-- ==========================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- --------------------------------------------------------
-- 1. Countries / Regions
-- --------------------------------------------------------

INSERT INTO regions (region_id, name, description)
SELECT 1, 'Northern Alliance', 'Advanced markets with high-tech leadership and aging demographics.'
WHERE NOT EXISTS (SELECT 1 FROM regions WHERE region_id = 1 OR name = 'Northern Alliance');

INSERT INTO regions (region_id, name, description)
SELECT 2, 'Eastern Coalition', 'Industrial powerhouse with state-led growth and export-driven policy.'
WHERE NOT EXISTS (SELECT 1 FROM regions WHERE region_id = 2 OR name = 'Eastern Coalition');

INSERT INTO regions (region_id, name, description)
SELECT 3, 'Southern Resource Pact', 'Resource-rich bloc with high commodity exposure and political risk.'
WHERE NOT EXISTS (SELECT 1 FROM regions WHERE region_id = 3 OR name = 'Southern Resource Pact');

INSERT INTO regions (region_id, name, description)
SELECT 4, 'Oceanic Tech Arch', 'Financial hubs and tax havens fueling volatile innovation.'
WHERE NOT EXISTS (SELECT 1 FROM regions WHERE region_id = 4 OR name = 'Oceanic Tech Arch');

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
SELECT 5, 4, 'San Verde'
WHERE NOT EXISTS (SELECT 1 FROM countries WHERE country_id = 5 OR name = 'San Verde');

-- --------------------------------------------------------
-- 2. Sectors
-- --------------------------------------------------------

INSERT INTO sectors (sector_id, code, name) VALUES
    (1, 'TECH', 'TECH'),
    (2, 'ENERGY', 'ENERGY'),
    (3, 'METAL', 'METAL')
ON DUPLICATE KEY UPDATE
    code = VALUES(code),
    name = VALUES(name);

-- --------------------------------------------------------
-- 3. Initial Symbols (Companies / Assets)
-- --------------------------------------------------------

INSERT INTO companies (
    company_id,
    country_id,
    sector_id,
    name,
    ticker_symbol,
    description,
    user_id,
    max_production_capacity,
    current_inventory,
    last_capex_at,
    shares_issued,
    shares_outstanding,
    treasury_stock
) VALUES
    (
        101,
        (SELECT country_id FROM countries WHERE name = 'Arcadia' LIMIT 1),
        (SELECT sector_id FROM sectors WHERE code = 'TECH' LIMIT 1),
        'Omni Dynamics',
        'OMNI',
        'Core technology issuer in Arcadia.',
        NULL,
        10000,
        0,
        0,
        1000000,
        500000,
        500000
    ),
    (
        102,
        (SELECT country_id FROM countries WHERE name = 'Boros Federation' LIMIT 1),
        (SELECT sector_id FROM sectors WHERE code = 'ENERGY' LIMIT 1),
        'Nyx Energy',
        'NYX',
        'Energy major with global fuel infrastructure.',
        NULL,
        10000,
        0,
        0,
        1000000,
        500000,
        500000
    )
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

INSERT INTO assets (
    asset_id,
    ticker,
    company_id,
    resource_id,
    type,
    base_price,
    lot_size,
    is_tradable,
    created_at
) VALUES
    (101, 'OMNI', 101, NULL, 'STOCK', 15250, 1, TRUE, 0),
    (102, 'NYX', 102, NULL, 'STOCK', 9825, 1, TRUE, 0),
    (103, 'AUR', NULL, NULL, 'COMMODITY', 18750, 1, TRUE, 0)
ON DUPLICATE KEY UPDATE
    ticker = VALUES(ticker),
    company_id = VALUES(company_id),
    resource_id = VALUES(resource_id),
    type = VALUES(type),
    base_price = VALUES(base_price),
    lot_size = VALUES(lot_size),
    is_tradable = VALUES(is_tradable);

SET FOREIGN_KEY_CHECKS = 1;
