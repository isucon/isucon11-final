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

const UserType = {
  Student: "student",
  Teacher: "teacher",
} as const;
type UserType = typeof UserType[keyof typeof UserType];

interface User extends RowDataPacket {
  id: string;
  code: string;
  name: string;
  hashedPassword: Buffer;
  type: UserType;
}

const CourseType = {
  LiberalArts: "liberal-arts",
  MajorSubjects: "major-subjects",
} as const;
type CourseType = typeof CourseType[keyof typeof CourseType];

const DayOfWeek = {
  Sunday: "sunday",
  Monday: "monday",
  Tuesday: "tuesday",
  Wednesday: "wednesday",
  Thursday: "thursday",
  Friday: "friday",
  Saturday: "saturday",
} as const;
type DayOfWeek = typeof DayOfWeek[keyof typeof DayOfWeek];

const CourseStatus = {
  StatusRegistration: "registration",
  StatusInProgress: "in-progress",
  StatusClosed: "closed",
} as const;
type CourseStatus = typeof CourseStatus[keyof typeof CourseStatus];

interface Course extends RowDataPacket {
  id: string;
  code: string;
  type: CourseType;
  name: string;
  description: string;
  credit: number;
  period: number;
  dayOfWeek: DayOfWeek;
  teacherId: string;
  keywords: string;
  status: CourseStatus;
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
          user.hashedPassword.toString()
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

interface GetRegisteredCourseResponseContent {
  id: string;
  name: string;
  teacher: string;
  period: number;
  day_of_week: DayOfWeek;
}

// GetRegisteredCourses 履修中の科目一覧取得
usersApi.get("/me/courses", async (req, res) => {
  let userInfo: SessionUserInfo;
  try {
    userInfo = getUserInfo(req.session);
  } catch (err) {
    console.error(err);
    return res.status(500).send();
  }

  const db = await pool.getConnection();
  try {
    const query =
      "SELECT `courses`.*" +
      " FROM `courses`" +
      " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`" +
      " WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?";
    const [courses] = await db.query<Course[]>(query, [
      CourseStatus.StatusClosed,
      userInfo.userId,
    ]);

    // 履修科目が0件の時は空配列を返却
    const response: GetRegisteredCourseResponseContent[] = [];
    for (const course of courses) {
      const [[teacher]] = await db.query<User[]>(
        "SELECT * FROM `users` WHERE `id` = ?",
        [course.teacherId]
      );
      if (!teacher) {
        throw new Error();
      }
      response.push({
        id: course.id,
        name: course.name,
        teacher: teacher.name,
        period: course.period,
        day_of_week: course.dayOfWeek,
      });
    }

    return res.status(200).json(response);
  } catch (err) {
    console.error(err);
    return res.status(500).send();
  } finally {
    db.release();
  }
});

interface RegisterCourseRequestContent {
  id: string;
}

function isValidRegisterCourseRequestContent(
  body: RegisterCourseRequestContent[]
): body is RegisterCourseRequestContent[] {
  return (
    Array.isArray(body) &&
    body.every((data) => {
      return typeof data === "object" && typeof data.id === "string";
    })
  );
}

interface RegisterCoursesErrorResponse {
  course_not_found: string[];
  not_registrable_status: string[];
  schedule_conflict: string[];
}

usersApi.put(
  "/me/courses",
  async (
    req: express.Request<
      Record<string, never>,
      unknown,
      RegisterCourseRequestContent[]
    >,
    res
  ) => {
    let userInfo: SessionUserInfo;
    try {
      userInfo = getUserInfo(req.session);
    } catch (err) {
      console.error(err);
      return res.status(500).send();
    }

    const request = req.body;
    if (!isValidRegisterCourseRequestContent(request)) {
      return res.status(400).send("Invalid format.");
    }
    request.sort((a, b) => {
      if (a.id < b.id) {
        return -1;
      }
      if (a.id > b.id) {
        return 1;
      }
      return 0;
    });

    const db = await pool.getConnection();
    try {
      await db.beginTransaction();

      const errors: RegisterCoursesErrorResponse = {
        course_not_found: [],
        not_registrable_status: [],
        schedule_conflict: [],
      };
      const newlyAdded: Course[] = [];
      for (const courseReq of request) {
        const [[course]] = await db.query<Course[]>(
          "SELECT * FROM `courses` WHERE `id` = ? FOR SHARE",
          [courseReq.id]
        );
        if (!course) {
          errors.course_not_found.push(courseReq.id);
          continue;
        }

        if (course.status !== CourseStatus.StatusRegistration) {
          errors.not_registrable_status.push(course.id);
          continue;
        }

        // MEMO: すでに履修登録済みの科目は無視する
        const [[{ cnt }]] = await db.query<({ cnt: number } & RowDataPacket)[]>(
          "SELECT COUNT(*) AS `cnt` FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?",
          [course.id, userInfo.userId]
        );
        if (cnt > 0) {
          continue;
        }

        newlyAdded.push(course);
      }

      // MEMO: スケジュールの重複バリデーション
      const query =
        "SELECT `courses`.*" +
        " FROM `courses`" +
        " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`" +
        " WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?";
      const [alreadyRegistered] = await db.query<Course[]>(query, [
        CourseStatus.StatusClosed,
        userInfo.userId,
      ]);

      alreadyRegistered.push(...newlyAdded);
      for (const course1 of newlyAdded) {
        for (const course2 of alreadyRegistered) {
          if (
            course1.id !== course2.id &&
            course1.period === course2.period &&
            course1.dayOfWeek === course2.dayOfWeek
          ) {
            errors.schedule_conflict.push(course1.id);
            break;
          }
        }
      }

      if (
        errors.course_not_found.length > 0 ||
        errors.not_registrable_status.length > 0 ||
        errors.schedule_conflict.length > 0
      ) {
        await db.rollback();
        return res.status(400).json(errors);
      }

      for (const course of newlyAdded) {
        await db.query(
          "INSERT INTO `registrations` (`course_id`, `user_id`) VALUES (?, ?)",
          [course.id, userInfo.userId]
        );
      }

      await db.commit();

      return res.status(200).send();
    } catch (err) {
      await db.rollback();
      return res.status(500).send();
    } finally {
      db.release();
    }
  }
);

app.listen(parseInt(process.env["PORT"] ?? "7000", 10));
