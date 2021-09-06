import mysql from "mysql2/promise";

export function getDbInfo(batch: boolean): mysql.ConnectionOptions {
  return {
    host: process.env["MYSQL_HOSTNAME"] ?? "127.0.0.1",
    port: parseInt(process.env["MYSQL_PORT"] ?? "3306", 10),
    user: process.env["MYSQL_USER"] ?? "isucon",
    password: process.env["MYSQL_PASS"] || "isucon",
    database: process.env["MYSQL_DATABASE"] ?? "isucholar",
    timezone: "+00:00",
    decimalNumbers: true,
    multipleStatements: batch,
  };
}
