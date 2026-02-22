# Initial Asset Allocation

このドキュメントでは、ゲーム開始時（Day 1）における各エンティティの初期資産（ARC: Arcadian Credit）の配分を定義します。

## 概要 (Overview)
*   **総資産供給量 (Total Supply)**: **200,000,000 ARC**
*   **プレイヤー資産比率**: 5.0% (10,000,000 ARC)
    *   プレイヤーの資産が全体市場に与える影響を限定的（5%以内）にするため、総資産を2億ARCに設定しました。

## 配分詳細 (Allocation Breakdown)

| エンティティ種別 (Entity Type) | 数 (Count) | 合計資産 (Total ARC) | 1体あたり (Per Entity) | 比率 (%) | 詳細 (Details) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Players** | 1,000 | 10,000,000 | 10,000 | 5.00% | 初期資金: 10,000 ARC/人 |
| **National AIs** | 7 | 100,000,000 | (varies) | 50.00% | 為替介入や国債売買を行うための巨額資金。 |
| **Whales (Institutional)** | 3 | 40,000,000 | ~13,333,333 | 20.00% | 相場を動かす機関投資家。 |
| **Corporations** | 21 | 30,000,000 | (varies) | 15.00% | 事業活動（仕入れ・投資）用資金。 |
| **Market Makers** | 5 | 15,000,000 | 3,000,000 | 7.50% | 現金50% : 商品50% で初期化。 |
| **Insurance Fund** | 1 | 3,000,000 | 3,000,000 | 1.50% | ロスカット補償用のシード資金。 |
| **Arbitrageurs** | 2 | 1,000,000 | 500,000 | 0.50% | インデックス裁定取引ボット。 |
| **News Reactors** | 2 | 400,000 | 200,000 | 0.20% | ニュース反応ボット。 |
| **Other Bots** | 12 | 600,000 | 50,000 | 0.30% | Momentum, Dip, Reversal, Grid (各3体)。 |
| **TOTAL** | **1,053** | **200,000,000** | - | **100.00%** | |

## 詳細設定 (Detailed Settings)

### 1. National AIs (国家AI)
国家AIは、自国通貨の防衛や経済政策の実行に必要な最も潤沢な資金を持ちます。

*   **Arcadia (Tech Utopia)**: **22,000,000 ARC**
    *   基軸通貨国であり、世界経済の中心であるため、最大の資金を保有します。
*   **Other 6 Nations**: **13,000,000 ARC each**
    *   Boros Federation, El Dorado, Neo Venice, San Verde, Novaya Zemlya, Pearl River Zone
    *   合計: 13,000,000 * 6 = 78,000,000 ARC

### 2. Whales (機関投資家)
大口注文でトレンドを形成したり、逆張りを行ったりする強力なボットです。

*   **Whale 1**: 13,300,000 ARC
*   **Whale 2**: 13,300,000 ARC
*   **Whale 3**: 13,400,000 ARC

### 3. Corporations (企業)
企業の規模（時価総額ベースのTier）に応じて初期資金が異なります。

*   **Tier 1 (Large Cap)**: **2,000,000 ARC each** (8社)
    *   OmniCorp, Goliath Bank, Titan Energy, Trans-Oceanic, Red Ox Food, Iron Fist Armaments, Silicon Dragon, Global News Network
    *   合計: 16,000,000 ARC
*   **Tier 2 (Mid Cap)**: **1,000,000 ARC each** (8社)
    *   Quantum Dynamics, CyberLife, Aegis Systems, Helios Solar, Atomos Energy, Chimera Genetics, Stardust Luxury, Horizon Logistics
    *   合計: 8,000,000 ARC
*   **Tier 3 (Small Cap / Venture)**: **1,200,000 ARC each** (5社)
    *   Nebula Mining, Shadow Fund, Verde Pharma, Panacea Corp, Void Cargo
    *   合計: 6,000,000 ARC
    *   *Note*: ベンチャー企業は運転資金として現金比率が高めに設定されています。

### 4. Market Makers (マーケットメイカー)
流動性提供のため、現金と在庫（Asset Inventory）をバランスよく保有します。

*   **初期配分**:
    *   **Cash**: 1,500,000 ARC
    *   **Inventory**: 1,500,000 ARC相当の株式・コモディティ

### 5. Other Bots (その他ボット)
特定の戦略に特化した小規模なボット群です。

*   **Momentum Chasers**: 3体 (50,000 ARC each)
*   **Dip Buyers**: 3体 (50,000 ARC each)
*   **Reversal Snipers**: 3体 (50,000 ARC each)
*   **Grid Traders**: 3体 (50,000 ARC each)


## 初期市場レート (Initial Market Rates)

### 為替レート (Exchange Rates)
全ての通貨ペアはARCを基軸（Quote Currency）として取引されます。

| Currency Name | Code | Initial Rate (ARC) | Origin | Notes |
| :--- | :--- | :--- | :--- | :--- |
| **Boros Ruble** | **BRB** | **0.50** | Boros Federation | 工業輸出に適した安値圏。 |
| **Dorado Real** | **DRL** | **2.00** | El Dorado | 豊富な資源を背景とした強気相場。 |
| **Venice Dollar** | **VND** | **1.00** | Neo Venice | 金融ハブとしてARCと等価(Parity)で開始。 |
| **Verde Peso** | **VDP** | **0.20** | San Verde | 農業国であり、最も安価な通貨。 |
| **Zemlya Ruble** | **ZMR** | **0.60** | Novaya Zemlya | エネルギー資源国。 |
| **River Dollar** | **RVD** | **0.40** | Pearl River Zone | 製造業特区。BRBと同様に安値を維持。 |

### 商品価格 (Commodity Prices)
商品は産出国または主要企業の拠点通貨で価格が設定されます。

| Commodity | Origin Company | Origin Currency | Initial Price (Local) | Initial Price (ARC Est.) | Unit |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Crude Oil** | Titan Energy | **BRB** | **100.00** | 50.00 | Barrel |
| **Rare Earth** | El Dorado (State) | **DRL** | **50.00** | 100.00 | kg |
| **Grain** | San Verde (State) | **VDP** | **20.00** | 4.00 | Bushel |
| **Semiconductors** | Silicon Dragon | **RVD** | **250.00** | 100.00 | Chip |
| **Hydrogen** | Helios Solar | **DRL** | **10.00** | 20.00 | Unit |
| **Uranium** | Atomos Energy | **ZMR** | **500.00** | 300.00 | kg |


## 詳細資産内訳 (Detailed Asset Breakdown)

### 1. National AIs (Foreign Reserves)
国家AIは、自国通貨の安定化のために「自国通貨」と「外貨準備（ARC）」を保有します。
各国の資産総額は13,000,000 ARC相当です。内訳は自国通貨50%（6.5M ARC相当）、ARC準備50%（6.5M ARC）となります。

| Nation | Currency | Exchange Rate | **Local Currency Holding** | **ARC Reserve** |
| :--- | :--- | :--- | :--- | :--- |
| **Arcadia** | ARC | 1.00 | 22,000,000 ARC | - |
| **Boros Federation** | BRB | 0.50 | **13,000,000 BRB** | 6,500,000 ARC |
| **El Dorado** | DRL | 2.00 | **3,250,000 DRL** | 6,500,000 ARC |
| **Neo Venice** | VND | 1.00 | **6,500,000 VND** | 6,500,000 ARC |
| **San Verde** | VDP | 0.20 | **32,500,000 VDP** | 6,500,000 ARC |
| **Novaya Zemlya** | ZMR | 0.60 | **10,833,333 ZMR** | 6,500,000 ARC |
| **Pearl River Zone** | RVD | 0.40 | **16,250,000 RVD** | 6,500,000 ARC |

### 2. Corporations (Treasury & Inventory)
企業は運転資金としての現金（70%）と在庫（30%）を保有します。
現金は原則として**本拠地のある国の通貨 (Local Currency)** で保有します。

#### Tier 1 (Total Value: 2,000,000 ARC / Cash: 1,400,000 ARC Value)
| Company | HQ | Currency | **Initial Cash Holding** |
| :--- | :--- | :--- | :--- |
| **OmniCorp** | Arcadia | ARC | **1,400,000 ARC** |
| **Goliath Bank** | Arcadia | ARC | **1,400,000 ARC** |
| **Titan Energy** | Boros | BRB | **2,800,000 BRB** |
| **Trans-Oceanic** | Neo Venice | VND | **1,400,000 VND** |
| **Red Ox Food** | San Verde | VDP | **7,000,000 VDP** |
| **Iron Fist Armaments** | Boros | BRB | **2,800,000 BRB** |
| **Silicon Dragon** | Pearl River | RVD | **3,500,000 RVD** |
| **Global News Network** | Neo Venice | VND | **1,400,000 VND** |

#### Tier 2 (Total Value: 1,000,000 ARC / Cash: 700,000 ARC Value)
| Company | HQ | Currency | **Initial Cash Holding** |
| :--- | :--- | :--- | :--- |
| **Quantum Dynamics** | Arcadia | ARC | **700,000 ARC** |
| **CyberLife** | Neo Venice | VND | **700,000 VND** |
| **Aegis Systems** | Arcadia | ARC | **700,000 ARC** |
| **Helios Solar** | El Dorado | DRL | **350,000 DRL** |
| **Atomos Energy** | Novaya Zemlya | ZMR | **1,166,666 ZMR** |
| **Chimera Genetics** | Arcadia | ARC | **700,000 ARC** |
| **Stardust Luxury** | Neo Venice | VND | **700,000 VND** |
| **Horizon Logistics** | Arcadia | ARC | **700,000 ARC** |

#### Tier 3 (Total Value: 1,200,000 ARC / Cash: 840,000 ARC Value)
*Tier 3はベンチャー企業のため、現金比率が同じ70%でも評価額ベースで少し多めに設定されています。*

| Company | HQ | Currency | **Initial Cash Holding** |
| :--- | :--- | :--- | :--- |
| **Nebula Mining** | Novaya Zemlya | ZMR | **1,400,000 ZMR** |
| **Shadow Fund** | Neo Venice | VND | **840,000 VND** |
| **Verde Pharma** | San Verde | VDP | **4,200,000 VDP** |
| **Panacea Corp** | El Dorado | DRL | **420,000 DRL** |
| **Void Cargo** | Arcadia | ARC | **840,000 ARC** |

### 3. Market Makers (Liquidity Inventory)
マーケットメイカーは、全通貨ペアおよび主要商品の流動性を提供するため、バスケットを保有します。

#### Currency Specialists (3 Bots)
*   **Total Assets per Bot**: 3,000,000 ARC
*   **ARC Allocation (40%)**: 1,200,000 ARC
*   **Foreign Currency Allocation (60%)**: 1,800,000 ARC Value (300,000 ARC Value per currency)

| Currency | Rate | ARC Value | **Holding Amount** |
| :--- | :--- | :--- | :--- |
| **BRB** | 0.50 | 300,000 | **600,000 BRB** |
| **DRL** | 2.00 | 300,000 | **150,000 DRL** |
| **VND** | 1.00 | 300,000 | **300,000 VND** |
| **VDP** | 0.20 | 300,000 | **1,500,000 VDP** |
| **ZMR** | 0.60 | 300,000 | **500,000 ZMR** |
| **RVD** | 0.40 | 300,000 | **750,000 RVD** |

#### Commodity Specialists (2 Bots)
*   **Total Assets per Bot**: 3,000,000 ARC
*   **ARC Allocation (30%)**: 900,000 ARC
*   **Commodity Allocation (40%)**: 1,200,000 ARC Value (Basket of commodities)
*   **Local Currency Allocation (30%)**: 900,000 ARC Value (Equally split: 150,000 ARC Value per currency)

| Currency | Rate | ARC Value | **Holding Amount** |
| :--- | :--- | :--- | :--- |
| **BRB** | 0.50 | 150,000 | **300,000 BRB** |
| **DRL** | 2.00 | 150,000 | **75,000 DRL** |
| **VND** | 1.00 | 150,000 | **150,000 VND** |
| **VDP** | 0.20 | 150,000 | **750,000 VDP** |
| **ZMR** | 0.60 | 150,000 | **250,000 ZMR** |
| **RVD** | 0.40 | 150,000 | **375,000 RVD** |
