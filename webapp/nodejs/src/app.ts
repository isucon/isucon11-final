import { spawn as _spawn } from "child_process";
import { readFile, writeFile } from "fs/promises";
import { join } from "path";
import { cwd } from "process";
import { URL } from "url";

import bcrypt from "bcrypt";
import session from "cookie-session";
import express from "express";
import morgan from "morgan";
import multer from "multer";
import mysql, { ResultSetHeader, RowDataPacket } from "mysql2/promise";

import {
  averageFloat,
  averageInt,
  max,
  min,
  newUlid,
  tScoreFloat,
  tScoreInt,
} from "./util";

const SqlDirectory = "../sql/";
const AssignmentsDirectory = "../assignments/";
const InitDataDirectory = "../data/";
const SessionName = "isucholar_nodejs";
const MysqlErrNumDuplicateEntry = 1062;

const dbinfo: mysql.PoolOptions = {
  host: process.env["MYSQL_HOSTNAME"] ?? "127.0.0.1",
  port: parseInt(process.env["MYSQL_PORT"] ?? "3306", 10),
  user: process.env["MYSQL_USER"] ?? "isucon",
  password: process.env["MYSQL_PASS"] || "isucon",
  database: process.env["MYSQL_DATABASE"] ?? "isucholar",
  timezone: "+00:00",
  decimalNumbers: true,
};
const spawn = (command: string, ...args: string[]) =>
  new Promise((resolve, reject) => {
    const cmd = _spawn(command, args, { stdio: "ignore" });
    cmd.on("error", (err) => reject(err));
    cmd.on("close", (code) => {
      if (code === 0) {
        resolve(code);
      } else {
        reject(new Error(`Unexpected exit code: ${code} on ${command}`));
      }
    });
  });
const pool = mysql.createPool(dbinfo);
const upload = multer();

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
const coursesApi = express.Router();
const announcementsApi = express.Router();
app.use("/api", api);
api.use(isLoggedIn);
api.use("/users", usersApi);
api.use("/courses", coursesApi);
api.use("/announcements", announcementsApi);

interface InitializeResponse {
  language: string;
}

// POST /initialize 初期化エンドポイント
app.post("/initialize", async (_, res) => {
  const db = await mysql.createConnection({
    ...dbinfo,
    multipleStatements: true,
  });
  try {
    const files = ["1_schema.sql", "2_init.sql", "3_sample.sql"];
    for (const file of files) {
      const data = await readFile(SqlDirectory + file);
      await db.query(data.toString());
    }
  } catch (err) {
    console.error(err);
    return res.status(500).send();
  } finally {
    await db.end();
  }

  try {
    await spawn("rm", "-rf", AssignmentsDirectory);
    await spawn("cp", "-r", InitDataDirectory, AssignmentsDirectory);
  } catch (err) {
    console.error(err);
    return res.status(500).send();
  }

  const response: InitializeResponse = { language: "nodejs" };
  return res.status(200).json(response);
});

// ログイン確認用middleware
async function isLoggedIn(
  req: express.Request,
  res: express.Response,
  next: express.NextFunction
) {
  if (!req.session) {
    return res.status(500).send();
  }
  if (req.session.isNew) {
    return res.status(401).type("text").send("You are not logged in.");
  }
  if (!("userID" in req.session)) {
    return res.status(401).type("text").send("You are not logged in.");
  }
  next();
}

// admin確認用middleware
async function isAdmin(
  req: express.Request,
  res: express.Response,
  next: express.NextFunction
) {
  if (!req.session) {
    return res.status(500).send();
  }
  if (!("isAdmin" in req.session)) {
    console.error("failed to get isAdmin from session");
    return res.status(500).send();
  }
  if (!req.session["isAdmin"]) {
    return res.status(403).type("text").send("You are not admin user.");
  }
  next();
}

function getUserInfo(
  session?: CookieSessionInterfaces.CookieSessionObject | null
): [string, string, boolean] {
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
  return [session["userID"], session["userName"], session["isAdmin"]];
}

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

const CourseType = {
  LiberalArts: "liberal-arts",
  MajorSubjects: "major-subjects",
} as const;
type CourseType = typeof CourseType[keyof typeof CourseType];

const DayOfWeek = {
  Monday: "monday",
  Tuesday: "tuesday",
  Wednesday: "wednesday",
  Thursday: "thursday",
  Friday: "friday",
} as const;
type DayOfWeek = typeof DayOfWeek[keyof typeof DayOfWeek];

const DaysOfWeek = [
  DayOfWeek.Monday,
  DayOfWeek.Tuesday,
  DayOfWeek.Wednesday,
  DayOfWeek.Thursday,
  DayOfWeek.Friday,
];

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
  day_of_week: DayOfWeek;
  teacher_id: string;
  keywords: string;
  status: CourseStatus;
}

// ---------- Public API ----------

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

// POST /login ログイン
app.post("/login", async (req, res) => {
  const request = req.body;
  if (!isValidLoginRequest(request)) {
    return res.status(400).type("text").send("Invalid format.");
  }

  const db = await pool.getConnection();
  try {
    const [[user]] = await db.query<User[]>(
      "SELECT * FROM `users` WHERE `code` = ?",
      [request.code]
    );
    if (!user) {
      return res.status(401).type("text").send("Code or Password is wrong.");
    }

    if (
      !(await bcrypt.compare(request.password, user.hashed_password.toString()))
    ) {
      return res.status(401).type("text").send("Code or Password is wrong.");
    }

    if (req.session && req.session["userID"] === user.id) {
      return res.status(400).type("text").send("You are already logged in.");
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
});

// POST /logout ログアウト
app.post("/logout", (req, res) => {
  req.session = null;
  return res.status(200).send();
});

// ---------- Users API ----------

interface GetMeResponse {
  code: string;
  name: string;
  is_admin: boolean;
}

// GET /api/users/me 自身の情報を取得
usersApi.get("/me", async (req, res) => {
  let userId: string;
  let userName: string;
  let isAdmin: boolean;
  try {
    [userId, userName, isAdmin] = getUserInfo(req.session);
  } catch (err) {
    console.error(err);
    return res.status(500).send();
  }

  const db = await pool.getConnection();
  try {
    const [[{ code }]] = await db.query<({ code: string } & RowDataPacket)[]>(
      "SELECT `code` FROM `users` WHERE `id` = ?",
      [userId]
    );
    if (!code) {
      throw new Error();
    }

    const response: GetMeResponse = {
      code,
      name: userName,
      is_admin: isAdmin,
    };
    return res.status(200).json(response);
  } catch (err) {
    console.error(err);
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

// GET /api/users/me/courses 履修中の科目一覧取得
usersApi.get("/me/courses", async (req, res) => {
  let userId: string;
  try {
    [userId] = getUserInfo(req.session);
  } catch (err) {
    console.error(err);
    return res.status(500).send();
  }

  const db = await pool.getConnection();
  try {
    await db.beginTransaction();

    const [courses] = await db.query<Course[]>(
      "SELECT `courses`.*" +
        " FROM `courses`" +
        " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`" +
        " WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?",
      [CourseStatus.StatusClosed, userId]
    );

    // 履修科目が0件の時は空配列を返却
    const response: GetRegisteredCourseResponseContent[] = [];
    for (const course of courses) {
      const [[teacher]] = await db.query<User[]>(
        "SELECT * FROM `users` WHERE `id` = ?",
        [course.teacher_id]
      );
      if (!teacher) {
        throw new Error();
      }
      response.push({
        id: course.id,
        name: course.name,
        teacher: teacher.name,
        period: course.period,
        day_of_week: course.day_of_week,
      });
    }

    await db.commit();

    return res.status(200).json(response);
  } catch (err) {
    console.error(err);
    await db.rollback();
    return res.status(500).send();
  } finally {
    db.release();
  }
});

type RegisterCourseRequest = RegisterCourseRequestContent[];

interface RegisterCourseRequestContent {
  id: string;
}

function isValidRegisterCourseRequest(
  body: RegisterCourseRequest
): body is RegisterCourseRequest {
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

// PUT /api/users/me/courses 履修登録
usersApi.put("/me/courses", async (req, res) => {
  let userId: string;
  try {
    [userId] = getUserInfo(req.session);
  } catch (err) {
    console.error(err);
    return res.status(500).send();
  }

  const request = req.body;
  if (!isValidRegisterCourseRequest(request)) {
    return res.status(400).type("text").send("Invalid format.");
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

      // すでに履修登録済みの科目は無視する
      const [[{ cnt }]] = await db.query<({ cnt: number } & RowDataPacket)[]>(
        "SELECT COUNT(*) AS `cnt` FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?",
        [course.id, userId]
      );
      if (cnt > 0) {
        continue;
      }

      newlyAdded.push(course);
    }

    const [alreadyRegistered] = await db.query<Course[]>(
      "SELECT `courses`.*" +
        " FROM `courses`" +
        " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`" +
        " WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?",
      [CourseStatus.StatusClosed, userId]
    );

    alreadyRegistered.push(...newlyAdded);
    for (const course1 of newlyAdded) {
      for (const course2 of alreadyRegistered) {
        if (
          course1.id !== course2.id &&
          course1.period === course2.period &&
          course1.day_of_week === course2.day_of_week
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
        "INSERT INTO `registrations` (`course_id`, `user_id`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `course_id` = VALUES(`course_id`), `user_id` = VALUES(`user_id`)",
        [course.id, userId]
      );
    }

    await db.commit();

    return res.status(200).send();
  } catch (err) {
    console.error(err);
    await db.rollback();
    return res.status(500).send();
  } finally {
    db.release();
  }
});

interface Class extends RowDataPacket {
  id: string;
  course_id: string;
  part: number;
  title: string;
  description: string;
  submission_closed: number;
}

interface GetGradeResponse {
  summary: Summary;
  courses: CourseResult[];
}

interface Summary {
  credits: number;
  gpa: number;
  gpa_t_score: number; // 偏差値
  gpa_avg: number; // 平均値
  gpa_max: number; // 最大値
  gpa_min: number; // 最小値
}

interface CourseResult {
  name: string;
  code: string;
  total_score: number;
  total_score_t_score: number; // 偏差値
  total_score_avg: number; // 平均値
  total_score_max: number; // 最大値
  total_score_min: number; // 最小値
  class_scores: ClassScore[];
}

interface ClassScore {
  class_id: string;
  title: string;
  part: number;
  score: number | null;
  submitters: number; // 提出した学生数
}

// GET /api/users/me/grades 成績取得
usersApi.get("/me/grades", async (req, res) => {
  let userId: string;
  try {
    [userId] = getUserInfo(req.session);
  } catch (err) {
    console.error(err);
    return res.status(500).send();
  }

  const db = await pool.getConnection();
  try {
    // 履修している科目一覧取得
    const [registeredCourses] = await db.query<Course[]>(
      "SELECT `courses`.*" +
        " FROM `registrations`" +
        " JOIN `courses` ON `registrations`.`course_id` = `courses`.`id`" +
        " WHERE `user_id` = ?",
      [userId]
    );

    // 科目毎の成績計算処理
    const courseResults: CourseResult[] = [];
    let myGPA = 0.0;
    let myCredits = 0;
    for (const course of registeredCourses) {
      // 講義一覧の取得
      const [classes] = await db.query<Class[]>(
        "SELECT *" +
          " FROM `classes`" +
          " WHERE `course_id` = ?" +
          " ORDER BY `part` DESC",
        [course.id]
      );

      // 講義毎の成績計算処理
      const classScores: ClassScore[] = [];
      let myTotalScore = 0;
      for (const cls of classes) {
        const [[{ submissionsCount }]] = await db.query<
          ({ submissionsCount: number } & RowDataPacket)[]
        >(
          "SELECT COUNT(*) AS `submissionsCount` FROM `submissions` WHERE `class_id` = ?",
          [cls.id]
        );

        const [[row]] = await db.query<({ score: number } & RowDataPacket)[]>(
          "SELECT `submissions`.`score` FROM `submissions` WHERE `user_id` = ? AND `class_id` = ?",
          [userId, cls.id]
        );
        if (!row || typeof row.score !== "number") {
          classScores.push({
            class_id: cls.id,
            part: cls.part,
            title: cls.title,
            score: null,
            submitters: submissionsCount,
          });
        } else {
          const myScore = row.score;
          myTotalScore += myScore;
          classScores.push({
            class_id: cls.id,
            part: cls.part,
            title: cls.title,
            score: myScore,
            submitters: submissionsCount,
          });
        }
      }

      // この科目を履修している学生のTotalScore一覧を取得
      const [rows] = await db.query<
        ({ total_score: number } & RowDataPacket)[]
      >(
        "SELECT IFNULL(SUM(`submissions`.`score`), 0) AS `total_score`" +
          " FROM `users`" +
          " JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`" +
          " JOIN `courses` ON `registrations`.`course_id` = `courses`.`id`" +
          " LEFT JOIN `classes` ON `courses`.`id` = `classes`.`course_id`" +
          " LEFT JOIN `submissions` ON `users`.`id` = `submissions`.`user_id` AND `submissions`.`class_id` = `classes`.`id`" +
          " WHERE `courses`.`id` = ?" +
          " GROUP BY `users`.`id`",
        [course.id]
      );
      const totals = rows.map((row) => row.total_score);
      courseResults.push({
        name: course.name,
        code: course.code,
        total_score: myTotalScore,
        total_score_t_score: tScoreInt(myTotalScore, totals),
        total_score_avg: averageInt(totals, 0),
        total_score_max: max(totals, 0),
        total_score_min: min(totals, 0),
        class_scores: classScores,
      });

      // 自分のGPA計算
      if (course.status === CourseStatus.StatusClosed) {
        myGPA += myTotalScore * course.credit;
        myCredits += course.credit;
      }
    }
    if (myCredits > 0) {
      myGPA = myGPA / 100 / myCredits;
    }

    // GPAの統計値
    // 一つでも修了した科目がある学生のGPA一覧
    const [rows] = await db.query<({ gpa: number } & RowDataPacket)[]>(
      "SELECT IFNULL(SUM(`submissions`.`score` * `courses`.`credit`), 0) / 100 / `credits`.`credits` AS `gpa`" +
        " FROM `users`" +
        " JOIN (" +
        "     SELECT `users`.`id` AS `user_id`, SUM(`courses`.`credit`) AS `credits`" +
        "     FROM `users`" +
        "     JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`" +
        "     JOIN `courses` ON `registrations`.`course_id` = `courses`.`id` AND `courses`.`status` = ?" +
        "     GROUP BY `users`.`id`" +
        " ) AS `credits` ON `credits`.`user_id` = `users`.`id`" +
        " JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`" +
        " JOIN `courses` ON `registrations`.`course_id` = `courses`.`id` AND `courses`.`status` = ?" +
        " LEFT JOIN `classes` ON `courses`.`id` = `classes`.`course_id`" +
        " LEFT JOIN `submissions` ON `users`.`id` = `submissions`.`user_id` AND `submissions`.`class_id` = `classes`.`id`" +
        " WHERE `users`.`type` = ?" +
        " GROUP BY `users`.`id`",
      [CourseStatus.StatusClosed, CourseStatus.StatusClosed, UserType.Student]
    );
    const gpas = rows.map((row) => row.gpa);

    const response: GetGradeResponse = {
      summary: {
        credits: myCredits,
        gpa: myGPA,
        gpa_t_score: tScoreFloat(myGPA, gpas),
        gpa_avg: averageFloat(gpas, 0),
        gpa_max: max(gpas, 0),
        gpa_min: min(gpas, 0),
      },
      courses: courseResults,
    };
    return res.status(200).json(response);
  } catch (err) {
    console.error(err);
    return res.status(500).send();
  } finally {
    db.release();
  }
});

// ---------- Courses API ----------

interface SearchCoursesQuery {
  type: string;
  credit: string;
  teacher: string;
  period: string;
  day_of_week: string;
  keywords: string;
  status: string;
  page: string;
}

// GET /api/courses 科目検索
coursesApi.get(
  "",
  async (
    req: express.Request<
      Record<string, never>,
      unknown,
      never,
      SearchCoursesQuery
    >,
    res
  ) => {
    const query =
      "SELECT `courses`.*, `users`.`name` AS `teacher`" +
      " FROM `courses` JOIN `users` ON `courses`.`teacher_id` = `users`.`id`" +
      " WHERE 1=1";
    let condition = "";
    const args = [];

    // 無効な検索条件はエラーを返さず無視して良い

    if (req.query.type) {
      condition += " AND `courses`.`type` = ?";
      args.push(req.query.type);
    }

    if (req.query.credit) {
      const credit = parseInt(req.query.credit, 10);
      if (!isNaN(credit) && credit > 0) {
        condition += " AND `courses`.`credit` = ?";
        args.push(credit);
      }
    }

    if (req.query.teacher) {
      condition += " AND `users`.`name` = ?";
      args.push(req.query.teacher);
    }

    if (req.query.period) {
      const period = parseInt(req.query.period, 10);
      if (!isNaN(period) && period > 0) {
        condition += " AND `courses`.`period` = ?";
        args.push(period);
      }
    }

    if (req.query.day_of_week) {
      condition += " AND `courses`.`day_of_week` = ?";
      args.push(req.query.day_of_week);
    }

    if (req.query.keywords) {
      const arr = req.query.keywords.split(" ");
      let nameCondition = "";
      arr.forEach((keyword) => {
        nameCondition += " AND `courses`.`name` LIKE ?";
        args.push(`%${keyword}%`);
      });
      let keywordsCondition = "";
      arr.forEach((keyword) => {
        keywordsCondition += " AND `courses`.`keywords` LIKE ?";
        args.push(`%${keyword}%`);
      });
      condition += ` AND ((1=1${nameCondition}) OR (1=1${keywordsCondition}))`;
    }

    if (req.query.status) {
      condition += " AND `courses`.`status` = ?";
      args.push(req.query.status);
    }

    condition += " ORDER BY `courses`.`code`";

    let page: number;
    if (!req.query.page) {
      page = 1;
    } else {
      page = parseInt(req.query.page, 10);
      if (isNaN(page) || page <= 0) {
        return res.status(400).type("text").send("Invalid page.");
      }
    }
    const limit = 20;
    const offset = limit * (page - 1);

    // limitより多く上限を設定し、実際にlimitより多くレコードが取得できた場合は次のページが存在する
    condition += " LIMIT ? OFFSET ?";
    args.push(limit + 1, offset);

    const db = await pool.getConnection();
    try {
      // 結果が0件の時は空配列を返却
      let [response] = await db.query<GetCourseDetailResponse[]>(
        query + condition,
        args
      );

      const links = [];
      const linkUrl = new URL(
        req.originalUrl,
        `${req.protocol}://${req.hostname}`
      );

      const q = linkUrl.searchParams;
      if (page > 1) {
        q.set("page", `${page - 1}`);
        links.push(`<${linkUrl.pathname}?${q}>; rel="prev"`);
      }
      if (response.length > limit) {
        q.set("page", `${page + 1}`);
        links.push(`<${linkUrl.pathname}?${q}>; rel="next"`);
      }
      if (links.length > 0) {
        res.append("Link", links.join(","));
      }

      if (response.length === limit + 1) {
        response = response.slice(0, response.length - 1);
      }

      return res.status(200).json(
        response.map((r) => ({
          ...r,
          teacher_id: undefined,
        }))
      );
    } catch (err) {
      console.error(err);
      return res.status(500).send();
    } finally {
      db.release();
    }
  }
);

interface AddCourseRequest {
  code: string;
  type: CourseType;
  name: string;
  description: string;
  credit: number;
  period: number;
  day_of_week: DayOfWeek;
  keywords: string;
}

function isValidAddCourseRequest(
  body: AddCourseRequest
): body is AddCourseRequest {
  return (
    typeof body === "object" &&
    typeof body.code === "string" &&
    typeof body.type === "string" &&
    typeof body.name === "string" &&
    typeof body.description === "string" &&
    typeof body.credit === "number" &&
    typeof body.period === "number" &&
    typeof body.day_of_week === "string" &&
    typeof body.keywords === "string"
  );
}

interface AddCourseResponse {
  id: string;
}

// POST /api/courses 新規科目登録
coursesApi.post("", isAdmin, async (req, res) => {
  let userId: string;
  try {
    [userId] = getUserInfo(req.session);
  } catch (err) {
    console.error(err);
    return res.status(500).send();
  }

  const request = req.body;
  if (!isValidAddCourseRequest(request)) {
    return res.status(400).type("text").send("Invalid format.");
  }

  if (
    request.type !== CourseType.LiberalArts &&
    request.type !== CourseType.MajorSubjects
  ) {
    return res.status(400).type("text").send("Invalid course type.");
  }
  if (!DaysOfWeek.includes(request.day_of_week)) {
    return res.status(400).type("text").send("Invalid day of week.");
  }

  const db = await pool.getConnection();
  try {
    const courseId = newUlid();
    try {
      await db.query(
        "INSERT INTO `courses` (`id`, `code`, `type`, `name`, `description`, `credit`, `period`, `day_of_week`, `teacher_id`, `keywords`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
        [
          courseId,
          request.code,
          request.type,
          request.name,
          request.description,
          request.credit,
          request.period,
          request.day_of_week,
          userId,
          request.keywords,
        ]
      );
    } catch (err) {
      if (
        err &&
        typeof err === "object" &&
        (err as { errno: number }).errno === MysqlErrNumDuplicateEntry
      ) {
        const [[course]] = await db.query<Course[]>(
          "SELECT * FROM `courses` WHERE `code` = ?",
          [request.code]
        );
        if (
          request.type !== course.type ||
          request.name !== course.name ||
          request.description !== course.description ||
          request.credit !== course.credit ||
          request.period !== course.period ||
          request.day_of_week !== course.day_of_week ||
          request.keywords !== course.keywords
        ) {
          return res
            .status(409)
            .type("text")
            .send("A course with the same code already exists.");
        }
        const response: AddCourseResponse = { id: course.id };
        return res.status(201).json(response);
      }
      console.error(err);
      return res.status(500).send();
    }

    const response: AddCourseResponse = { id: courseId };
    return res.status(201).json(response);
  } catch (err) {
    console.error(err);
    return res.status(500).send();
  } finally {
    db.release();
  }
});

interface GetCourseDetailResponse extends RowDataPacket {
  id: string;
  code: string;
  type: string;
  name: string;
  description: string;
  credit: number;
  period: number;
  day_of_week: string;
  teacher_id: string;
  keywords: string;
  status: CourseStatus;
  teacher: string;
}

// GET /api/courses/:courseId 科目詳細の取得
coursesApi.get(
  "/:courseId",
  async (req: express.Request<{ courseId: string }>, res) => {
    const courseId = req.params.courseId;

    const db = await pool.getConnection();
    try {
      const [[response]] = await db.query<GetCourseDetailResponse[]>(
        "SELECT `courses`.*, `users`.`name` AS `teacher`" +
          " FROM `courses`" +
          " JOIN `users` ON `courses`.`teacher_id` = `users`.`id`" +
          " WHERE `courses`.`id` = ?",
        [courseId]
      );
      if (!response) {
        return res.status(404).type("text").send("No such course.");
      }

      return res.status(200).json({ ...response, teacher_id: undefined });
    } catch (err) {
      console.error(err);
      return res.status(500).send();
    } finally {
      db.release();
    }
  }
);

interface SetCourseStatusRequest {
  status: CourseStatus;
}

function isValidSetCourseStatusRequest(
  body: SetCourseStatusRequest
): body is SetCourseStatusRequest {
  return typeof body === "object" && typeof body.status === "string";
}

// PUT /api/courses/:courseId/status 科目のステータスを変更
coursesApi.put(
  "/:courseId/status",
  isAdmin,
  async (req: express.Request<{ courseId: string }>, res) => {
    const courseId = req.params.courseId;

    const request = req.body;
    if (!isValidSetCourseStatusRequest(request)) {
      return res.status(400).type("text").send("Invalid format.");
    }

    const db = await pool.getConnection();
    try {
      await db.beginTransaction();

      const [[{ cnt }]] = await db.query<({ cnt: number } & RowDataPacket)[]>(
        "SELECT COUNT(*) AS `cnt` FROM `courses` WHERE `id` = ? FOR UPDATE",
        [courseId]
      );
      if (cnt === 0) {
        await db.rollback();
        return res.status(404).type("text").send("No such course.");
      }

      await db.query<ResultSetHeader>(
        "UPDATE `courses` SET `status` = ? WHERE `id` = ?",
        [request.status, courseId]
      );

      await db.commit();

      return res.status(200).send();
    } catch (err) {
      console.error(err);
      await db.rollback();
      return res.status(500).send();
    } finally {
      db.release();
    }
  }
);

interface ClassWithSubmitted extends Class {
  submitted: number;
}

interface GetClassResponse {
  id: string;
  part: number;
  title: string;
  description: string;
  submission_closed: boolean;
  submitted: boolean;
}

// GET /api/courses/:courseId/classes 科目に紐づく講義一覧の取得
coursesApi.get(
  "/:courseId/classes",
  async (req: express.Request<{ courseId: string }>, res) => {
    let userId: string;
    try {
      [userId] = getUserInfo(req.session);
    } catch (err) {
      console.error(err);
      return res.status(500).send();
    }

    const courseId = req.params.courseId;

    const db = await pool.getConnection();
    try {
      await db.beginTransaction();

      const [[{ cnt }]] = await db.query<({ cnt: number } & RowDataPacket)[]>(
        "SELECT COUNT(*) AS `cnt` FROM `courses` WHERE `id` = ?",
        [courseId]
      );
      if (cnt === 0) {
        await db.rollback();
        return res.status(404).type("text").send("No such course.");
      }

      const [classes] = await db.query<ClassWithSubmitted[]>(
        "SELECT `classes`.*, `submissions`.`user_id` IS NOT NULL AS `submitted`" +
          " FROM `classes`" +
          " LEFT JOIN `submissions` ON `classes`.`id` = `submissions`.`class_id` AND `submissions`.`user_id` = ?" +
          " WHERE `classes`.`course_id` = ?" +
          " ORDER BY `classes`.`part`",
        [userId, courseId]
      );

      await db.commit();

      // 結果が0件の時は空配列を返却
      const response: GetClassResponse[] = classes.map((cls) => {
        return {
          id: cls.id,
          part: cls.part,
          title: cls.title,
          description: cls.description,
          submission_closed: !!cls.submission_closed,
          submitted: !!cls.submitted,
        };
      });

      return res.status(200).json(response);
    } catch (err) {
      console.error(err);
      await db.rollback();
      return res.status(500).send();
    } finally {
      db.release();
    }
  }
);

interface AddClassRequest {
  part: number;
  title: string;
  description: string;
}

function isValidAddClassRequest(
  body: AddClassRequest
): body is AddClassRequest {
  return (
    typeof body === "object" &&
    typeof body.part === "number" &&
    typeof body.title === "string" &&
    typeof body.description === "string"
  );
}

interface AddClassResponse {
  class_id: string;
}

// POST /api/courses/:courseId/classes 新規講義(&課題)追加
coursesApi.post(
  "/:courseId/classes",
  isAdmin,
  async (req: express.Request<{ courseId: string }>, res) => {
    const courseId = req.params.courseId;

    const request = req.body;
    if (!isValidAddClassRequest(request)) {
      return res.status(400).type("text").send("Invalid format.");
    }

    const db = await pool.getConnection();
    try {
      await db.beginTransaction();

      const [[course]] = await db.query<Course[]>(
        "SELECT * FROM `courses` WHERE `id` = ? FOR SHARE",
        [courseId]
      );
      if (!course) {
        await db.rollback();
        return res.status(404).type("text").send("No such course.");
      }
      if (course.status !== CourseStatus.StatusInProgress) {
        await db.rollback();
        return res
          .status(400)
          .type("text")
          .send("This course is not in-progress.");
      }

      const classId = newUlid();
      try {
        await db.query(
          "INSERT INTO `classes` (`id`, `course_id`, `part`, `title`, `description`) VALUES (?, ?, ?, ?, ?)",
          [classId, courseId, request.part, request.title, request.description]
        );
      } catch (err) {
        await db.rollback();
        if (
          err &&
          typeof err === "object" &&
          (err as { errno: number }).errno === MysqlErrNumDuplicateEntry
        ) {
          const [[cls]] = await db.query<Class[]>(
            "SELECT * FROM `classes` WHERE `course_id` = ? AND `part` = ?",
            [courseId, request.part]
          );
          if (
            request.title !== cls.title ||
            request.description !== cls.description
          ) {
            return res
              .status(409)
              .type("text")
              .send("A class with the same part already exists.");
          }
          const response: AddClassResponse = { class_id: cls.id };
          return res.status(201).json(response);
        }
        console.error(err);
        return res.status(500).send();
      }

      await db.commit();

      const response: AddClassResponse = { class_id: classId };
      return res.status(201).json(response);
    } catch (err) {
      console.error(err);
      await db.rollback();
      return res.status(500).send();
    } finally {
      db.release();
    }
  }
);

// POST /api/courses/:courseId/classes/:classId/assignments 課題の提出
coursesApi.post(
  "/:courseId/classes/:classId/assignments",
  upload.single("file"),
  async (req: express.Request<{ courseId: string; classId: string }>, res) => {
    let userId: string;
    try {
      [userId] = getUserInfo(req.session);
    } catch (err) {
      console.error(err);
      return res.status(500).send();
    }

    const courseId = req.params.courseId;
    const classId = req.params.classId;

    const db = await pool.getConnection();
    try {
      await db.beginTransaction();

      const [[course]] = await db.query<Course[]>(
        "SELECT * FROM `courses` WHERE `id` = ? FOR SHARE",
        [courseId]
      );
      if (!course) {
        await db.rollback();
        return res.status(404).type("text").send("No such course.");
      }
      if (course.status !== CourseStatus.StatusInProgress) {
        await db.rollback();
        return res
          .status(400)
          .type("text")
          .send("This course is not in-progress.");
      }

      const [[{ registrationCount }]] = await db.query<
        ({ registrationCount: number } & RowDataPacket)[]
      >(
        "SELECT COUNT(*) AS `registrationCount` FROM `registrations` WHERE `user_id` = ? AND `course_id` = ?",
        [userId, courseId]
      );
      if (registrationCount === 0) {
        await db.rollback();
        return res
          .status(400)
          .type("text")
          .send("You have not taken this course.");
      }

      const [[row]] = await db.query<
        ({ submission_closed: number } & RowDataPacket)[]
      >("SELECT `submission_closed` FROM `classes` WHERE `id` = ? FOR SHARE", [
        classId,
      ]);
      if (!row) {
        await db.rollback();
        return res.status(404).type("text").send("No such class.");
      }
      if (row.submission_closed) {
        await db.rollback();
        return res
          .status(400)
          .type("text")
          .send("Submission has been closed for this class.");
      }

      if (!req.file) {
        await db.rollback();
        return res.status(400).type("text").send("Invalid file.");
      }
      await db.query(
        "INSERT INTO `submissions` (`user_id`, `class_id`, `file_name`) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE `file_name` = VALUES(`file_name`)",
        [userId, classId, req.file.originalname]
      );

      await writeFile(
        AssignmentsDirectory + classId + "-" + userId + ".pdf",
        req.file.buffer
      );

      await db.commit();

      return res.status(204).send();
    } catch (err) {
      await db.rollback();
      console.error(err);
      return res.status(500).send();
    } finally {
      db.release();
    }
  }
);

type RegisterScoresRequest = Score[];

interface Score {
  user_code: string;
  score: number;
}

function isValidRegisterScoresRequest(
  body: RegisterScoresRequest
): body is RegisterScoresRequest {
  return (
    Array.isArray(body) &&
    body.every((data) => {
      return (
        typeof data === "object" &&
        typeof data.user_code === "string" &&
        typeof data.score === "number"
      );
    })
  );
}

// PUT /api/courses/:courseId/classes/:classId/assignments/scores 採点結果登録
coursesApi.put(
  "/:courseId/classes/:classId/assignments/scores",
  isAdmin,
  async (req: express.Request<{ courseId: string; classId: string }>, res) => {
    const classId = req.params.classId;

    const db = await pool.getConnection();
    try {
      await db.beginTransaction();

      const [[row]] = await db.query<
        ({ submission_closed: number } & RowDataPacket)[]
      >("SELECT `submission_closed` FROM `classes` WHERE `id` = ? FOR SHARE", [
        classId,
      ]);
      if (!row) {
        await db.rollback();
        return res.status(404).type("text").send("No such class.");
      }
      if (!row.submission_closed) {
        await db.rollback();
        return res
          .status(400)
          .type("text")
          .send("This assignment is not closed yet.");
      }

      const request = req.body;
      if (!isValidRegisterScoresRequest(request)) {
        await db.rollback();
        return res.status(400).type("text").send("Invalid format.");
      }

      for (const score of request) {
        await db.query(
          "UPDATE `submissions` JOIN `users` ON `users`.`id` = `submissions`.`user_id` SET `score` = ? WHERE `users`.`code` = ? AND `class_id` = ?",
          [score.score, score.user_code, classId]
        );
      }

      await db.commit();

      return res.status(204).send();
    } catch (err) {
      console.error(err);
      await db.rollback();
      return res.status(500).send();
    } finally {
      db.release();
    }
  }
);

interface Submission extends RowDataPacket {
  user_id: string;
  user_code: string;
  file_name: string;
}

// GET /api/courses/:courseId/classes/:classId/assignments/export 提出済みの課題ファイルをzip形式で一括ダウンロード
coursesApi.get(
  "/:courseId/classes/:classId/assignments/export",
  isAdmin,
  async (req: express.Request<{ courseId: string; classId: string }>, res) => {
    const classId = req.params.classId;

    const db = await pool.getConnection();
    try {
      await db.beginTransaction();

      const [[{ classCount }]] = await db.query<
        ({ classCount: number } & RowDataPacket)[]
      >(
        "SELECT COUNT(*) AS `classCount` FROM `classes` WHERE `id` = ? FOR UPDATE",
        [classId]
      );
      if (classCount === 0) {
        await db.rollback();
        return res.status(404).type("text").send("No such class.");
      }

      const [submissions] = await db.query<Submission[]>(
        "SELECT `submissions`.`user_id`, `submissions`.`file_name`, `users`.`code` AS `user_code`" +
          " FROM `submissions`" +
          " JOIN `users` ON `users`.`id` = `submissions`.`user_id`" +
          " WHERE `class_id` = ?",
        [classId]
      );

      const zipFilePath = AssignmentsDirectory + classId + ".zip";
      await createSubmissionsZip(zipFilePath, classId, submissions);

      await db.query(
        "UPDATE `classes` SET `submission_closed` = true WHERE `id` = ?",
        [classId]
      );

      await db.commit();

      return res.sendFile(join(cwd(), zipFilePath));
    } catch (err) {
      console.error(err);
      await db.rollback();
      return res.status(500).send();
    } finally {
      db.release();
    }
  }
);

async function createSubmissionsZip(
  zipFilePath: string,
  classId: string,
  submissions: Submission[]
) {
  const tmpDir = AssignmentsDirectory + classId + "/";
  await spawn("rm", "-rf", tmpDir);
  await spawn("mkdir", tmpDir);

  // ファイル名を指定の形式に変更
  for (const submission of submissions) {
    await spawn(
      "cp",
      AssignmentsDirectory + classId + "-" + submission.user_id + ".pdf",
      tmpDir + submission.user_code + "-" + submission.file_name
    );
  }

  // -i 'tmpDir/*': 空zipを許す
  await spawn("zip", "-j", "-r", zipFilePath, tmpDir, "-i", tmpDir + "*");
}

// ---------- Announcement API ----------

interface AnnouncementWithoutDetail {
  id: string;
  course_id: string;
  course_name: string;
  title: string;
  unread: boolean;
}

interface AnnouncementWithoutDetailRow
  extends Omit<AnnouncementWithoutDetail, "unread">,
    RowDataPacket {
  unread: number;
}

interface GetAnnouncementListQuery {
  course_id: string;
  page: string;
}

interface GetAnnouncementsResponse {
  unread_count: number;
  announcements: AnnouncementWithoutDetail[];
}

// GET /api/announcements お知らせ一覧取得
announcementsApi.get(
  "",
  async (
    req: express.Request<
      Record<string, never>,
      unknown,
      never,
      GetAnnouncementListQuery
    >,
    res
  ) => {
    let userId: string;
    try {
      [userId] = getUserInfo(req.session);
    } catch (err) {
      console.error(err);
      return res.status(500).send();
    }

    const args = [];
    let query =
      "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name` AS `course_name`, `announcements`.`title`, NOT `unread_announcements`.`is_deleted` AS `unread`" +
      " FROM `announcements`" +
      " JOIN `courses` ON `announcements`.`course_id` = `courses`.`id`" +
      " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`" +
      " JOIN `unread_announcements` ON `announcements`.`id` = `unread_announcements`.`announcement_id`" +
      " WHERE 1=1";

    if (req.query.course_id) {
      query += " AND `announcements`.`course_id` = ?";
      args.push(req.query.course_id);
    }

    query +=
      " AND `unread_announcements`.`user_id` = ?" +
      " AND `registrations`.`user_id` = ?" +
      " ORDER BY `announcements`.`id` DESC" +
      " LIMIT ? OFFSET ?";
    args.push(userId, userId);

    let page: number;
    if (!req.query.page) {
      page = 1;
    } else {
      page = parseInt(req.query.page, 10);
      if (isNaN(page) || page <= 0) {
        return res.status(400).type("text").send("Invalid page.");
      }
    }
    const limit = 20;
    const offset = limit * (page - 1);
    // limitより多く上限を設定し、実際にlimitより多くレコードが取得できた場合は次のページが存在する
    args.push(limit + 1, offset);

    const db = await pool.getConnection();
    try {
      await db.beginTransaction();

      let [announcements] = await db.query<AnnouncementWithoutDetailRow[]>(
        query,
        args
      );

      const [[{ unreadCount }]] = await db.query<
        ({ unreadCount: number } & RowDataPacket)[]
      >(
        "SELECT COUNT(*) AS `unreadCount` FROM `unread_announcements` WHERE `user_id` = ? AND NOT `is_deleted`",
        [userId]
      );

      await db.commit();

      const links = [];
      const linkUrl = new URL(
        req.originalUrl,
        `${req.protocol}://${req.hostname}`
      );

      const q = linkUrl.searchParams;
      if (page > 1) {
        q.set("page", `${page - 1}`);
        links.push(`<${linkUrl.pathname}?${q}>; rel="prev"`);
      }
      if (announcements.length > limit) {
        q.set("page", `${page + 1}`);
        links.push(`<${linkUrl.pathname}?${q}>; rel="next"`);
      }
      if (links.length > 0) {
        res.append("Link", links.join(","));
      }

      if (announcements.length === limit + 1) {
        announcements = announcements.slice(0, announcements.length - 1);
      }

      // 対象になっているお知らせが0件の時は空配列を返却

      const response: GetAnnouncementsResponse = {
        unread_count: unreadCount,
        announcements: announcements.map((announcement) => {
          return {
            ...announcement,
            unread: !!announcement.unread,
          };
        }),
      };
      return res.status(200).json(response);
    } catch (err) {
      console.error(err);
      await db.rollback();
      return res.status(500).send();
    } finally {
      db.release();
    }
  }
);

interface Announcement extends RowDataPacket {
  id: string;
  course_id: string;
  title: string;
  message: string;
}

interface AddAnnouncementRequest {
  id: string;
  course_id: string;
  title: string;
  message: string;
}

function isValidAddAnnouncementRequest(
  body: AddAnnouncementRequest
): body is AddAnnouncementRequest {
  return (
    typeof body === "object" &&
    typeof body.id === "string" &&
    typeof body.course_id === "string" &&
    typeof body.title === "string" &&
    typeof body.message === "string"
  );
}

// POST /api/announcements 新規お知らせ追加
announcementsApi.post("", isAdmin, async (req, res) => {
  const request = req.body;
  if (!isValidAddAnnouncementRequest(request)) {
    return res.status(400).type("text").send("Invalid format.");
  }

  const db = await pool.getConnection();
  try {
    await db.beginTransaction();

    const [[{ cnt }]] = await db.query<({ cnt: number } & RowDataPacket)[]>(
      "SELECT COUNT(*) AS `cnt` FROM `courses` WHERE `id` = ?",
      [request.course_id]
    );
    if (cnt === 0) {
      await db.rollback();
      return res.status(404).type("text").send("No such course.");
    }

    try {
      await db.query(
        "INSERT INTO `announcements` (`id`, `course_id`, `title`, `message`) VALUES (?, ?, ?, ?)",
        [request.id, request.course_id, request.title, request.message]
      );
    } catch (err) {
      await db.rollback();
      if (
        err &&
        typeof err === "object" &&
        (err as { errno: number }).errno === MysqlErrNumDuplicateEntry
      ) {
        const [[announcement]] = await db.query<Announcement[]>(
          "SELECT * FROM `announcements` WHERE `id` = ?",
          [request.id]
        );
        if (
          request.course_id !== announcement.course_id ||
          request.title !== announcement.title ||
          request.message !== announcement.message
        ) {
          return res.status(409).json({
            message: "An announcement with the same id already exists.",
          });
        }
        return res.status(201).send();
      }
      console.error(err);
      return res.status(500).send();
    }

    const [targets] = await db.query<User[]>(
      "SELECT `users`.* FROM `users`" +
        " JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`" +
        " WHERE `registrations`.`course_id` = ?",
      [request.course_id]
    );
    for (const user of targets) {
      await db.query(
        "INSERT INTO `unread_announcements` (`announcement_id`, `user_id`) VALUES (?, ?)",
        [request.id, user.id]
      );
    }

    await db.commit();

    return res.status(201).send();
  } catch (err) {
    console.error(err);
    await db.rollback();
    return res.status(500).send();
  } finally {
    db.release();
  }
});

interface AnnouncementDetail {
  id: string;
  course_id: string;
  course_name: string;
  title: string;
  message: string;
  unread: boolean;
}

interface AnnouncementDetailRow
  extends Omit<AnnouncementDetail, "unread">,
    RowDataPacket {
  unread: number;
}

// GET /api/announcements/:announcementId お知らせ詳細取得
announcementsApi.get(
  "/:announcementId",
  async (req: express.Request<{ announcementId: string }>, res) => {
    let userId: string;
    try {
      [userId] = getUserInfo(req.session);
    } catch (err) {
      console.error(err);
      return res.status(500).send();
    }

    const announcementId = req.params.announcementId;

    const db = await pool.getConnection();
    try {
      await db.beginTransaction();

      const [[announcement]] = await db.query<AnnouncementDetailRow[]>(
        "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name` AS `course_name`, `announcements`.`title`, `announcements`.`message`, NOT `unread_announcements`.`is_deleted` AS `unread`" +
          " FROM `announcements`" +
          " JOIN `courses` ON `courses`.`id` = `announcements`.`course_id`" +
          " JOIN `unread_announcements` ON `unread_announcements`.`announcement_id` = `announcements`.`id`" +
          " WHERE `announcements`.`id` = ?" +
          " AND `unread_announcements`.`user_id` = ?",
        [announcementId, userId]
      );
      if (!announcement) {
        await db.rollback();
        return res.status(404).type("text").send("No such announcement.");
      }

      const [[{ registrationCount }]] = await db.query<
        ({ registrationCount: number } & RowDataPacket)[]
      >(
        "SELECT COUNT(*) AS `registrationCount` FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?",
        [announcement.course_id, userId]
      );
      if (registrationCount === 0) {
        await db.rollback();
        return res.status(404).type("text").send("No such announcement.");
      }

      await db.query(
        "UPDATE `unread_announcements` SET `is_deleted` = true WHERE `announcement_id` = ? AND `user_id` = ?",
        [announcementId, userId]
      );

      await db.commit();

      const response: AnnouncementDetail = {
        ...announcement,
        unread: !!announcement.unread,
      };
      return res.status(200).json(response);
    } catch (err) {
      console.error(err);
      await db.rollback();
      return res.status(500).send();
    } finally {
      db.release();
    }
  }
);

app.listen(parseInt(process.env["PORT"] ?? "7000", 10));
