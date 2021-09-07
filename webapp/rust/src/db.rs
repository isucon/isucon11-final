use futures::StreamExt as _;

/*
 * sqlx の MySQL ドライバには
 *
 * - commit()/rollback() していないトランザクション (sqlx::Transaction) が drop される
 *   - このとき drop 後に自動的に ROLLBACK が実行される
 * - fetch_one()/fetch_optional() のように MySQL からのレスポンスを最後まで読まない関数を最後に使っ
 *   ている
 *
 * の両方を満たす場合に、sqlx::Transaction が drop された後に panic する不具合がある。
 * panic しても正常にレスポンスは返されておりアプリケーションとしての動作には影響無い。
 *
 * この不具合を回避するため、fetch() したストリームを最後まで詠み込むような
 * fetch_one()/fetch_optional() をここで定義し、アプリケーションコードではトランザクションに関して
 * これらの関数を使うことにする。
 *
 * 上記のワークアラウンド以外にも、sqlx::Transaction が drop される前に必ず commit()/rollback() を
 * 呼ぶように気をつけて実装することでも不具合を回避できる。
 *
 * - https://github.com/launchbadge/sqlx/issues/1078
 * - https://github.com/launchbadge/sqlx/issues/1358
 */

pub async fn fetch_one_as<'q, 'c, O>(
    query: sqlx::query::QueryAs<'q, sqlx::MySql, O, sqlx::mysql::MySqlArguments>,
    tx: &mut sqlx::Transaction<'c, sqlx::MySql>,
) -> sqlx::Result<O>
where
    O: 'q + Send + Unpin + for<'r> sqlx::FromRow<'r, sqlx::mysql::MySqlRow>,
{
    match fetch_optional_as(query, tx).await? {
        Some(row) => Ok(row),
        None => Err(sqlx::Error::RowNotFound),
    }
}

pub async fn fetch_one_scalar<'q, 'c, O>(
    query: sqlx::query::QueryScalar<'q, sqlx::MySql, O, sqlx::mysql::MySqlArguments>,
    tx: &mut sqlx::Transaction<'c, sqlx::MySql>,
) -> sqlx::Result<O>
where
    O: 'q + Send + Unpin,
    (O,): for<'r> sqlx::FromRow<'r, sqlx::mysql::MySqlRow>,
{
    match fetch_optional_scalar(query, tx).await? {
        Some(row) => Ok(row),
        None => Err(sqlx::Error::RowNotFound),
    }
}

pub async fn fetch_optional_as<'q, 'c, O>(
    query: sqlx::query::QueryAs<'q, sqlx::MySql, O, sqlx::mysql::MySqlArguments>,
    tx: &mut sqlx::Transaction<'c, sqlx::MySql>,
) -> sqlx::Result<Option<O>>
where
    O: Send + Unpin + for<'r> sqlx::FromRow<'r, sqlx::mysql::MySqlRow>,
{
    let mut rows = query.fetch(tx);
    let mut resp = None;
    while let Some(row) = rows.next().await {
        let row = row?;
        if resp.is_none() {
            resp = Some(row);
        }
    }
    Ok(resp)
}

pub async fn fetch_optional_scalar<'q, 'c, O>(
    query: sqlx::query::QueryScalar<'q, sqlx::MySql, O, sqlx::mysql::MySqlArguments>,
    tx: &mut sqlx::Transaction<'c, sqlx::MySql>,
) -> sqlx::Result<Option<O>>
where
    O: 'q + Send + Unpin,
    (O,): for<'r> sqlx::FromRow<'r, sqlx::mysql::MySqlRow>,
{
    let mut rows = query.fetch(tx);
    let mut resp = None;
    while let Some(row) = rows.next().await {
        let row = row?;
        if resp.is_none() {
            resp = Some(row);
        }
    }
    Ok(resp)
}
