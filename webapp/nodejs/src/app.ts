import { readFile } from "fs/promises";

import bcrypt from "bcrypt";
import session from "cookie-session";
import express from "express";
import morgan from "morgan";
import mysql, { RowDataPacket } from "mysql2/promise";

import { getDbInfo } from "./db";

const SqlDirectory = "../sql/";
// const AssignmentsDirectory = "../assignments/";
const SessionName = "isucholar_nodejs";
// const MysqlErrNumDuplicateEntry = 1062;

const UserType = {
  Student: "student",
  Teacher: "teacher",
} as const;
type UserType = typeof UserType[keyof typeof UserType];

interface User extends RowDataPacket {
  id: string;
  code: string;
  name: string;
  hashed_password: Buffer;
  type: UserType;
}

const pool = mysql.createPool(getDbInfo(false));

const app = express();

app.use(express.json());
app.use(
  session({
    secret: "trapnomura",
    name: SessionName,
  })
);
app.use(morgan("combined"));
app.set("etag", false);

const api = express.Router();
const usersApi = express.Router();
app.use("/api", api);
api.use(isLoggedIn);
api.use("/users", usersApi);

interface InitializeResponse {
  language: string;
}

// Initialize 初期化エンドポイント
app.post("/initialize", async (_, res) => {
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

// IsLoggedIn ログイン確認用middleware
async function isLoggedIn(
  req: express.Request,
  res: express.Response,
  next: express.NextFunction
) {
  if (!req.session) {
    return res.status(500).send();
  }
  if (req.session.isNew) {
    return res.status(401).send("You are not logged in.");
  }
  if (!("userID" in req.session)) {
    return res.status(401).send("You are not logged in.");
  }
  next();
}

interface SessionUserInfo {
  userId: string;
  userName: string;
  isAdmin: boolean;
}

function getUserInfo(
  session?: CookieSessionInterfaces.CookieSessionObject | null
): SessionUserInfo {
  if (!session) {
    throw new Error();
  }
  if (!("userID" in session)) {
    throw new Error("failed to get userID from session");
  }
  if (!("userName" in session)) {
    throw new Error("failed to get userName from session");
  }
  if (!("isAdmin" in session)) {
    throw new Error("failed to get isAdmin from session");
  }
  return {
    userId: session["userID"],
    userName: session["userName"],
    isAdmin: session["isAdmin"],
  };
}

interface LoginRequest {
  code: string;
  password: string;
}

function isValidLoginRequest(body: LoginRequest): body is LoginRequest {
  return (
    typeof body === "object" &&
    typeof body.code === "string" &&
    typeof body.password === "string"
  );
}

// Login ログイン
app.post(
  "/login",
  async (
    req: express.Request<Record<string, never>, unknown, LoginRequest>,
    res
  ) => {
    const request = req.body;
    if (!isValidLoginRequest(request)) {
      return res.status(400).send();
    }

    const db = await pool.getConnection();
    try {
      const [[user]] = await db.query<User[]>(
        "SELECT * FROM `users` WHERE `code` = ?",
        [request.code]
      );
      if (!user) {
        return res.status(401).send("Code or Password is wrong.");
      }

      if (
        !(await bcrypt.compare(
          request.password,
          user.hashed_password.toString()
        ))
      ) {
        return res.status(401).send("Code or Password is wrong.");
      }

      if (req.session && req.session["userID"] === user.id) {
        return res.status(400).send("You are already logged in.");
      }

      req.session = {
        userID: user.id,
        userName: user.name,
        isAdmin: user.type === UserType.Teacher,
      };
      req.sessionOptions.maxAge = 3600 * 1000;

      return res.status(200).send();
    } catch (err) {
      console.error(err);
      return res.status(500).send();
    } finally {
      db.release();
    }
  }
);

app.post("/logout", (req, res) => {
  req.session = null;
  return res.status(200).send();
});

interface GetMeResponse {
  code: string;
  name: string;
  is_admin: boolean;
}

usersApi.get("/me", async (req, res) => {
  let userInfo: SessionUserInfo;
  try {
    userInfo = getUserInfo(req.session);
  } catch (err) {
    console.error(err);
    return res.status(500).send();
  }

  const db = await pool.getConnection();
  try {
    const [[{ code }]] = await db.query<({ code: string } & RowDataPacket)[]>(
      "SELECT `code` FROM `users` WHERE `id` = ?",
      [userInfo.userId]
    );
    if (!code) {
      throw new Error();
    }

    const response: GetMeResponse = {
      code,
      name: userInfo.userName,
      is_admin: userInfo.isAdmin,
    };
    return res.status(200).json(response);
  } catch (err) {
    console.log(err);
    return res.status(500).send();
  } finally {
    db.release();
  }
});

app.listen(parseInt(process.env["PORT"] ?? "7000", 10));
