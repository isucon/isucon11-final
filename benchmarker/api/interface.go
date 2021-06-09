package api

// interface.goではapiパッケージで実装するリクエストアクションの関数型を載せています。
// DI機構は現状不要だと思っているのでinterfaceとしては実装はしていませんが、アクションの実装の参考にしてください。
// *http.Responseを返却しているのはレスポンス検証時(scenario/action.go)のエラーログに原因のpath/methodなどを出力させたいためです。
// *http.ResponseのBodyはapi内で読み切ってください。

// T.B.D.
// browserAccess(ctx context.Context, a *agent.Agent, path string) (agent.Resources, *http.Response, error)
// ユーザ操作によるブラウザアクセスアクション。静的ファイルまで読み込みを行う。

// FuncName(ctx context.Context, (a *agent.Agent,) requestData ..interface{}) (*http.Response, error)
// レスポンスボディを検証する必要がないリクエストアクション。レスポンスボディが不要ですBodyは読み捨てすること。（AgentClientのコネクションが切れます）
// ex) ログイン, 出席コード入力/登録, ...

// FuncName(ctx context.Context, a *agent.Agent, requestData ..interface{}) (interface{}, *http.Response, error)
// レスポンスボディのデータが必要なリクエストアクション。レスポンスStructureはapiパッケージで定義する。
// ex) 科目検索, お知らせ詳細
