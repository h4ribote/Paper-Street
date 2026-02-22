# Paper Street: World Setting & Lore

このドキュメントでは、Paper Streetの世界観、架空の国家、企業、および発生するイベントの定義を行います。
プレイヤーがニュースや指標を見た際に、「この国ならありそうだ」「この企業ならこう動くだろう」という予測や愛着を持てるような魅力的な設定を目指します。

## 1. 世界の概観 (Geopolitical Overview)
技術発展が進んだ近未来、または現代と並行する別の世界線。
国家間の対立は武力よりも経済・情報戦争が主軸となっており、巨大企業が国家に匹敵する影響力を持つ場合もあります。

地域（Region）ごとの特色を分け、ポートフォリオの分散投資（地域分散）に意味を持たせます。

### 地域区分 (Regions)
1.  **Northern Alliance (北部先進連合)**
    *   **特徴**: 高度な技術力、成熟した金融市場、安定した政治。しかし成長率は鈍化しており、少子高齢化や規制の厳しさが課題。
    *   **モデル**: 北米、西欧、日本
2.  **Eastern Coalition (東部産業同盟)**
    *   **特徴**: 世界の工場。急激な成長、強力な政府主導の経済、豊富な労働力。環境問題や人権問題のリスクを抱える。
    *   **モデル**: 中国、東南アジア、インド
3.  **Southern Resource Pact (南部資源協定)**
    *   **特徴**: 石油、レアアース、食料の主要供給源。資源価格に経済が大きく左右される。政治腐敗や紛争のリスクが高い。
    *   **モデル**: 中東、南米、アフリカ
4.  **Oceanic Tech Arch (海洋技術諸島)**
    *   **特徴**: 税制優遇（タックスヘイブン）と規制緩和により、テック企業やブロックチェーン企業が集まる。ボラティリティが激しい。
    *   **モデル**: シンガポール、ドバイ、ケイマン諸島

## 経済と通貨の相互作用 (Economic Interdependence)
本作の経済システムにおいて最も重要な特徴の一つは、**取引される商品（株式や資源）によって、決済に使用される通貨が異なる**という点です。
各商品はその発行体である企業の拠点、または資源の主要産出国の通貨で価格が設定され、取引されます。

この仕組みにより、商品市場の動向は為替市場にダイレクトに波及します。

*   **資源国通貨の連動**: 例えば、原油価格が高騰すれば、原油の決済通貨である `Boros Ruble (BRB)` の需要が増加し、BRB高・他通貨安の圧力となります。
*   **企業業績と通貨**: 特定の巨大企業の決算が好調であれば、その企業の株式を購入するために本拠地通貨（例: `Arcadian Credit (ARC)`）が買われます。
*   **リスクオフ**: 知政学リスクが高まった際、安全資産とされる資源（金や食料など）の産出国通貨が買われる傾向があります。

プレイヤーは単に商品価格の変動だけでなく、その商品の決済通貨の為替レート変動も考慮に入れた複合的なトレード戦略を立てる必要があります。

### 初期資産配分 (Initial Asset Allocation)
各国家や企業の経済規模（初期資産）については、[Initial Asset Allocation](./design/INITIAL_ASSET_ALLOCATION.md) を参照してください。プレイヤー資産が市場全体の5%以内となるように調整されています。


## 2. 国家タイプ (Country Archetypes)
各国家には以下のパラメータが設定され、経済指標やイベント発生率に影響します。

| タイプ | 特徴 | 長所 | 短所 | 市場介入スタンス | 発生しやすいイベント |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Tech Utopia** | 技術立国。AI・バイオ・ロボティクスが産業の中心。 | 高い生産性、通貨高、低インフレ | 失業率上昇（自動化）、電力不足、サイバー攻撃 | **不干渉**: 自由市場を信奉するが、システム障害時には緊急停止を行う。 | 技術革新、AI暴走、サーバーダウン |
| **Industrial Giant** | 重厚長大産業。輸出主導型経済。 | 安定した貿易黒字、高い雇用率 | 環境汚染、通貨安政策、貿易摩擦 | **通貨安誘導**: 輸出競争力を維持するため、積極的に自国通貨売り介入を行う。 | 公害訴訟、関税戦争、資源価格高騰 |
| **Resource Kingdom** | 資源輸出に依存。政府が富を独占。 | 資源高時の爆発的利益、無税国家 | 資源安時の財政破綻、独裁政治、貧富の差 | **固定相場維持**: 通貨価値を資源価格と連動させようと強引な介入を行う。 | クーデター、国有化、資源枯渇 |
| **Agri-Republic** | 食料生産大国。牧歌的だが食料安全保障の要。 | 景気に左右されにくい、安定した需要 | 天候リスク、低いイノベーション、労働力不足 | **補助金**: 農産物価格維持のために買い支えを行う。通貨介入は消極的。 | 干ばつ/洪水、疫病、食料価格高騰 |
| **Financial Hub** | 金融・観光立国。資源はないが金が集まる。 | 通貨流通量が多い、海外投資が活発 | 金融危機に弱い、マネーロンダリング温床 | **流動性供給**: 金融危機時に量的緩和を行い、市場のパニックを沈静化させる。 | 金融スキャンダル、バブル崩壊、規制強化 |


## 3. 架空国家リスト (Fictional Countries)
**Note**: 国家の数は固定されており、ゲーム進行中に新たに追加されたり、既存の国家が消滅したりすることはありません（定数として扱います）。

### 1. Arcadia (アルカディア)
*   **地域**: Northern Alliance
*   **タイプ**: Tech Utopia
*   **通貨**: Arcadian Credit (ARC)
*   **詳細**:
    *   完全なデジタル社会を実現した先進国。国民IDと銀行口座が統合されており、脱税が不可能。
    *   主要産業はAI開発と量子コンピューティング。
    *   **リスク**:  高度にネットワーク化されているため、大規模なシステム障害（Blackout）が起きると経済が停止する。

### 2. Boros Federation (ボロス連邦)
*   **地域**: Eastern Coalition
*   **タイプ**: Industrial Giant
*   **通貨**: Boros Ruble (BRB)
*   **詳細**:
    *   広大な国土と人口を持つ軍事・産業大国。重工業と兵器産業が盛ん。
    *   政府の権限が強く、外国企業の資産凍結などの強権的な動きを見せることがある。
    *   **リスク**: 隣国との領土紛争、経済制裁の発動。

### 3. El Dorado (エルドラド)
*   **地域**: Southern Resource Pact
*   **タイプ**: Resource Kingdom
*   **通貨**: Dorado Real (DRL)
*   **詳細**:
    *   世界最大のレアメタル埋蔵量を誇る。電子機器の生産に不可欠な素材を独占している。
    *   国王による絶対王政。市民への還元は手厚いが、言論の自由はない。
    *   **リスク**: 国王の健康不安、王位継承争いによる内戦。

### 4. Neo Venice (ネオ・ヴェネツィア)
*   **地域**: Oceanic Tech Arch
*   **タイプ**: Financial Hub
*   **通貨**: Venice Dollar (VND)
*   **詳細**:
    *   海上に建設された人工浮遊都市群。世界中の富裕層が移住している。
    *   法人税ゼロ、仮想通貨の利用が完全に自由。
    *   **リスク**: 海面上昇による都市水没の危機、国際的なマネーロンダリング規制のターゲット。

### 5. San Verde (サン・ヴェルデ)
*   **地域**: Southern Resource Pact
*   **タイプ**: Agri-Republic
*   **通貨**: Verde Peso (VDP)
*   **詳細**:
    *   広大なプランテーションを持つ農業国。遺伝子組み換え作物の実験場となっている。
    *   主要輸出品は遺伝子組み換え大豆と合成肉。
    *   **リスク**: 干ばつ、病害虫、他国企業による農地や水源の買収。

### 6. Novaya Zemlya (ノヴァヤ・ゼムリャ)
*   **地域**: Northern Alliance
*   **タイプ**: Energy / Industrial
*   **通貨**: Zemlya Ruble (ZMR)
*   **詳細**:
    *   極寒の地に建設された巨大エネルギー都市。地熱発電と原子力発電のメッカ。
    *   豊富なエネルギーを背景に、データセンターや重工業が集積している。
    *   **リスク**: 原子力事故、冷却水不足、永久凍土の融解によるインフラ崩壊。

### 7. Pearl River Zone (パールリバー特区)
*   **地域**: Eastern Coalition
*   **タイプ**: Industrial Giant / Tech
*   **通貨**: River Dollar (RVD)
*   **詳細**:
    *   世界最大の電子機器生産拠点。サイバネティクス技術の実用化特区。
    *   24時間稼働の工場群と、ネオン輝くサイバーパンクな街並みが特徴。
    *   **リスク**: 労働争議、産業スパイ、新型ウイルスの発生源。


## 4. 企業・セクター (Sectors & Companies)
**Note**: 企業のリストは固定されており、ゲーム進行中に新規上場や倒産によって企業数が増減することはありません（定数として扱います）。

### Technology (TECH)
*   **OmniCorp (OMNI)**: [Arcadia] 世界最大のAI企業。生活のあらゆるインフラを握っている。「Omni sees everything.」
    *   **Supply Chain**: 半導体原料としてレアメタル（El Dorado）を大量に消費。電力価格の影響を強く受ける。
*   **Quantum Dynamics (QDY)**: [Arcadia] 量子コンピューティングのハードウェアメーカー。OmniCorpと競合しつつも、AI開発に必要な計算能力を提供している。
    *   **Supply Chain**: 極低温環境を維持するため、Novaya Zemlyaの冷却システム技術に依存。
*   **CyberLife (CYB)**: [Neo Venice] 人工義体やサイバネティクス技術のトップシェア。倫理的な問題を度々起こす。
    *   **Supply Chain**: Titan Energyから特殊な合成燃料を購入。
*   **Silicon Dragon (SLD)**: [Pearl River] 半導体およびマイクロチップの世界最大手。
    *   **Supply Chain**: El Doradoからレアメタルを輸入し、OmniCorpやAegis Systemsにチップを供給。

### Energy (ENERGY)
*   **Titan Energy (TTN)**: [Boros] 旧時代の化石燃料採掘大手だが、再生可能エネルギーへの転換を進めている。
    *   **Supply Chain (Upstream)**: 原油・天然ガスを世界中に供給。特にBoros国内の重工業（Iron Fist Armaments）への供給責任を持つ。
*   **Nebula Mining (NEB)**: [Novaya Zemlya] 小惑星採掘ベンチャー。レアメタル供給の新たな源泉として期待されるが、コストが高く失敗リスクも高い。
    *   **Supply Chain (Upstream)**: 宇宙空間から持ち帰った鉱物を精錬し、Silicon DragonやAegis Systemsに供給。
*   **Helios Solar (SOL)**: [El Dorado] 砂漠地帯を利用した大規模太陽光発電を行う。
    *   **Supply Chain (Upstream)**: クリーンエネルギーをOmniCorpのデータセンター群に供給。
*   **Atomos Energy (ATM)**: [Novaya Zemlya] 次世代原子炉と地熱発電を運営。安価で安定したベースロード電源を供給。
    *   **Supply Chain (Upstream)**: 世界中の電力会社やエネルギー多消費産業（データセンター等）に供給。

### Finance (FIN)
*   **Goliath Bank (GLT)**: [Arcadia] 世界最大の投資銀行。「Too big to fail」の代名詞。
*   **Shadow Fund (SHD)**: [Neo Venice] 実態のつかめないヘッジファンド。アルゴリズム取引で市場を支配する。

### Healthcare / Bio (BIO)
*   **Chimera Genetics (CHM)**: [Arcadia] 遺伝子編集技術のパイオニア。「デザイナーベビー」問題で倫理的な議論を呼ぶ。寿命延長技術を富裕層に提供。
*   **Verde Pharma (VPH)**: [San Verde] 農業廃棄物を利用した安価なジェネリック医薬品メーカー。途上国向けのシェアが高い。
    *   **Supply Chain**: Red Ox Foodから廃棄される有機物を原料として安く調達している。
*   **Panacea Corp (PAN)**: [El Dorado] ジャングル奥地の希少植物から特効薬を開発する製薬会社。特許を独占し、高額で販売している。
    *   **Supply Chain**: ジャングルの植生保護のため、San Verdeの農業企業と対立することがある。

### Consumer / Retail (CONS)
*   **Stardust Luxury (SDL)**: [Neo Venice] 富裕層向けの超高級ブランド。宇宙旅行用のドレスやサイボーグ用装飾品を販売。
    *   **Supply Chain**: CyberLifeから装飾用パーツを仕入れている。
*   **Red Ox Food (ROX)**: [San Verde] 世界最大の食肉加工・穀物商社。食料危機が起きると逆に利益を上げる構造を持つ。
    *   **Supply Chain (Upstream)**: 世界中の食品メーカーに原材料を供給。
*   **Global News Network (GNN)**: [Neo Venice] 世界的なメディアコングロマリット。情報の偏向報道で世論を操作しているとの噂が絶えない。
    *   **Supply Chain**: 通信インフラ企業に依存。

### Defense / Industrial (DEF)
*   **Iron Fist Armaments (IFA)**: [Boros] ボロス連邦国営の兵器メーカー。安価で頑丈な戦車や銃器を紛争地帯に輸出。
    *   **Supply Chain**: Titan Energyからエネルギーを、国内鉱山から鉄鉱石を調達。鋼材価格の高騰がコスト増になる。
*   **Aegis Systems (AEG)**: [Arcadia] 最新鋭の自律型ドローン兵器とミサイル防衛システムを開発。
    *   **Supply Chain**: OmniCorpからAIチップの供給を受ける。

### Logistics / Transport (LOG)
*   **Trans-Oceanic (TRN)**: [Neo Venice] 海上物流の最大手。完全無人化された巨大コンテナ船団を運航。
*   **Void Cargo (VDC)**: [Arcadia] 軌道エレベーター建設に関わる宇宙輸送ベンチャー。ロケット打ち上げ失敗で株価が急落することがある。
*   **Horizon Logistics (HRZ)**: [Arcadia] ドローン配送ネットワークを運営。ラストワンマイル物流を支配。
    *   **Supply Chain**: Aegis Systemsのドローン技術を民生転用。


## 5. イベント・シナリオ素案 (Scenario Hooks)

### 経済イベント
*   **Tech Bubble Burst**: Arcadiaの主要テック企業の不正会計が発覚し、テック株が全面安になる。
*   **Resource War**: El Doradoがレアメタルの輸出制限を発表。テック製品の価格が高騰し、Borosとの緊張が高まる。
*   **Digital Currency Crisis**: Neo Veniceの取引所がハッキングされ、巨額の仮想通貨が流出。信用収縮が起こる。

### 政治イベント
*   **Boros Election**: ボロス連邦の大統領選。現職（強硬派）か対立候補（改革派）かによって、軍事費や貿易政策が大きく変わる。
*   **Arcadia Privacy Act**: アルカディアで厳格なプライバシー法が成立。データを売買するビジネスモデルの企業が大打撃を受ける。
*   **El Dorado Succession**: 国王が急死し、親北派の長男と親東派の次男による対立が激化。内戦リスクで通貨DRLが暴落。


## 6. 資源・コモディティ (Resources & Commodities)
ゲーム内で取引・消費される主要な資源リスト。これらは`Assets`として取引されるほか、企業の業績（`Supply Chain`）に直接影響を与えます。

### Energy (エネルギー)
*   **Crude Oil (原油)**: [Titan Energy] 旧世界の血液。プラスチック原料としても必須。
*   **Natural Gas (天然ガス)**: [Titan Energy] 火力発電や暖房用。
*   **Hydrogen (水素)**: [Helios Solar] クリーンエネルギーの貯蔵・輸送媒体として注目される。
*   **Uranium (ウラン)**: [Atomos Energy] 原子力発電の燃料。
*   **Energy Credits (電力証書)**: [Helios Solar, Atomos Energy] データセンターやAI企業の稼働に不可欠。

### Metals / Materials (金属・素材)
*   **Rare Earth Metals (レアアース)**: [El Dorado] 半導体、バッテリー、先端兵器の製造に必須。供給不足はテックセクターに打撃。
*   **Lithium (リチウム)**: [El Dorado] EVバッテリーやモバイル機器に必須。Nebula Miningも宇宙からの供給を目指している。
*   **Steel (鉄鋼)**: [Boros] ビル建設、兵器、自動車などの基礎素材。
*   **Water (工業用水/飲料水)**: [San Verde, Novaya Zemlya] 気候変動により価値が高騰。半導体洗浄にも大量に必要。

### Agriculture / Food (農業・食料)
*   **Grain (穀物)**: [San Verde] 小麦、トウモロコシなど。食料危機の指標。
*   **Synthetic Meat (合成肉)**: [Red Ox Food] 培養肉。環境負荷が低いが、食の安全性が問われることもある。

### Technology / Components (技術・部品)
*   **Semiconductors (半導体)**: [Silicon Dragon] 「産業の米」。あらゆる電子機器に搭載される。
*   **Cyber-Implants (サイバーインプラント)**: [CyberLife] 人体拡張用パーツ。医療・軍事両面で需要がある。
*   **Bio-Gel (バイオジェル)**: [Chimera Genetics] 医療ポッドや再生治療に使われる特殊素材。
