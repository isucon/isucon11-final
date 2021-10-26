# ISUCHOLAR - isucon11-final

## ディレクトリ構成

競技用サーバーにはwebapp以下が配布されます。

```
.
├── benchmarker   # ベンチマーカー
├── dev           # 開発用設定ファイル等
├── docs          # 当日マニュアル等
├── provisioning  # デプロイスクリプト
└── webapp        # 各参考実装
```

## ISUCON11 本選の競技環境について

### マシンスペック

+ ベンチマーカー
  + インスタンスタイプ: c5.xlarge
  +  EBS: gp3 20GB
+ 競技用サーバー 3台
  + インスタンスタイプ: c5.large
  +  EBS: gp3 30GB

ただし、競技用サーバーのメモリは元の 4GB から 2GB に制限されています。

### Prerequirements

+ ベンチマーカー
  + Go
+ webapp
  + 各種実装言語
  + zip コマンド

### 負荷走行の実行

```
cd benchmarker
make
./bin/benchmarker -target {対象 IP アドレス}
```
