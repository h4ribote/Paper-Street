# News System Format & Patterns

## 概要 (Overview)

本ドキュメントでは、Paper Street のニュースシステムで使用されるデータフォーマットと、`docs/news_patterns.json` に定義されたニュースパターンの構造について解説します。

このJSONファイルは、ゲーム内のニュースフィード（News Feed）に表示されるヘッドラインと本文を生成するためのテンプレート集です。システムはこれらのテンプレートを読み込み、状況に応じて変数を埋め込むことで、無限に近いバリエーションのニュースを生成します。


## JSON構造 (Schema)

`news_patterns.json` のルートオブジェクトは以下の構造を持ちます。

```json
{
  "version": "1.1",
  "meta": {
    "languages": ["en", "ja"],
    ...
  },
  "categories": {
    "CATEGORY_ID": {
      "id": "CATEGORY_ID",
      "name_en": "English Name",
      "name_ja": "Japanese Name",
      "description": "Description of the category",
      "patterns": [
        {
          "id": "PATTERN_ID",
          "headline_template_en": "English String with {variables}",
          "headline_template_ja": "日本語文字列 {variables}",
          "body_template_en": "English String with {variables}",
          "body_template_ja": "日本語文字列 {variables}",
          "sentiment_range": [min, max],
          "impact_scope": ["SCOPE_1", "SCOPE_2"],
          "variables": ["var1", "var2"]
        },
        ...
      ]
    }
  }
}
```

### フィールド詳細

*   **categories**: ニュースのカテゴリごとの定義を含むオブジェクト。キーはカテゴリID（例: `EARNINGS`, `MACRO`）。
*   **id**: カテゴリまたはパターンのユニークID。
*   **name_en / name_ja**: カテゴリの表示名（英語/日本語）。
*   **headline_template_en / _ja**: ニュースのタイトル（英語/日本語）。`{variable}` の形式でプレースホルダーを含みます。
*   **body_template_en / _ja**: ニュースの本文（英語/日本語）。同様にプレースホルダーを含みます。
*   **sentiment_range**: このニュースが市場に与える心理的影響の範囲。`[-1.0, 1.0]` の配列。生成時にこの範囲内でランダムな値が決定されます。
    *   `1.0`: 極めてポジティブ（買い）
    *   `-1.0`: 極めてネガティブ（売り）
    *   `0.0`: 中立（影響なし）
*   **impact_scope**: ニュースが影響を与える対象のリスト。
    *   `{ticker}`: 特定の銘柄（変数値で置換される）
    *   `{sector}`: 特定のセクター
    *   `ALL_STOCKS`: 全株式市場
    *   `{currency}`: 特定の通貨
*   **variables**: テンプレート内で使用されている変数のリスト。


## 多言語対応 (Localization)

本システムは日本語と英語に対応しています。

1.  **テンプレートの選択**: システムはユーザーの言語設定（Locale）またはサーバーのデフォルト設定に基づいて、`_en` または `_ja` のサフィックスが付いたテンプレートを選択します。
2.  **変数の共通化**: 変数（`{company_name}` など）は言語間で共通です。ただし、変数の値自体（例: 企業名）を挿入する際は、可能な限りその言語に適した表記を使用することが推奨されます（例: "OmniCorp" vs "オムニコープ"）。


## 変数リスト (Variables)

テンプレート内で使用される主な変数は以下の通りです。実装時には、`world_setting.md` やデータベースの値を元にこれらを置換してください。

| 変数名 | 説明 | 例 |
| :--- | :--- | :--- |
| `{company_name}` | 企業名 | OmniCorp, Titan Energy |
| `{ticker}` | ティッカーシンボル | OMNI, TTN |
| `{sector}` | 産業セクター | Tech, Energy, Bio |
| `{quarter}` | 四半期 (1-4) | 1, 2, 3, 4 |
| `{eps}` | 一株当たり利益 (実績) | 2.50 |
| `{eps_est}` | 一株当たり利益 (予想) | 2.30 |
| `{revenue}` | 売上高 (実績) | 10.5 |
| `{revenue_est}` | 売上高 (予想) | 10.0 |
| `{country}` | 国名 | Arcadia, Boros Federation |
| `{currency}` | 通貨コード | ARC, BRB |
| `{central_bank}` | 中央銀行名 | Bank of Arcadia |
| `{rate}` | 金利 (%) | 5.25 |
| `{product}` | 製品名 | AI Chips, Synth-Meat |
| `{person_name}` | 人名 (CEO, 政治家等) | John Doe |
| `{location}` | 地名 | Neo Venice, Sector 7 |


## カテゴリ一覧 (Categories)

1.  **EARNINGS (決算発表)**
    *   企業の四半期決算。売上や利益が予想を上回るか下回るかで株価が大きく動く。
2.  **MACRO (マクロ経済指標)**
    *   GDP、インフレ率(CPI)、失業率など。国全体の通貨価値や株価指数に影響。
3.  **CENTRAL_BANK (中央銀行政策)**
    *   金利の引き上げ・引き下げ。市場全体の流動性とボラティリティに最大の影響を与える。
4.  **GEOPOLITICS (地政学)**
    *   国家間の紛争、条約、選挙。関連国の通貨や防衛産業株に影響。
5.  **MARKET (市場概況)**
    *   セクターごとのトレンドやアナリストの評価。
6.  **FLASH (突発ニュース)**
    *   事故、災害、技術革新、サイバー攻撃など。予測不能な急変動を引き起こす。
7.  **FLAVOR (フレーバー)**
    *   文化、芸能、流行など。市場への直接的な影響はないが、世界観を深める要素。
