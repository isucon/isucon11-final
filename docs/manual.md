# ISUCON11 本選当日マニュアル

**本マニュアルならびに、[ISUCON11 本選レギュレーション](TODO: これあるんだっけ？) をご確認ください。**
**本マニュアルとレギュレーションの内容に矛盾がある場合、本マニュアルの内容が優先されます。**

## スケジュール

TODO: 見直す
- 10:00 競技開始
- 17:00 リーダーボードの更新停止
- 18:00 競技終了
- 翌 14:00 結果発表

## 課題アプリケーション ISUCHOLAR について

課題アプリケーション ISUCHOLAR の仕様については、[ISUCHOLAR アプリケーションマニュアル](TODO: gist link)を参照してください。

## ISUCON11 ポータルサイト（ポータル）

ISUCON11 の競技では下記のウェブサイトを利用します。事前に登録した情報を用いてログインしてください。
なお、このページは競技開始時刻までアクセスすることはできません。

ポータルでは、負荷走行（ベンチマーク）の実行/結果確認、質問/サポート依頼の送信、リーダーボードの確認ができます。

https://portal.isucon.net/contestant

### リーダーボードの更新について

ポータル上のリーダーボードは、競技終了 1 時間前に他チームの情報が更新されなくなり、自チームの情報のみ更新されます。

## Discord の利用について

ISUCON11 サポート Discord サーバーは競技中ならびにその前後の時間はすべてのチャンネルが発言不可となります。
競技時間中はポータルを通して質問/サポート依頼を送信することができますので、そちらを利用してください。

ただし、本選参加者（以下選手）は競技時間中も Discord の確認が可能な状態、通知が受け取れる状態を維持してください。
これは主催者が選手とリアルタイムでのチャットが必要だと判断した場合、主催者が Discord 上でプライベートチャンネルを作成しメンションの上、呼びかけを行う場合があるためです。呼びかけに応じない場合、競技に支障をきたす可能性があるため、必ず応答可能な状態を維持してください。

また、主催者からのアナウンス等も Discord で実施されます。

## 質問について

選手は主催者へ質問を送信することができます。質問は競技内容・マニュアル・レギュレーション等に対する疑問点の確認や、サーバー障害などのトラブル報告・サポート依頼に利用することができますが、これに限りません。

主催者は質問された内容が競技の一環である場合は、回答できない旨を返答することがあります。

競技時間中の質問について、主催者からの回答は全選手へ公開、あるいは個別に回答されます。全選手へ公開される場合、質問内容の原文、あるいは主催者による内容の要約が公開されます。未回答の質問・未公開の回答については質問した選手およびそのチームメンバー、主催者のみが確認できます。質問への回答・更新はポータル上にて選手およびそのチームへ通知されます。

主催者は競技時間中の質問への回答を、原則として全選手へ公開します。ただし、重複する質問や、選手およびチーム個別の問題に対する対応の場合、この限りではありません。

### サポート対象外の事項

主催者が事前に Discord サーバー等で告知していた通り、下記はサポート対象外となります。

- スポンサー各社が提供しているサービスについての質問/サポート依頼

## 競技環境について

本選では、競技に必要なすべてのサーバーを主催者が用意します。

### サーバー、ネットワーク構成

事前に用意されたサーバは 3 台あり（以下、競技用サーバー）、これらのサーバに SSH 接続し競技を行います。
ポータル右上に表示されているチーム ID を元に、以下の gist で割り当てられている「踏み台 IP アドレス」と「チームサブネット」を確認してください。

TODO: gist link

競技用サーバーのIPアドレスは、チームのサブネットに対してそれぞれ 101 ~ 103 が第 4 オクテットとなります。具体的な IP アドレスはポータルにて確認することができます。

競技者サーバーに接続するためには踏み台を経由して SSH 接続する必要があります。[ssh_config(5)](https://man.openbsd.org/ssh_config.5) の例を以下に示します。なお、あくまで例示であり必ず以下の設定を利用する必要はありません。

TODO: 後で見直す
```
 Host isucon-bastion
   HostName <踏み台用IPアドレス>
   Port 20340
   User isucon

 Host isucon-server
   ProxyJump isucon-bastion
   User isucon
   HostName <自チームサーバのIPアドレス>
```
なお、踏み台用のサーバはポート番号 20340 でログインすることができます。

### 競技環境の再構築について

選手自らが設定変更等により競技環境を破壊するなどして、初期状態に戻す必要がある場合は（TODO: どうする？）。再構築以前の競技環境上で変更を加えたソースコードや設定ファイル等の移行が必要な場合は、各チームの責任で行ってください。

### 重要事項

- **競技に利用できる計算機資源は主催者が用意した 3 台のサーバーのみです。**
  - レギュレーションに記載した通り、モニタリングやテスト、開発において外部の資源を用いても構いませんが（例： メトリクス計測サービス）、スコアを向上させるいかなる効果を持つものであってはいけません。
- **競技終了後は、主催者が追試を行います。Discord サーバー および https://isucon.net にて主催者がアナウンスをするまで、競技環境の操作はしないでください。**
  - 競技終了後、別途アナウンスがあるまでの作業は失格となります。

#### 変更してはいけない点

下記は追試や環境確認に利用するため、変更した場合は失格となります。

- `isucon-env-checker.service`に関わるファイル（TODO: こういう系っていらないんだっけ？）
    - `/etc/systemd/system/isucon-env-checker.service`
    - `/etc/systemd/system/multi-user.target.wants/isucon-env-checker.service`
    - `/opt/isucon-env-checker` 内のファイル
- その他、主催者による追試を妨げる変更（例： サーバー上の `isucon` 以外のユーザに関する、ユーザ削除や既存の公開鍵の削除）

## アプリケーションの動作確認

ISUCHOLAR には、サーバーの IP アドレス指定で Web ブラウザから HTTP アクセスできます。

### ISUCONDITION へのログイン

ISUCHOLAR には、学籍番号とパスワードを使ってログインしてください。

| タイプ | 学籍番号 | パスワード |
| ------ | -------- | ---------- |
| 学生   | isucon   | isucon     |

その他ユーザーのログイン情報については、[ユーザーのログイン情報一覧](TODO: gist link)を確認してください。

## 負荷走行 （ベンチマーク） の実行

負荷走行はポータル上からリクエストします。

[ポータル](https://portal.isucon.net/contestant) にアクセスし、「Job Enqueue Form」から負荷走行対象のサーバーを選択、「Enqueue」をクリックすることで負荷走行のリクエストが行われ、順次開始されます。

なお、負荷走行が待機中（PENDING）もしくは実行中（RUNNING）の間は追加でリクエストを行うことはできません。

## 参考実装

下記の言語での実装が提供されています。

TODO: 移植担当の方に聞く。
- Go
- Node.js
- Perl
- PHP
- Python
- Ruby
- Rust

### 参考実装の切り替え方法

初期状態では Go による実装が起動しています。

各言語実装は systemd で管理されています。
例えば、参考実装を Go から Ruby に切り替えるには以下のコマンドを実行します。

```shell
sudo systemctl disable --now isucholar.go.service

sudo systemctl enable --now isucholar.ruby.service
```

#### PHP への切り替え

ただし、PHP を使う場合のみ、systemd の設定変更の他に、次のように nginx の設定ファイルの変更が必要です。

```shell
sudo unlink /etc/nginx/sites-enabled/isucholar.conf
sudo ln -s /etc/nginx/sites-available/isucholar-php.conf /etc/nginx/sites-enabled/isucholar-php.conf
sudo systemctl restart nginx.service
```

### データベースのリカバリ方法

参考実装では、初期化処理（`POST /initialize`）においてデータベースを初期状態に戻します。
以下のコマンドでもデータベースを初期状態に戻すことができます。

TODO: 用意する？
```
~isucon/webapp/sql/init.sh
```

初期化処理は用意された環境内で、ベンチマーカーが要求する範囲の整合性を担保します。
サーバーサイドで処理の変更・データ構造の変更などを行う場合、この処理が行っている内容を漏れなく提供してください。
また、初期状態のデータベースの詳細は[`webapp/docs/schema`](TODO: gist link)で確認できます。

## 負荷走行について

ベンチマーカーによる負荷走行は以下のように実施されます。

1. 初期化処理の実行 `POST /initialize`（20 秒でタイムアウト）
2. アプリケーション互換性チェック（数秒～数十秒）
3. 負荷走行（60 秒）
4. 待ち時間（最大 10 秒）
5. 整合性チェック（数秒～数十秒）

ベンチマーカーは負荷走行終了後、5 秒待ってから整合性チェックを行います。
負荷走行終了後、5 秒経ってもレスポンスが返ってきていないリクエストはすべて強制的に切断され、タイムアウトとして数えられます。

初期化処理、アプリケーション互換性チェック、整合性チェックのいずれでも失敗すると、負荷走行は即時失敗（fail）になります。

### リダイレクトについて

ベンチマーカーは HTTP リダイレクトを処理しません。

### キャッシュについて

ベンチマーカーは下記を除き、データの更新が即時反映されていることを期待して検証を行います。ただし、アプリケーションはベンチマーカーが検知しない限りは古い情報を返しても構いません。

#### TODO: キャッシュの許容について書く。

#### Conditional GET のサポートについて

ベンチマーカーは一般的なブラウザの挙動を模した [Conditional GET](https://tools.ietf.org/html/rfc7232) に対応しています。
アプリケーションは、 `Cache-Control` やその他必要なレスポンスヘッダを返すことで、ベンチマーカーから Conditional GET リクエストを受けることができます。
データが更新されていないことが期待されるリクエストにおいては、`304 Not Modified` を返したり、あるいはブラウザのキャッシュ有効期限の制御によってリクエストが発生していない場合も、ベンチマーカーはそれらのキャッシュを利用してレスポンスがあったものとみなします。

なお、ベンチマーカー内のユーザは独立しているため、 `Cache-Control: public` 等が指定されていたとしても、ユーザ同士でキャッシュを共有することはありません。

### タイムアウトについて

負荷走行において設定されているタイムアウト値は下記の通りです。

- `POST /initialize`
  - 20 秒以内にレスポンスを返す必要があります。これを超えた場合、負荷走行は即時失敗（fail）します。
- TODO: 考える。
- 上記以外の HTTP リクエスト
  - 5 秒以内にレスポンスを返す必要があります。これを超えた場合、後述のスコア計算に従い減点の対象となります。

### 負荷走行における仮想時間について

ベンチマーカー上では現実の n 万倍の速度で時間が流れており（現実の 1 秒がベンチマーカー上では n 秒）、TODO: ちょうど1年ぐらいにしたい

### 動作確認用ユーザー

ベンチマーカーは、学生の 1 人として isucon を必ず利用します。
isucon ユーザーを用いることで、負荷走行後のアプリケーションの状態確認が可能です。

## スコア計算

アプリケーション互換性チェックを通過し負荷走行が開始されると、下記の通り加点・減点が行われます。

### 負荷走行時における加点について

TODO: 加点について書く。

### 負荷走行時における減点、即時失敗（fail）について

下記のエラーは減点や即時失敗（fail）となります。fail となった場合スコアは 0 点となります。

- 初期化処理、アプリケーション互換性チェック、整合性チェックのどれかに失敗した場合
  - 1 回以上で fail

- HTTP ステータスコードやレスポンス内容などに誤りがある場合
  - 1 回あたり減点 1 点
  - 100 回の失敗時点で fail

- リクエストがタイムアウトした場合（[タイムアウトについて](#タイムアウトについて)）
  - 10 回あたり減点 1 点
  - fail は発生しない
  - TODO: 考える。

### 最終スコア

競技終了後、主催者は __全サーバーの再起動後に負荷走行を実施__ します。その際のスコアを最終スコアとします。

#### 追試

最終スコアが確定後、主催者による確認作業（追試）を行います。下記の点が確認できなかった場合は fail となります。

- 負荷走行実行時にアプリケーションに書き込まれたデータはサーバー再起動後にも取得できること
- アプリケーションはブラウザ上での挙動を初期状態と同様に保っていること

### その他

#### `POST /initialize` での実装言語の出力

`POST /initialize` レスポンスにて、本競技で利用した言語を出力してください。参考実装はそのようになっています。この情報は集計し [ISUCON 公式Blog](https://isucon.net/) での公表や、参考情報として利用させていただきます。

`POST /initialize` のレスポンスは以下のような JSON となります。

```json
{
    "language": "実装言語"
}
```

`language` の値が実装に利用した言語となります。 `language` が空の場合は初期化処理が失敗と見なされます。

## 禁止事項

以下の行為を特に禁止する。

- 競技終了時間までに、競技の内容に関するあらゆる事項（問題内容・計測ツールの計測方法など）を公開・共有すること（内容を推察できる発言も含む）
  - 不特定多数への公開はもちろん、他チームの選手と連絡を取り、問題内容等を共有する事（結託行為）も禁止とする。
  - ただし主催者が Twitter, Web サイトにおいて公開している情報は除く。ポータルでログインを要するページ（選手が参加する Discord を含む）において記載されている内容は公開情報でない旨留意すること
- 競技時間中、チーム外の人物と ISUCON11 問題にまつわる事項のやりとり（ISUCON11 選手であるかどうかを問わない、SNS での発言も含む）
- 主催者の指示以外で利用が認められたサーバー以外の外部リソースを使用する行為（他のインスタンスに処理を委譲するなど) は禁止する。
  - ただしモニタリングやテスト、開発などにおいては、PC や外部のサーバーを利用しても構わない。
- 選手が主催者からその選手が属するチームへ提供されていないサーバーについて直接のアクセスを試みる行為や、外部への不正アクセスを試みる行為。具体的にはベンチマーカーへのログイン試行等。（なお、例示のため、これに限らない）
- 他チームと結託する行為（程度を問わず）
- 主催者が他チームへの妨害、競技への支障となるとみなす全ての行為

本マニュアルやレギュレーション、ポータルにおいて禁止とされた行為（禁止事項）への違反は、失格となる。