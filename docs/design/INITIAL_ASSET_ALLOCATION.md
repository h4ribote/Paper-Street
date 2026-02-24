# Initial Asset Allocation

このドキュメントでは、ゲーム開始時（Day 1）における各エンティティの初期資産（ARC: Arcadian Credit および 各国通貨）の配分メカニズムを定義します。


## 設計方針 (Design Philosophy)

1.  **ボトムアップ・アプローチ (Bottom-up Allocation)**:
    *   全体のパイ（総発行量）を固定せず、各プレイヤー、企業、国家に必要な「活動資金」を積み上げて総資産を決定します。
    *   5%程度のプレイヤー比率を目安とします。

2.  **現地通貨ベースの定義 (Local Currency First)**:
    *   企業や国家の資産は、ARC換算額ではなく、その本拠地で使用される**現地通貨（Local Currency）の絶対量**で定義します。

3.  **整数ベースの管理 (Integer-based Definition)**:
    *   割り算による端数を排除し、設定ファイル（Config）で管理しやすい整数値を採用します。


## エンティティ別配分 (Allocation by Entity)

### 1. Players (プレイヤー)
全プレイヤーに固定の初期資金を配分します。

*   **Initial Cash**: **10,000 ARC** (固定)
*   **Target Count**: 1,000 players (想定)
*   **Total Allocation**: 10,000,000 ARC

### 2. Whales & Institutions (機関投資家)
**通貨ブロック（地域）**ごとに資金を分割して保有します。
これにより、特定の経済圏内での大規模な実需取引や裁定取引を担当します。

| Entity Name | Focus Region | Currencies | Initial Assets (Each) |
| :--- | :--- | :--- | :--- | :--- |
| **Whale 1** | Northern / Eastern | **ARC, BRB, DRL** | **5,000,000** |
| **Whale 2** | Oceanic / Southern | **VND, VDP** | **7,500,000** |
| **Whale 3** | Energy / Industrial | **ZMR, RVD** | **7,500,000** |

### 3. National AIs (国家AI)
自国通貨の防衛資金（現地通貨）と、介入用の外貨準備（ARC）をそれぞれ物理的な枚数で保有します。
為替レートに関わらず、この数量が初期インベントリとして付与されます。

*   **Arcadia (基軸国)**:
    *   **ARC Reserve**: **30,000,000 ARC**
    *   *Note*: 基軸国のため外貨準備（他国通貨）は初期状態では持ちません。

*   **Other 6 Nations (他6カ国)**:
    *   **Local Currency Reserve**: **20,000,000 (Local)**
    *   **ARC Reserve (Foreign Reserve)**: **10,000,000 ARC**

| Nation | Currency | Local Reserve | ARC Reserve |
| :--- | :--- | :--- | :--- |
| **Boros Federation** | BRB | 20,000,000 BRB | 10,000,000 ARC |
| **El Dorado** | DRL | 20,000,000 DRL | 10,000,000 ARC |
| **Neo Venice** | VND | 20,000,000 VND | 10,000,000 ARC |
| **San Verde** | VDP | 20,000,000 VDP | 10,000,000 ARC |
| **Novaya Zemlya** | ZMR | 20,000,000 ZMR | 10,000,000 ARC |
| **Pearl River Zone** | RVD | 20,000,000 RVD | 10,000,000 ARC |

### 4. Corporations (企業)
企業の規模（Tier）ごとに**現地通貨ベースでの資金量**と**生産能力（Capacity）ベースの在庫量**を固定します。

| Tier | Scale | Cash (Local Currency) | Inventory (Product) | Count |
| :--- | :--- | :--- | :--- | :--- |
| **Tier 1** | Large Cap | **5,000,000** | Capacity × **2 Quarters** | 8 |
| **Tier 2** | Mid Cap | **2,000,000** | Capacity × **2 Quarters** | 8 |
| **Tier 3** | Small Cap | **1,000,000** | Capacity × **3 Quarters** | 5 |

*   **現金（Cash）**: 本拠地のある国の通貨で保有します。
*   **在庫（Inventory）**: 生産能力（Max Capacity）のNヶ月分として計算されます。

### 5. Market Makers (マーケットメイカー)
各通貨ペアおよび主要商品の流動性プールに、対等な価値ではなく**固定の数量**を供給します。
初期為替レートとの乖離は、市場開始直後の裁定取引（Arbitrage）によって調整されることを許容します。

*   **Base Allocation per Pair**:
    *   **ARC Side**: **1,000,000 ARC**
    *   **Local Side**: **2,000,000 (Local)**
    *   *Note*: これにより、初期プール比率は `1 ARC : 2 Local` (Rate = 0.50) となりますが、実際のTick設定時に調整可能です。

### 6. Algorithmic Traders (Other Bots)
特定の戦略に特化した小規模なボット群です。商品の取引に集中させるため、それぞれ3つの通貨グループに分かれて資金を保有します。

*   **対象ボット**: Momentum Chaser, Dip Buyer, Reversal Sniper, Grid Trader
*   **合計**: 12体 (4 types × 3 groups)

#### Group A: Northern / Eastern Mix
*   **保有通貨**: [**ARC**, **BRB**, **DRL**]
*   **初期資産**: 各通貨 **200,000** ずつ

#### Group B: Oceanic / Southern Agri
*   **保有通貨**: [**VND**, **VDP**]
*   **初期資産**: 各通貨 **300,000** ずつ

#### Group C: Energy / Industrial Zone
*   **保有通貨**: [**ZMR**, **RVD**]
*   **初期資産**: 各通貨 **300,000** ずつ
