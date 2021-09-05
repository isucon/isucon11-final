import { readFile } from "fs/promises";

import bcrypt from "bcrypt";
import session from "cookie-session";
import express from "express";
import morgan from "morgan";
import mysql, { RowDataPacket } from "mysql2/promise";

import { getDbInfo } from "./db";
import {
  averageFloat,
  averageInt,
  max,
  min,
  tScoreFloat,
  tScoreInt,
} from "./util";

const SqlDirectory = "../sql/";
// const AssignmentsDirectory = "../assignments/";
const SessionName = "isucholar_nodejs";

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
const syllabusApi = express.Router();
app.use("/api", api);
api.use(isLoggedIn);
api.use("/users", usersApi);
api.use("/syllabus", syllabusApi);

interface InitializeResponse {
  language: string;
}

// POST /initialize 初期化エンドポイント
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
    return res.status(401).send("You are not logged in.");
  }
  if (!("userID" in req.session)) {
    return res.status(401).send("You are not logged in.");
  }
  next();
}

// admin確認用middleware
// TODO

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

// POST /logout ログアウト
app.post("/logout", (req, res) => {
  req.session = null;
  return res.status(200).send();
});

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

// PUT /api/users/me/courses 履修登録
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
    let userId: string;
    try {
      [userId] = getUserInfo(req.session);
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
          "INSERT INTO `registrations` (`course_id`, `user_id`) VALUES (?, ?)",
          [course.id, userId]
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
  submitters: number; // 提出した生徒数
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

        const [[{ myScore }]] = await db.query<
          ({ score: number } & RowDataPacket)[]
        >(
          "SELECT `submissions`.`score` FROM `submissions` WHERE `user_id` = ? AND `class_id` = ?",
          [userId, cls.id]
        );
        if (typeof myScore !== "number") {
          classScores.push({
            class_id: cls.id,
            part: cls.part,
            title: cls.title,
            score: null,
            submitters: submissionsCount,
          });
        } else {
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

      // この科目を受講している学生のTotalScore一覧を取得
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
      myGPA += myTotalScore * course.credit;
      myCredits += course.Credit;
    }
    if (myCredits > 0) {
      myGPA = myGPA / 100 / myCredits;
    }

    // GPAの統計値
    // 一つでも科目を履修している学生のGPA一覧
    const [rows] = await db.query<({ gpa: number } & RowDataPacket)[]>(
      "SELECT IFNULL(SUM(`submissions`.`score` * `courses`.`credit`), 0) / 100 / `credits`.`credits` AS `gpa`" +
        " FROM `users`" +
        " JOIN (" +
        "     SELECT `users`.`id` AS `user_id`, SUM(`courses`.`credit`) AS `credits`" +
        "     FROM `users`" +
        "     JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`" +
        "     JOIN `courses` ON `registrations`.`course_id` = `courses`.`id`" +
        "     GROUP BY `users`.`id`" +
        " ) AS `credits` ON `credits`.`user_id` = `users`.`id`" +
        " JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`" +
        " JOIN `courses` ON `registrations`.`course_id` = `courses`.`id`" +
        " LEFT JOIN `classes` ON `courses`.`id` = `classes`.`course_id`" +
        " LEFT JOIN `submissions` ON `users`.`id` = `submissions`.`user_id` AND `submissions`.`class_id` = `classes`.`id`" +
        " WHERE `users`.`type` = ?" +
        " GROUP BY `users`.`id`",
      [UserType.Student]
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
    return res.status(500).send();
  } finally {
    db.release();
  }
});

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

// GET /api/syllabus 科目検索
syllabusApi.get(
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
      if (!isNaN(credit)) {
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
      if (!isNaN(period)) {
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
        keywordsCondition += " AND `courses`.`name` LIKE ?";
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
    if (req.query.page) {
      page = parseInt(req.query.page, 10);
      if (isNaN(page) || page <= 0) {
        return res.status(400).send("Invalid page.");
      }
    } else {
      page = 1;
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

      if (response.length == limit + 1) {
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

syllabusApi.get("/:courseId", async (req, res) => {
  return res.status(200).send("hoge");
});

app.listen(parseInt(process.env["PORT"] ?? "7000", 10));
