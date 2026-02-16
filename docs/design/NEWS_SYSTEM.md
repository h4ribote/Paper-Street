# Paper Street: ニュースシステム詳細設計案

## 1. システムアーキテクチャと情報の流れ

ニュースは単なるテキスト表示ではなく、**市場価格を動かすトリガー**として機能させる必要があります。

**フロー:**

1.  **News Engine (Generator)**: ニュースを生成（定時 or ランダム）。
2.  **Database**: `news_feed` テーブルに保存。
3.  **WebSocket**: `news` チャンネルを通じて全クライアントに配信。
4.  **News Reactors (Bot)**: ニュースを受信し、センチメントに基づいて即座に注文を発注。
5.  **Market Impact**: Botの注文により板（Order Book）が食われ、価格が変動。
6.  **Player Action**: プレイヤーがニュースまたは値動きを見て追随。

---

## 2. ニュースの種類と生成ロジック

`init.sql` の構造および `world_setting.md` の世界観に基づき、3つのカテゴリに分類します。

### A. 定例経済イベント (Scheduled Events)

決算や指標発表など、発生時間が決まっているもの。プレイヤーはこれに向けてポジションを調整します。

*   **決算発表 (Earnings Release)**:
    *   **トリガー**: 各企業の `fiscal_quarter` 終了時（Day 7, 14等）。
    *   **データ**: 売上、EPS、ガイダンス。
    *   **ロジック**: 事前予測（Consensus）との乖離率（Surprise）でセンチメントを決定。
    *   **例**: 「OmniCorp (OMNI) Q3決算: EPSが予想を20%上回る。AI需要が牽引。」

*   **マクロ経済指標**:
    *   **トリガー**: 定期的な時刻（例: 毎時00分）。
    *   **データ**: CPI（インフレ率）、失業率、GDP。
    *   **影響**: 特定銘柄ではなく、セクター全体や国全体の通貨に影響。

### B. 突発的ニュース (Flash News)

ランダムに発生し、ボラティリティを生むイベント。

*   **ヘッドラインニュース**:
    *   `world_setting.md` のリスク設定に基づく。
    *   **例**: 「Boros連邦、Titan Energyへの輸出規制強化を示唆 (Sentiment: -0.8)」
    *   **例**: 「Neo Veniceで大規模停電発生。CyberLifeのサーバーダウン (Sentiment: -0.5)」

*   **噂 (Rumors) & 観測気球**:
    *   **特徴**: 真偽不明。センチメントスコアは高いが、後で「誤報（Correction）」が出る可能性がある。
    *   **ゲームプレイ**: リスクを取って飛び乗るか、事実確認（確定ニュース）を待つかの駆け引き。

### C. フレーバーテキスト (Fluff)

市場価格には影響しないが、世界観を深めるニュース。

*   **例**: 「Stardust Luxuryの新作ドレス、著名インフルエンサーが着用し話題に。」

---

## 3. 「情報の非対称性」の導入 (Information Asymmetry)

MMOとしての面白さを出すため、**「誰がいつニュースを知るか」**に差をつけます。

*   **Premium Feed (課金/上位ランク特典)**:
    *   一般プレイヤーより **3秒早く** ニュースを受信できる。
    *   `init.sql` の `fee_tier` が `VIP` のユーザー向け機能。

*   **Botの反応速度 (Latency)**:
    *   高性能Botはニュース発生から **10ms** で反応。
    *   低性能Botは **500ms〜2s** で反応。
    *   これにより、ニュース直後はBot同士の戦いで乱高下し、人間が介入する余地が生まれます。

---

## 4. センチメントとBotの挙動 (Math Model)

`init.sql` にある `sentiment_score` (-100 〜 +100 と仮定) をどう使うかの数式案です。

**News Reactor Bot のロジック:**

1.  **ニュース受信**: `headline` と `sentiment_score` を取得。
2.  **ターゲット価格算出**:

```python
# 影響力係数 (銘柄の時価総額や流動性による)
impact_factor = 0.05  # 5%

# 現在価格からの目標変動幅
target_delta = current_price * impact_factor * (sentiment_score / 100)

target_price = current_price + target_delta
```

3.  **注文執行**:
    *   **Positive (Score > 0)**: `target_price` までの売り板を成行買い (Market Buy)。
    *   **Negative (Score < 0)**: `target_price` までの買い板を成行売り (Market Sell)。

4.  **揺らぎ (Jitter)**:
    *   すべてのBotが同じ計算をすると価格が即座に張り付くため、Botごとに `impact_factor` に乱数（ノイズ）を混ぜます。

---

## 5. UI/UX デザイン案 (The Terminal)

`GAMEPLAY_AND_UI.md` にある「News Feed」ウィジェットの詳細仕様です。

*   **外観**: ブルームバーグ端末のような、黒背景に色付き文字のリスト表示。
*   **カラーコーディング**:
    *   **Positive**: 緑色 (Green)
    *   **Negative**: 赤色 (Red)
    *   **Neutral**: 白色 (White)
    *   **Breaking (重要)**: 黄色点滅 または 背景ハイライト

*   **タグ付け (Tags)**:
    *   `[EARNINGS]`, `[MACRO]`, `[RUMOR]`, `[WAR]` などのタグを先頭に付与。
    *   ユーザーはタグや保有銘柄（Related Assets）でフィルタリング可能。

---

## 6. コンテンツ作成のための具体例 (Content Ideas)

`world_setting.md` の企業設定を使ったニューステンプレート案です。

| Category | Headline Template | Sentiment | Target Asset |
| :--- | :--- | :--- | :--- |
| **Tech** | **OmniCorp** が次世代AIモデル「The Eye」を発表。処理能力が従来比300%向上。 | +0.80 | `OMNI` |
| **Energy** | **Titan Energy** のパイプラインで爆発事故。供給懸念から原油価格が上昇。 | +0.60 | `TTN`, `Crude Oil` |
| **Energy** | 原油価格上昇により、エネルギー輸入国である **Arcadia** の貿易収支が悪化。 | -0.30 | `ARC` (Currency) |
| **Bio** | **Chimera Genetics** の実験施設でバイオハザード警報。周辺地域が封鎖。 | -0.90 | `CHM` |
| **War** | **Boros** 軍が国境付近に集結。**Iron Fist Armaments** の株価が急騰。 | +0.70 | `IFA` |
| **Macro** | 中央銀行が政策金利を0.25%引き上げ。インフレ抑制へ。 | -0.40 | All Stocks, Bond Yield Up |
