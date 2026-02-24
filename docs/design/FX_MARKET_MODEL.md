# FX Market Model (AMM) Specification

## 1. 概要 (Overview)

Paper Street の FX市場は、**Uniswap V3** をベースとした **集中流動性 (Concentrated Liquidity)** モデルを採用した Automated Market Maker (AMM) です。
従来の $x \times y = k$ モデル（Constant Product AMM）とは異なり、流動性提供者（LP）は特定の価格帯（Tick Range）に資本を集中させることができ、資本効率を劇的に向上させています。

すべての通貨ペアは、基軸通貨である **Arcadian Credit (ARC)** とのペア（例: `VDP/ARC`, `BRB/ARC`）として存在します。直接のペアが存在しない通貨間の交換（例: `VDP` → `BRB`）は、ARCを経由するマルチホップ取引（Multi-hop Swap）として実行されます。


## 2. 数学的モデル (Mathematical Model)

### 2.1. 仮想流動性と不変量 (Virtual Reserves & Invariant)
集中流動性モデルでは、特定の価格範囲 $[P_a, P_b]$ において、以下の不変量が成立するように動作します。

$$ (x + \frac{L}{\sqrt{P_b}})(y + L\sqrt{P_a}) = L^2 $$

ここで:
*   $x$: 通貨0（Base Currency, e.g., ARC）の実残高
*   $y$: 通貨1（Quote Currency, e.g., VDP）の実残高
*   $L$: 流動性 (Liquidity)
*   $P$: 価格 ($\sqrt{P} = \sqrt{y/x}$)

このモデルは、範囲外では $x$ または $y$ のどちらか一方のみの資産を保有し、範囲内では $x \times y = k$ の曲線に従って資産が交換されることを意味します。

### 2.2. Tick System
価格空間は離散的な **Tick** ($i$) によって分割されます。各Tickにおける価格 $P(i)$ は以下のように定義されます。

$$ P(i) = 1.0001^i $$

LPは、整数インデックス $i_{lower}$ と $i_{upper}$ を指定して流動性を提供します。
現在の価格（Tick）がこの範囲内にある場合のみ、LPは手数料を獲得し、資産の交換に応じます。

### 2.3. Swap Math (Single Tick)
あるTick内で価格が $P_{current}$ から $P_{target}$ まで移動する際の、必要なトークン量（$\Delta x, \Delta y$）は以下の通りです。

**$x$ (ARC) を $y$ (Quote) に交換する場合 (Price increases):**
$$ \Delta x = \Delta \frac{1}{\sqrt{P}} \cdot L = L \cdot (\frac{1}{\sqrt{P_{current}}} - \frac{1}{\sqrt{P_{target}}}) $$
$$ \Delta y = \Delta \sqrt{P} \cdot L = L \cdot (\sqrt{P_{target}} - \sqrt{P_{current}}) $$

価格がTick境界（$P_{next}$）に到達すると、次のTickの流動性 $L_{next}$ をロードし、計算を継続します。


## 3. 手数料体系 (Fee Tiers)

Paper Street では、各通貨ペアに対して異なるリスクプロファイルとボラティリティに対応するため、2つの手数料層（Fee Tier）を用意しています。
LPは、自分のリスク許容度に応じて、どちらのプール（あるいは両方）に流動性を提供するかを選択できます。

### Tier 1: 0.04% (Low Fee)
*   **手数料率**: **0.04%** (4 bps)
*   **Tick Spacing**: 10 ticks (約 0.1%)
*   **推奨用途**:
    *   ボラティリティの低い通貨ペア（例: 安定したTech国家同士の通貨ペア）。
    *   取引頻度が高く、スプレッドを極限まで縮めたい場合。
    *   **Leviathan Rank**: 適用手数料 **0.02%**

### Tier 2: 0.20% (Standard Fee)
*   **手数料率**: **0.20%** (20 bps)
*   **Tick Spacing**: 50 ticks (約 0.5%)
*   **推奨用途**:
    *   ボラティリティの高い通貨ペア（例: Resource Kingdomの通貨や、新興国通貨）。
    *   価格変動が激しく、LPがインパーマネントロス（IL）のリスクヘッジとして高い手数料を要求する場合。
    *   **Leviathan Rank**: 適用手数料 **0.10%**


## 4. 取引ルート最適化 (Router Optimization)

トレーダーが交換（Swap）を行う際、システム（Router）は「最も多くのOutputトークンを獲得できる（＝実質コストが最も低い）」ルートを自動的に計算して提示します。

### 4.1. 最適化アルゴリズム
Routerは以下の要素を考慮してパス探索を行います。

1.  **Direct Swap (直接ペア)**:
    *   `Token A` → `Token B` の直接プールが存在する場合。
    *   複数のFee Tier（0.04%プールと0.20%プール）が存在する場合、取引額を**分割（Split）**して両方のプールで約定させることがあります。
    *   *例*: 1,000,000 ARCの売り注文を、流動性の厚い0.04%プールで70%、0.20%プールで30%約定させることで、Price Impactと手数料の合計を最小化します。

2.  **Multi-hop Swap (マルチホップ)**:
    *   直接ペアがない場合、基軸通貨（ARC）を経由します。
    *   `Token A` → `ARC` → `Token B`
    *   各ホップで手数料が発生するため、トータルコストは高くなりますが、流動性が存在しない場合は唯一のルートとなります。

3.  **Graph Pathfinding**:
    *   全プールをグラフのノード・エッジと見なし、Dijkstraアルゴリズムの変種を用いて「Output量」を最大化するパスを探索します。
    *   エッジの重みは `Log(1 - Fee_Rate) - Log(Slippage)` のようにモデル化されます。

### 4.2. Leviathanランクの影響
Routerは、プレイヤーのランクが **Leviathan** である場合、**割引後の手数料率（0.02% / 0.10%）** を用いて最適ルートを再計算します。
手数料が安くなるため、通常ランクのプレイヤーよりも「手数料は高いが流動性が厚いプール（Tier 2）」や「マルチホップルート」が選択肢に入りやすくなり、結果としてより有利なレート（低いPrice Impact）で約定できる可能性が高まります。


## 5. プレイヤーランクによる手数料優遇 (Leviathan Privilege)

最高ランクである **Leviathan** に到達したプレイヤーには、以下の手数料優遇が適用されます。

| Fee Tier | Standard Fee | **Leviathan Fee** |
| :--- | :--- | :--- |
| **Low** | 0.04% (4 bps) | **0.02%** (2 bps) |
| **Standard** | 0.20% (20 bps) | **0.10%** (10 bps) |

### 5.1. 流動性提供者（LP）への影響
**重要**: Leviathanに対する手数料割引分は、システムや運営によって補填されることはありません。

*   **LPの受取額**: Leviathanがスワップを行った場合、そのプールに蓄積される手数料収入は、割引後のレート（0.02% または 0.10%）に基づきます。
*   **LPのリスクとリターン**:
    *   Leviathanは大口トレーダーであることが多いため、取引ボリューム（出来高）自体はLPにとって魅力的です。
    *   しかし、単位あたりの収益性は半減するため、LP戦略においては「Leviathanのフローが多いペア」と「一般プレイヤー（Shrimp-Whale）のフローが多いペア」を見極めることが新たなゲームプレイ要素となります。
