import mysql, { RowDataPacket } from "mysql2/promise";

const dbinfo: mysql.PoolOptions = {
  host: process.env["MYSQL_HOSTNAME"] ?? "127.0.0.1",
  port: parseInt(process.env["MYSQL_PORT"] ?? "3306", 10),
  user: process.env["MYSQL_USER"] ?? "isucon",
  database: process.env["MYSQL_DATABASE"] ?? "isucholar",
  password: process.env["MYSQL_PASS"] || "isucon",
  connectionLimit: 10,
  timezone: "+00:00",
};

const pool = mysql.createPool(dbinfo);

const UserType = {
  Student: "student",
  Teacher: "teacher",
} as const;
type UserType = typeof UserType[keyof typeof UserType];

type UUID = Buffer;

interface User extends RowDataPacket {
  id: UUID;
  code: string;
  name: string;
  hashed_password: Buffer;
  type: UserType;
}

(async () => {
  const db = await pool.getConnection();
  try {
    const [rows] = await db.query<User[]>("SELECT * FROM `users`");
    rows.forEach((row) => {
      console.log(`[${row.code}]: isAdmin: ${row.type === UserType.Teacher}`);
    });
  } finally {
    db.release();
    pool.end();
  }
})();
