# Theoretical FX Rate Calculation System

本ドキュメントでは、Paper Streetにおける為替レートの理論値を算出するためのシステム仕様について記述します。
この理論レートは、マクロ経済指標に基づいた「適正価格（Fair Value）」を示し、主にBot（Arbitrageur, National AI等）のトレード判断基準や、市場の過熱感を測る指標として利用されます。

## 1. 概要 (Overview)
為替レートは市場の需給によって決定されますが、その根底には各国の経済ファンダメンタルズが存在します。
本システムでは、Arcadia（基軸通貨 ARC）と対象国（Target Country）の経済指標の差分を用いて、理論的な為替レート（スコア）を算出します。

## 2. 計算式 (Formula)

理論レートは、以下の式によって算出されます。

$$Theoretical\ Rate (Local/ARC) = Base\_Score \times (1 + \alpha \cdot GDP\_Factor + \beta \cdot Rate\_Factor + \gamma \cdot CPI\_Factor)$$

### 2.1 パラメータ定義

| パラメータ | 定義 | 説明 |
| :--- | :--- | :--- |
| **Base_Score** | **1.0** (初期値) | 各通貨ペアごとの基準スコア。経済規模や流動性を考慮して調整可能ですが、基本はパリティ（1.0）から開始します。 |
| **GDP_Factor** | $\frac{Target\ GDP\ Growth}{Arcadia\ GDP\ Growth}$ | 経済成長率の比。成長率が高い国の通貨ほど買われやすくなります。 |
| **Rate_Factor** | $Target\ Interest\ Rate - Arcadia\ Interest\ Rate$ | 金利差。金利が高い通貨はスワップポイント狙いで買われやすくなります（キャリートレード）。 |
| **CPI_Factor** | $Arcadia\ Inflation\ Rate - Target\ Inflation\ Rate$ | インフレ率差。インフレ率が高い（＝通貨価値の減少が速い）通貨は売られやすくなります。Targetのインフレが高いとマイナスに寄与します。 |

### 2.2 係数 (Coefficients)
各ファクターがレートに与える影響度（感応度）を調整するための係数です。
初期バランスとして、以下の値を推奨します。

| 係数 | 値 (推奨) | 根拠 |
| :--- | :--- | :--- |
| **$\alpha$ (GDP)** | **0.2** | 成長率は長期的なトレンドを作りますが、短期的なインパクトは金利に劣るため、低めに設定します。GDP比が同等(1.0)の場合、+0.2のベースアップ効果を持ちます。 |
| **$\beta$ (Rate)** | **10.0** | 為替市場において金利差は最も強力なドライバであるため、高い感応度を設定します。1%の金利差でレートを10%動かします。 |
| **$\gamma$ (CPI)** | **5.0** | インフレ率は購買力平価説に基づき中程度の影響力を持ちます。1%のインフレ差でレートを5%動かします。 |

## 3. 計算フロー (Calculation Flow)

1.  **データ取得**:
    *   `docs/design/MACRO_ECONOMICS.md` で定義されたマクロ経済指標データベースから、最新の値を四半期（14日）ごと、または中央銀行の金利変更時に取得します。
2.  **ファクター計算**:
    *   Arcadiaと対象国の値を比較し、各ファクターを算出します。
3.  **理論レート更新**:
    *   計算式に基づいて `Theoretical Rate` を更新します。
4.  **Botへの通知**:
    *   更新された理論レートは、`Arbitrageur` や `National AI` などのBotにブロードキャストされ、彼らの指値注文や成行売買の基準価格（Anchor Price）として機能します。

## 4. 計算例 (Example)

**シナリオ**:
*   **Arcadia (ARC)**:
    *   GDP Growth: **2.0%**
    *   Interest Rate: **2.5%** (0.025)
    *   Inflation (CPI): **2.0%** (0.020)
*   **Target Country (Local)**:
    *   GDP Growth: **3.0%**
    *   Interest Rate: **4.0%** (0.040)
    *   Inflation (CPI): **3.0%** (0.030)

**各ファクターの計算**:
*   **GDP_Factor** = $3.0 / 2.0 = 1.5$
*   **Rate_Factor** = $0.040 - 0.025 = +0.015$ (1.5%)
*   **CPI_Factor** = $0.020 - 0.030 = -0.010$ (-1.0%)

**理論レートの算出**:
$$Theoretical\ Rate = 1.0 \times (1 + 0.2 \times 1.5 + 10.0 \times 0.015 + 5.0 \times (-0.010))$$
$$= 1.0 \times (1 + 0.30 + 0.15 - 0.05)$$
$$= 1.0 \times 1.40$$
$$= \mathbf{1.40}$$

**解釈**:
この場合、対象国（Local）の通貨は、経済成長と高い金利により、Arcadia（ARC）に対して **1.4倍** の理論的強さを持つと評価されます。
市場価格が `1.20` であれば「割安（Undervalued）」と判断され、Botによる買い注文が入る可能性が高まります。
