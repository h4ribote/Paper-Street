# Equity Financing & Share Buybacks

本ドキュメントでは、企業による「自己株式の売却（資金調達）」および「自己株式の取得（自社株買い）」のメカニズムを記述します。
これらのアクションは、企業の財務状況や市場評価に基づいて自律的に実行され、株価や発行済株式数に直接的な影響を与えます。

## 1. 概要 (Overview)

Paper Streetの企業Botは、単に製品を売買するだけでなく、自社の資本政策（Capital Policy）をコントロールします。
特に、初期状態において各企業は**発行済株式の50%を自己株式（Treasury Stock）として保有**しています。資金調達時には、まずこの保有株を売却し、それが尽きた場合にのみ新株発行を行います。

*   **Equity Financing (資金調達)**:
    1.  **Treasury Stock Sale**: 保有する自己株式を市場で売却。
    2.  **New Share Issuance**: 新株を発行して市場で売却。
*   **Share Buyback (自社株買い)**: 余剰資金を使って市場から自社株を買い戻します。

## 2. 資金調達メカニズム (Equity Financing)

企業が現金を必要とする場合、以下のプロセスで資金調達を行います。

### 2.1 トリガー (Triggers)
以下のいずれかの条件を満たした場合、資金調達モードに入ります。

1.  **Safety Margin Breach (資金不足)**:
    *   `Current Cash < Weekly Operating Cost * 2.0`
    *   運転資金が枯渇しそうな場合、生存のために緊急増資を行います。
2.  **Aggressive Expansion (積極投資)**:
    *   大規模なCapEx（設備投資）が計画されており、手元資金では不足する場合。
3.  **Overvaluation Opportunity (割高是正)**:
    *   株価が適正価格（理論株価や移動平均）を大幅に上回っており（例: `Price > 200-day MA * 1.5`）、かつP/Eレシオが異常値（例: > 50倍）である場合。
    *   「高い株価で現金を調達するのは既存株主にとっても有利」と判断します。

### 2.2 調達フェーズ (Phases)
資金調達は、保有在庫（Treasury Stock）の有無によって2つのフェーズに分かれます。

#### Phase 1: Secondary Offering (自己株式の売却)
*   **条件**: `Treasury Stock Balance > 0`
*   **アクション**: 企業が保有している「自己株式（Treasury Stock）」を市場で売却します。
*   **影響**:
    *   `Total Shares Outstanding`（市場流通株式数）が増加します。
    *   `Total Shares Issued`（発行済総数）は変化しません。
    *   EPSの分母（Outstanding）が増えるため、**希薄化 (Dilution)** は発生します。

#### Phase 2: Primary Offering (新株発行)
*   **条件**: `Treasury Stock Balance == 0`
*   **アクション**: 新たに株式を発行（増資）し、市場で売却します。
*   **影響**:
    *   `Total Shares Outstanding` が増加します。
    *   `Total Shares Issued` も増加します。
    *   Phase 1と同様に **希薄化 (Dilution)** が発生します。

### 2.3 実行プロセス (Execution Process)

1.  **アナウンス (Announcement)**:
    *   ニュース配信: `[FINANCE] OmniCorp announces plan to raise 10M ARC via [Secondary Offering / Public Offering] to fund expansion.`
    *   市場への影響: ニュースが出た瞬間、希薄化懸念から株価は下落する傾向があります。

2.  **発行価格決定 (Pricing)**:
    *   市場価格から一定の**ディスカウント (Discount)** を適用して発行価格を決めます。
    *   `Offering Price = Current Market Price * (1 - Discount Rate)`
    *   `Discount Rate`: 通常 **3% 〜 5%**。緊急時の場合は **10% 〜 20%** となり、既存株主へのダメージが大きくなります。

3.  **売却プロセス (Selling)**:
    *   Botは「発行価格」を最低ライン（指値）として、市場の買い板（Bids）に売り注文をぶつけます（Iceberg Orderなどを利用し、時間をかけて売却）。

### 2.4 計算式 (Formulas)

*   **調達目標額 (Target Amount)**:
    *   `Target = Required Cash - Current Cash + Buffer`
*   **売却必要株式数 (Required Shares)**:
    *   `Required Shares = Target Amount / Offering Price`
*   **希薄化率 (Dilution Ratio)**:
    *   希薄化率は、Outstanding（流通株式数）の増加分で計算されます。
    *   `Dilution % = Required Shares / (Current Outstanding + Required Shares)`

## 3. 自社株買いメカニズム (Share Buyback)

企業が使い道のない過剰な現金を保有している場合、株主還元として自社株買いを行います。

### 3.1 トリガー (Triggers)
以下の条件を**すべて**満たした場合に検討されます。

1.  **Excess Cash (余剰資金)**:
    *   `Current Cash > Weekly Operating Cost * 5.0`
    *   かつ、直近で大規模なCapExの予定がない。
2.  **Undervaluation (割安放置)**:
    *   P/Eレシオがセクター平均を下回っている、または `Price < 200-day MA * 0.8` など、株価が低迷している場合。

### 3.2 実行プロセス (Execution)

1.  **アナウンス (Announcement)**:
    *   ニュース配信: `[BUYBACK] Titan Energy authorizes 50M BRB share repurchase program.`
    *   市場への影響: ポジティブサプライズとして株価上昇要因となります。

2.  **買付プロセス (Buying)**:
    *   Botは指定された予算（Buyback Budget）を使い切り、かつ株価を過度に釣り上げないように、VWAP（出来高加重平均価格）連動アルゴリズムで買い注文を出します。
    *   **価格制限**: `Limit Price = Current Market Price * 1.05` （現在値の+5%までしか追いかけない）。

3.  **保有・消却 (Holding vs Retirement)**:
    *   市場から買い戻した株式は、即座には消却されず、**Treasury Stock（金庫株）** として保有されます。
    *   これにより、将来の資金調達（Phase 1）の原資として再利用が可能になります。
    *   ただし、`Treasury Stock > Issued Shares * 60%` など過剰に積み上がった場合は、一部を消却（Retirement）してバランスを調整します。

### 3.3 計算式 (Impact Calculation)

*   **EPS上昇効果**:
    *   `New EPS = Net Income / (Old Outstanding - Repurchased Shares)`
    *   Treasury Stockとして保有された株式はOutstanding（市場流通）から除外されるため、EPS計算の分母が減り、EPSは上昇します。

## 4. 財務への影響 (Financial Impact)

これらのアクションは、B/S（貸借対照表）とP/L（損益計算書）の主要指標に即座に反映されます。

| 項目 | Phase 1 (Treasury Sale) | Phase 2 (New Issuance) | Share Buyback |
| :--- | :--- | :--- | :--- |
| **Cash (現金)** | **増加 (+)** | **増加 (+)** | **減少 (-)** |
| **Shares Outstanding (流通株式数)** | **増加 (+)** | **増加 (+)** | **減少 (-)** |
| **Shares Issued (発行済総数)** | 変化なし | **増加 (+)** | 変化なし |
| **Treasury Stock (自己株式)** | **減少 (-)** | 変化なし | **増加 (+)** |
| **EPS (1株益)** | **希薄化 (Dilution)** | **希薄化 (Dilution)** | **上昇 (Accretion)** |

## 5. 規制・制限 (Regulations)

市場の流動性を枯渇させないよう、1日の自社株買い数量は**過去5日間の平均出来高の25%**までに制限されます。
