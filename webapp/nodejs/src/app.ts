import { readFile } from "fs/promises";

import express from "express";
import morgan from "morgan";
import mysql from "mysql2/promise";

import { getDbInfo } from "./db";

const SqlDirectory = "../sql/";
// const AssignmentsDirectory = "../assignments/";
// const SessionName = "session";
// const MysqlErrNumDuplicateEntry = 1062;

// const UserType = {
//   Student: "student",
//   Teacher: "teacher",
// } as const;
// type UserType = typeof UserType[keyof typeof UserType];

// type UUID = Buffer;

// interface User extends RowDataPacket {
//   id: UUID;
//   code: string;
//   name: string;
//   hashed_password: Buffer;
//   type: UserType;
// }

const app = express();

app.use(morgan("combined"));
app.set("etag", false);

interface InitializeResponse {
  language: string;
}

app.post("/initialize", async (_req, res) => {
  const dbForInit = await mysql.createConnection(getDbInfo(true));
  try {
    const files = ["1_schema.sql", "2_init.sql"];
    for (const file of files) {
      const data = await readFile(SqlDirectory + file);
      dbForInit.query(data.toString());
    }
  } catch (err) {
    return res.status(500).send();
  } finally {
    await dbForInit.end();
  }

  // TODO rm & mkdir

  const response: InitializeResponse = { language: "nodejs" };
  return res.status(200).json(response);
});

app.listen(parseInt(process.env["PORT"] ?? "7000", 10));
