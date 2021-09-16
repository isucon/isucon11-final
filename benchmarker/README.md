# isucon11-benchmarker

## Build
```
$ make build
```

## Benchmark
```
# localhost:8080 (デフォルト) に向けてベンチマーク実行
$ ./bin/benchmarker
```

## Generator
### 静的ファイルのチェックサム生成
```
$ make assets
```

### 初期ユーザ生成
```
$ gem install forgery_ja securerandom bcrypt ulid
$ ruby tools/gen_user_data.rb
```

### 初期科目作成
初期ユーザを生成し、 `/genenate/data/` 以下にTSVを配置してから実行する。
```
$ make initial_course
```
