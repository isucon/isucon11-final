<?php

declare(strict_types=1);

use Fig\Http\Message\StatusCodeInterface;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Server\RequestHandlerInterface as RequestHandler;
use Psr\Log\LoggerInterface;
use Slim\App;
use Slim\Psr7\Stream;
use Slim\Routing\RouteCollectorProxy;
use SlimSession\Helper as SessionHelper;

require __DIR__ . '/classes.php';
require __DIR__ . '/util.php';

return function (App $app) {
    $app->options('/{routes:.*}', function (Request $request, Response $response) {
        // CORS Pre-Flight OPTIONS Request Handler
        return $response;
    });

    $app->post('/initialize', Handler::class . ':initialize');

    $app->post('/login', Handler::class . ':login');
    $app->post('/logout', Handler::class . ':logout');

    /**
     * isLoggedIn ログイン確認用middleware
     */
    $isLoggedIn = function (Request $request, RequestHandler $handler) use ($app): Response {
        /** @var \Psr\Container\ContainerInterface $container */
        $container = $app->getContainer();
        /** @var SessionHelper $session */
        $session = $container->get(SessionHelper::class);

        if (!$session->exists('userID')) {
            $response = $app->getResponseFactory()->createResponse();
            $response->getBody()->write('You are not logged in.');

            return $response->withStatus(StatusCodeInterface::STATUS_UNAUTHORIZED);
        }

        return $handler->handle($request);
    };

    /**
     * isAdmin admin確認用middleware
     */
    $isAdmin = function (Request $request, RequestHandler $handler) use ($app): Response {
        /** @var \Psr\Container\ContainerInterface $container */
        $container = $app->getContainer();
        /** @var SessionHelper $session */
        $session = $container->get(SessionHelper::class);
        /** @var LoggerInterface $logger */
        $logger = $container->get(LoggerInterface::class);

        if (!$session->exists('isAdmin')) {
            $logger->error('failed to get isAdmin from session');

            return $app->getResponseFactory()
                ->createResponse()
                ->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        if (!$session->get('isAdmin')) {
            $response = $app->getResponseFactory()->createResponse();
            $response->getBody()->write('You are not admin user.');

            return $response->withStatus(StatusCodeInterface::STATUS_FORBIDDEN);
        }

        return $handler->handle($request);
    };

    $app->group('/api', function (RouteCollectorProxy $api) use ($isAdmin) {
        $api->group('/users', function (RouteCollectorProxy $usersApi) {
            $usersApi->get('/me', Handler::class . ':getMe');
            $usersApi->get('/me/courses', Handler::class . ':getRegisteredCourses');
            $usersApi->put('/me/courses', Handler::class . ':registerCourses');
            $usersApi->get('/me/grades', Handler::class . ':getGrades');
        });

        $api->group('/courses', function (RouteCollectorProxy $coursesApi) use ($isAdmin) {
            $coursesApi->get('', Handler::class . ':searchCourses');
            $coursesApi->post('', Handler::class . ':addCourse')->add($isAdmin);
            $coursesApi->get("/{courseId}", Handler::class . ':getCourseDetail');
            $coursesApi->put('/{courseId}/status', Handler::class . ':setCourseStatus')->add($isAdmin);
            $coursesApi->get('/{courseId}/classes', Handler::class . ':getClasses');
            $coursesApi->post('/{courseId}/classes', Handler::class . ':addClass')->add($isAdmin);
            $coursesApi->post('/{courseId}/classes/{classId}/assignments', Handler::class . ':submitAssignment');
            $coursesApi->put('/{courseId}/classes/{classId}/assignments/scores', Handler::class . ':registerScores')->add($isAdmin);
            $coursesApi->get('/{courseId}/classes/{classId}/assignments/export', Handler::class . ':downloadSubmittedAssignments')->add($isAdmin);
        });

        $api->group('/announcements', function (RouteCollectorProxy $announcementsApi) use ($isAdmin) {
            $announcementsApi->get('', Handler::class . ':getAnnouncementList');
            $announcementsApi->post('', Handler::class . ':addAnnouncement')->add($isAdmin);
            $announcementsApi->get('/{announcementId}', Handler::class . ':getAnnouncementDetail');
        });
    })->add($isLoggedIn);
};

final class Handler
{
    private const SQL_DIRECTORY                 = __DIR__ . '/../../sql/';
    private const ASSIGNMENTS_DIRECTORY         = __DIR__ . '/../../assignments/';
    private const MYSQL_ERR_NUM_DUPLICATE_ENTRY = 1062;

    private const USER_TYPE_STUDENT = 'student';
    private const USER_TYPE_TEACHER = 'teacher';

    private const COURSE_TYPE_LIBERAL_ARTS = 'liberal-arts';
    private const COURSE_TYPE_MAJOR_SUBJECTS = 'major-subjects';

    private const DAY_OF_WEEK_MONDAY = 'monday';
    private const DAY_OF_WEEK_TUESDAY = 'tuesday';
    private const DAY_OF_WEEK_WEDNESDAY = 'wednesday';
    private const DAY_OF_WEEK_THURSDAY = 'thursday';
    private const DAY_OF_WEEK_FRIDAY = 'friday';

    private const DAYS_OF_WEEK = [
        self::DAY_OF_WEEK_MONDAY,
        self::DAY_OF_WEEK_TUESDAY,
        self::DAY_OF_WEEK_WEDNESDAY,
        self::DAY_OF_WEEK_THURSDAY,
        self::DAY_OF_WEEK_FRIDAY,
    ];

    private const COURSE_STATUS_REGISTRATION = 'registration';
    private const COURSE_STATUS_IN_PROGRESS  = 'in-progress';
    private const COURSE_STATUS_CLOSED       = 'closed';

    public function __construct(
        private PDO $dbh,
        private SessionHelper $session,
        private LoggerInterface $logger
    ) {
    }

    /**
     * initialize POST /initialize 初期化エンドポイント
     */
    public function initialize(Request $request, Response $response): Response
    {
        $files = [
            '1_schema.sql',
            '2_init.sql',
        ];

        foreach ($files as $file) {
            $data = file_get_contents(self::SQL_DIRECTORY . $file);
            if ($data === false) {
                $this->logger->error('failed to read file: ' . self::SQL_DIRECTORY . $file);

                return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
            }

            try {
                $this->dbh->exec($data);
            } catch (PDOException $e) {
                $this->logger->error('db error: ' . $e->errorInfo[2]);

                return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
            }
        }

        if (exec(sprintf('rm -rf %s', escapeshellarg(self::ASSIGNMENTS_DIRECTORY))) === false) {
            $this->logger->error('failed to remove directory: ' . self::ASSIGNMENTS_DIRECTORY);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        if (exec(sprintf('mkdir %s', escapeshellarg(self::ASSIGNMENTS_DIRECTORY))) === false) {
            $this->logger->error('failed to make directory: ' . self::ASSIGNMENTS_DIRECTORY);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        $res = new InitializeResponse(language: 'php');

        return $this->jsonResponse($response, $res);
    }

    /**
     * @return array{0: string, 1: string, 2: bool, 3: string} [userId, userName, isAdmin, error]
     */
    private function getUserInfo(): array
    {
        $userId = $this->session->get('userID');
        if ($userId === null) {
            return ['', '', false, 'failed to get userID from session'];
        }

        $userName  = $this->session->get('userName');
        if ($userName === null) {
            return ['', '', false, 'failed to get userName from session'];
        }

        $isAdmin  = $this->session->get('isAdmin');
        if ($isAdmin === null) {
            return ['', '', false, 'failed to get isAdmin from session'];
        }

        return [$userId, $userName, $isAdmin, ''];
    }

    /**
     * login POST /login ログイン
     */
    public function login(Request $request, Response $response): Response
    {
        try {
            $req = LoginRequest::fromJson((string)$request->getBody());
        } catch (UnexpectedValueException) {
            // TODO: エラーハンドリング方法の統一待ち
            $response->getBody()->write('bind request: ');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        try {
            $stmt = $this->dbh->prepare('SELECT * FROM `users` WHERE `code` = ?');
            $stmt->execute([$req->code]);
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        if ($row === false) {
            $response->getBody()->write('Code or Password is wrong.');

            return $response->withStatus(StatusCodeInterface::STATUS_UNAUTHORIZED);
        }
        $user = User::fromDbRow($row);

        if (!password_verify($req->password, $user->hashedPassword)) {
            $response->getBody()->write('Code or Password is wrong.');

            return $response->withStatus(StatusCodeInterface::STATUS_UNAUTHORIZED);
        }

        if ($this->session->get('userID') === $user->id) {
            $response->getBody()->write('You are already logged in.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        $this->session->set('userID', $user->id);
        $this->session->set('userName', $user->name);
        $this->session->set('isAdmin', $user->type === self::USER_TYPE_TEACHER);

        return $response;
    }

    /**
     * logout POST /logout ログアウト
     */
    public function logout(Request $request, Response $response): Response
    {
        $this->session->destroy();

        return $response;
    }

    /**
     * getMe GET /api/users/me 自身の情報を取得
     */
    public function getMe(Request $request, Response $response): Response
    {
        [$userId, $userName, $isAdmin, $err] = $this->getUserInfo();

        if ($err !== '') {
            $this->logger->error($err);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        try {
            $stmt = $this->dbh->prepare('SELECT `code` FROM `users` WHERE `id` = ?');
            $stmt->execute([$userId]);
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        if ($row === false) {
            $this->logger->error('db error: no rows');

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }
        $userCode = $row[0];

        return $this->jsonResponse($response, new GetMeResponse(
            code: $userCode,
            name: $userName,
            isAdmin: $isAdmin
        ));
    }

    /**
     * getRegisteredCourses GET /api/users/me/courses 履修中の科目一覧取得
     */
    public function getRegisteredCourses(Request $request, Response $response): Response
    {
        [$userId, , , $err] = $this->getUserInfo();
        if ($err !== '') {
            $this->logger->error($err);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        /** @var array<Course> $courses */
        $courses = [];
        $query = 'SELECT `courses`.*' .
            ' FROM `courses`' .
            ' JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`' .
            ' WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?';
        try {
            $stmt = $this->dbh->prepare($query);
            $stmt->execute([self::COURSE_STATUS_CLOSED, $userId]);
            while ($row = $stmt->fetch()) {
                $courses[] = Course::fromDbRow($row);
            }
        } catch (PDOException $e) {
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        // 履修科目が0件の時は空配列を返却
        /** @var array<GetRegisteredCourseResponseContent> $res */
        $res = [];
        foreach ($courses as $course) {
            try {
                $stmt = $this->dbh->prepare('SELECT * FROM `users` WHERE `id` = ?');
                $stmt->execute([$course->teacherId]);
                $row = $stmt->fetch();
            } catch (PDOException $e) {
                $this->logger->error('db error: ' . $e->errorInfo[2]);

                return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
            }

            if ($row === false) {
                $this->logger->error('db error: no rows');

                return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
            }
            $teacher = User::fromDbRow($row);

            $res[] = new GetRegisteredCourseResponseContent(
                id:        $course->id,
                name:      $course->name,
                teacher:   $teacher->name,
                period:    $course->period,
                dayOfWeek: $course->dayOfWeek
            );
        }

        return $this->jsonResponse($response, $res);
    }

    /**
     * registerCourses PUT /api/users/me/courses 履修登録
     */
    public function registerCourses(Request $request, Response $response): Response
    {
        [$userId, , , $err] = $this->getUserInfo();
        if ($err !== '') {
            $this->logger->error($err);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        try {
            $req = RegisterCourseRequestContent::listFromJson((string)$request->getBody());
        } catch (UnexpectedValueException) {
            $response->getBody()->write('Invalid format.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        usort($req, function (RegisterCourseRequestContent $one, RegisterCourseRequestContent $another) {
            return $one->id <=> $another->id;
        });

        $this->dbh->beginTransaction();

        $errors = new RegisterCoursesErrorResponse();
        /** @var array<Course> $newlyAdded */
        $newlyAdded = [];
        foreach ($req as $courseReq) {
            $courseId = $courseReq->id;
            try {
                $stmt = $this->dbh->prepare('SELECT * FROM `courses` WHERE `id` = ? FOR SHARE');
                $stmt->execute([$courseId]);
                $row = $stmt->fetch();
            } catch (PDOException $e) {
                $this->dbh->rollBack();
                $this->logger->error('db error: ' . $e->errorInfo[2]);

                return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
            }

            if ($row === false) {
                $errors->courseNotFound[] = $courseReq->id;
                continue;
            }
            $course = Course::fromDbRow($row);

            if ($course->status !== self::COURSE_STATUS_REGISTRATION) {
                $errors->notRegistrableStatus[] = $course->id;
            }

            // すでに履修登録済みの科目は無視する
            try {
                $stmt = $this->dbh->prepare('SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?');
                $stmt->execute([$course->id, $userId]);
                $count = $stmt->fetch()[0];
            } catch (PDOException $e) {
                $this->dbh->rollBack();
                $this->logger->error('db error: ' . $e->errorInfo[2]);

                return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
            }

            if ($count > 0) {
                continue;
            }

            $newlyAdded[] = $course;
        }

        /** @var array<Course> $alreadyRegistered */
        $alreadyRegistered = [];
        $query = 'SELECT `courses`.*' .
            ' FROM `courses`' .
            ' JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`' .
            ' WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?';
        try {
            $stmt = $this->dbh->prepare($query);
            $stmt->execute([self::COURSE_STATUS_CLOSED, $userId]);
            while ($row = $stmt->fetch()) {
                $alreadyRegistered[] = Course::fromDbRow($row);
            }
        } catch (PDOException $e) {
            $this->dbh->rollBack();
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        $alreadyRegistered = array_merge($alreadyRegistered, $newlyAdded);
        foreach ($newlyAdded as $course1) {
            foreach ($alreadyRegistered as $course2) {
                if ($course1->id !== $course2->id && $course1->period === $course2->period && $course1->dayOfWeek === $course2->dayOfWeek) {
                    $errors->scheduleConflict[] = $course1->id;
                }
            }
        }

        if (count($errors->courseNotFound) > 0 || count($errors->notRegistrableStatus) > 0 || count($errors->scheduleConflict) > 0) {
            $this->dbh->rollBack();

            return $this->jsonResponse($response, $errors, StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        foreach ($newlyAdded as $course) {
            try {
                $stmt = $this->dbh->prepare('INSERT INTO `registrations` (`course_id`, `user_id`) VALUES (?, ?)');
                $stmt->execute([$course->id, $userId]);
            } catch (PDOException $e) {
                $this->dbh->rollBack();
                $this->logger->error('db error: ' . $e->errorInfo[2]);

                return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
            }
        }

        $this->dbh->commit();

        return $response;
    }

    /**
     * getGrades GET /api/users/me/grades 成績取得
     */
    public function getGrades(Request $request, Response $response): Response
    {
        [$userId, , , $err] = $this->getUserInfo();
        if ($err !== '') {
            $this->logger->error($err);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        // 履修している科目一覧取得
        /** @var array<Course> $registeredCourses */
        $registeredCourses = [];
        $query = 'SELECT `courses`.*' .
            ' FROM `registrations`' .
            ' JOIN `courses` ON `registrations`.`course_id` = `courses`.`id`' .
            ' WHERE `user_id` = ?';
        try {
            $stmt = $this->dbh->prepare($query);
            $stmt->execute([$userId]);
            while ($row = $stmt->fetch()) {
                $registeredCourses[] = Course::fromDbRow($row);
            }
        } catch (PDOException $e) {
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        // 科目毎の成績計算処理
        /** @var CourseResult $courseResult */
        $courseResults = [];
        $myGpa = 0.0;
        $myCredits = 0;
        foreach ($registeredCourses as $course) {
            // 講義一覧の取得
            /** @var array<Klass> $classes */
            $classes = [];
            $query = 'SELECT *' .
                ' FROM `classes`' .
                ' WHERE `course_id` = ?' .
                ' ORDER BY `part` DESC';
            try {
                $stmt = $this->dbh->prepare($query);
                $stmt->execute([$course->id]);
                while ($row = $stmt->fetch()) {
                    $classes[] = Klass::fromDbRow($row);
                }
            } catch (PDOException $e) {
                $this->logger->error('db error: ' . $e->errorInfo[2]);

                return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
            }

            // 講義毎の成績計算処理
            /** @var array<ClassScore> $classScores */
            $classScores = [];
            $myTotalScore = 0;
            foreach ($classes as $class) {
                try {
                    $stmt = $this->dbh->prepare('SELECT COUNT(*) FROM `submissions` WHERE `class_id` = ?');
                    $stmt->execute([$class->id]);
                    $submissionsCount = $stmt->fetch()[0];
                } catch (PDOException $e) {
                    $this->logger->error('db error: ' . $e->errorInfo[2]);

                    return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
                }

                try {
                    $stmt = $this->dbh->prepare('SELECT `submissions`.`score` FROM `submissions` WHERE `user_id` = ? AND `class_id` = ?');
                    $stmt->execute([$userId, $class->id]);
                    $row = $stmt->fetch();
                } catch (PDOException $e) {
                    $this->logger->error('db error: ' . $e->errorInfo[2]);

                    return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
                }

                if ($row === false || is_null($score = $row[0])) {
                    $classScores[] = new ClassScore(
                        classId: $class->id,
                        part: $class->part,
                        title: $class->title,
                        score: null,
                        submitters: $submissionsCount,
                    );
                } else {
                    $myTotalScore += $score;
                    $classScores[] = new ClassScore(
                        classId: $class->id,
                        part: $class->part,
                        title: $class->title,
                        score: $score,
                        submitters: $submissionsCount,
                    );
                }
            }

            /** @var array<int> $totals */
            $totals = [];
            $query = 'SELECT IFNULL(SUM(`submissions`.`score`), 0) AS `total_score`' .
                ' FROM `users`' .
                ' JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`' .
                ' JOIN `courses` ON `registrations`.`course_id` = `courses`.`id`' .
                ' LEFT JOIN `classes` ON `courses`.`id` = `classes`.`course_id`' .
                ' LEFT JOIN `submissions` ON `users`.`id` = `submissions`.`user_id` AND `submissions`.`class_id` = `classes`.`id`' .
                ' WHERE `courses`.`id` = ?' .
                ' GROUP BY `users`.`id`';
            try {
                $stmt = $this->dbh->prepare($query);
                $stmt->execute([$course->id]);
                while ($row = $stmt->fetch) {
                    $totals[] = $row['total_score'];
                }
            } catch (PDOException $e) {
                $this->logger->error('db error: ' . $e->errorInfo[2]);

                return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
            }

            $courseResults[] = new CourseResult(
                name: $course->name,
                code: $course->code,
                totalScore: $myTotalScore,
                totalScoreTScore: util\tScoreInt($myTotalScore, $totals),
                totalScoreAvg: util\average($totals, 0),
                totalScoreMax: util\max($totals, 0),
                totalScoreMin: util\min($totals, 0),
                classScores: $classScores,
            );

            // 自分のGPA計算
            if ($course->status === self::COURSE_STATUS_CLOSED) {
                $myGpa += $myTotalScore * $course->credit;
                $myCredits += $course->credit;
            }
        }

        if ($myCredits > 0) {
            $myGpa = (float)$myGpa / 100 / $myCredits;
        }

        // GPAの統計値
        // 一つでも修了した科目（履修した & ステータスがclosedである）がある学生のGPA一覧
        /** @var array<float> $gpas */
        $gpas = [];
        $query = 'SELECT IFNULL(SUM(`submissions`.`score` * `courses`.`credit`), 0) / 100 / `credits`.`credits` AS `gpa`' .
            ' FROM `users`' .
            ' JOIN (' .
            '     SELECT `users`.`id` AS `user_id`, SUM(`courses`.`credit`) AS `credits`' .
            '     FROM `users`' .
            '     JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`' .
            '     JOIN `courses` ON `registrations`.`course_id` = `courses`.`id` AND `courses`.`status` = ?' .
            '     GROUP BY `users`.`id`' .
            ' ) AS `credits` ON `credits`.`user_id` = `users`.`id`' .
            ' JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`' .
            ' JOIN `courses` ON `registrations`.`course_id` = `courses`.`id` AND `courses`.`status` = ?' .
            ' LEFT JOIN `classes` ON `courses`.`id` = `classes`.`course_id`' .
            ' LEFT JOIN `submissions` ON `users`.`id` = `submissions`.`user_id` AND `submissions`.`class_id` = `classes`.`id`' .
            ' WHERE `users`.`type` = ?' .
            ' GROUP BY `users`.`id`';
        try {
            $stmt = $this->dbh->prepare($query);
            $stmt->execute([self::COURSE_STATUS_CLOSED, self::COURSE_STATUS_CLOSED, self::USER_TYPE_STUDENT]);
            while ($row = $stmt->fetch()) {
                $gpas[] = $row['gpa'];
            }
        } catch (PDOException $e) {
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        $res = new GetGradeResponse(
            summary: new Summary(
                credits: $myCredits,
                gpa: $myGpa,
                gpaTScore: util\tScoreFloat($myGpa, $gpas),
                gpaAvg: util\average($gpas, 0),
                gpaMax: util\max($gpas, 0),
                gpaMin: util\min($gpas, 0),
            ),
            courseResults: $courseResults
        );

        return $this->jsonResponse($response, $res);
    }

    /**
     * searchCourses GET /api/courses 科目検索
     */
    public function searchCourses(Request $request, Response $response): Response
    {
        $query = 'SELECT `courses`.*, `users`.`name` AS `teacher`' .
            ' FROM `courses` JOIN `users` ON `courses`.`teacher_id` = `users`.`id`' .
            ' WHERE 1=1';
        $condition = '';
        $args = [];

        // 無効な検索条件はエラーを返さず無視して良い

        $courseType = $request->getQueryParams()['type'] ?? '';
        if ($courseType !== '') {
            $condition .= ' AND `courses`.`type` = ?';
            $args[] = [$courseType, PDO::PARAM_STR];
        }

        $credit = filter_var($request->getQueryParams()['credit'] ?? null, FILTER_VALIDATE_INT);
        if (is_int($credit) && $credit > 0) {
            $condition .= ' AND `courses`.`credit` = ?';
            $args[] = [$credit, PDO::PARAM_INT];
        }

        $teacher = $request->getQueryParams()['teacher'] ?? '';
        if ($teacher !== '') {
            $condition .= ' AND `users`.`name` = ?';
            $args[] = [$teacher, PDO::PARAM_STR];
        }

        $period = filter_var($request->getQueryParams()['period'] ?? null, FILTER_VALIDATE_INT);
        if (is_int($period) && $period > 0) {
            $condition .= ' AND `courses`.`period` = ?';
            $args[] = [$period, PDO::PARAM_INT];
        }

        $dayOfWeek = $request->getQueryParams()['day_of_week'] ?? '';
        if ($dayOfWeek !== '') {
            $condition .= ' AND `courses`.`day_of_week` = ?';
            $args[] = [$dayOfWeek, PDO::PARAM_STR];
        }

        $keywords = $request->getQueryParams()['keywords'] ?? '';
        if ($keywords !== '') {
            $arr = explode(' ', $keywords);
            $nameCondition = '';
            foreach ($arr as $keyword) {
                $nameCondition .= ' AND `courses`.`name` LIKE ?';
                $args[] = ['%' . $keyword . '%', PDO::PARAM_STR];
            }
            $keywordsCondition = '';
            foreach ($arr as $keyword) {
                $keywordsCondition .= ' AND `courses`.`keywords` LIKE ?';
                $args[] = ['%' . $keyword . '%', PDO::PARAM_STR];
            }
            $condition .= sprintf(' AND ((1=1%s) OR (1=1%s))', $nameCondition, $keywordsCondition);
        }

        $status = $request->getQueryParams()['status'] ?? '';
        if ($status !== '') {
            $condition .= ' AND `courses`.`status` = ?';
            $args[] = [$status, PDO::PARAM_STR];
        }

        $condition .= ' ORDER BY `courses`.`code`';

        $pageStr = $request->getQueryParams()['page'] ?? '';
        if ($pageStr === '') {
            $page = 1;
        } else {
            $page = filter_var($pageStr, FILTER_VALIDATE_INT);
            if (!is_int($page) || $page <= 0) {
                $response->getBody()->write('Invalid page.');

                return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
            }
        }
        $limit = 20;
        $offset = $limit * ($page - 1);

        // limitより多く上限を設定し、実際にlimitより多くレコードが取得できた場合は次のページが存在する
        $condition .= ' LIMIT ? OFFSET ?';
        $args = [...$args, [$limit + 1, PDO::PARAM_INT], [$offset, PDO::PARAM_INT]];

        // 結果が0件の時は空配列を返却
        /** @var array<GetCourseDetailResponse> $res */
        $res = [];
        try {
            $stmt = $this->dbh->prepare($query . $condition);
            foreach ($args as $i => [$value, $type]) {
                $stmt->bindValue($i + 1, $value, $type);
            }
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $res[] = GetCourseDetailResponse::fromDbRow($row);
            }
        } catch (PDOException $e) {
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        /** @var array<string> $links */
        $links = [];

        $q = $request->getQueryParams();
        if ($page > 1) {
            $q['page'] = $page - 1;
            $links[] = sprintf('<%s>; rel="prev"', $request->getUri()->getPath() . '?' . http_build_query($q));
        }
        if (count($res) > $limit) {
            $q['page'] = $page + 1;
            $links[] = sprintf('<%s>; rel="next"', $request->getUri()->getPath() . '?' . http_build_query($q));
        }
        if (count($links) > 0) {
            $response = $response->withHeader('Link', implode(',', $links));
        }

        if (count($res) === $limit + 1) {
            array_pop($res);
        }

        return $this->jsonResponse($response, $res);
    }

    /**
     * getCourseDetail GET /api/courses/:courseId 科目詳細の取得
     */
    public function getCourseDetail(Request $request, Response $response, array $params): Response
    {
        $courseId = $params['courseId'];

        $query = 'SELECT `courses`.*, `users`.`name` AS `teacher`' .
            ' FROM `courses`' .
            ' JOIN `users` ON `courses`.`teacher_id` = `users`.`id`' .
            ' WHERE `courses`.`id` = ?';
        try {
            $stmt = $this->dbh->prepare($query);
            $stmt->execute([$courseId]);
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        if ($row === false) {
            $response->getBody()->write('No such course.');

            return $response->withStatus(StatusCodeInterface::STATUS_NOT_FOUND);
        }
        $res = GetCourseDetailResponse::fromDbRow($row);

        return $this->jsonResponse($response, $res);
    }

    /**
     * addCourse POST /api/courses 新規科目登録
     */
    public function addCourse(Request $request, Response $response): Response
    {
        [$userId, , , $err] = $this->getUserInfo();
        if ($err !== '') {
            $this->logger->error($err);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        try {
            $req = AddCourseRequest::fromJson((string)$request->getBody());
        } catch (UnexpectedValueException) {
            $response->getBody()->write('Invalid format.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        if ($req->type !== self::COURSE_TYPE_LIBERAL_ARTS && $req->type !== self::COURSE_TYPE_MAJOR_SUBJECTS) {
            $response->getBody()->write('Invalid course type.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        if (!in_array($req->dayOfWeek, self::DAYS_OF_WEEK)) {
            $response->getBody()->write('Invalid day of week.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        $this->dbh->beginTransaction();

        $courseId = util\newUlid();
        try {
            $stmt = $this->dbh->prepare('INSERT INTO `courses` (`id`, `code`, `type`, `name`, `description`, `credit`, `period`, `day_of_week`, `teacher_id`, `keywords`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)');
            $stmt->execute([$courseId, $req->code, $req->type, $req->name, $req->description, $req->credit, $req->period, $req->dayOfWeek, $userId, $req->keywords]);
        } catch (PDOException $exception) {
            $this->dbh->rollBack();
            if ($exception->errorInfo[1] === self::MYSQL_ERR_NUM_DUPLICATE_ENTRY) {
                try {
                    $stmt = $this->dbh->prepare('SELECT * FROM `courses` WHERE `code` = ?');
                    $stmt->execute([$req->code]);
                    $row = $stmt->fetch();
                } catch (PDOException $e) {
                    $this->logger->error('db error: ' . $e->errorInfo[2]);

                    return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
                }

                if ($row === false) {
                    $this->logger->error('db error: no rows');

                    return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
                }
                $course = Course::fromDbRow($row);

                if ($req->type !== $course->type || $req->name !== $course->name || $req->description !== $course->description || $req->credit !== $course->credit || $req->period !== $course->period || $req->dayOfWeek !== $course->dayOfWeek || $req->keywords !== $course->keywords) {
                    $response->getBody()->write('A course with the same code already exists.');

                    return $response->withStatus(StatusCodeInterface::STATUS_CONFLICT);
                }
            }

            $this->logger->error('db error: ' . $exception->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        $this->dbh->commit();

        return $this->jsonResponse($response, new AddCourseResponse(id: $courseId), StatusCodeInterface::STATUS_CREATED);
    }

    /**
     * setCourseStatus PUT /api/courses/:courseId/status 科目のステータスを変更
     */
    public function setCourseStatus(Request $request, Response $response, array $params): Response
    {
        $courseId = $params['courseId'];

        try {
            $req = SetCourseStatusRequest::fromJson((string)$request->getBody());
        } catch (UnexpectedValueException) {
            $response->getBody()->write('Invalid format.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        try {
            $stmt = $this->dbh->prepare('SELECT COUNT(*) FROM `courses` WHERE `id` = ?');
            $stmt->execute([$courseId]);
            $count = $stmt->fetch()[0];
        } catch (PDOException $e) {
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        if ($count == 0) {
            $response->getBody()->write('No such course.');

            return $response->withStatus(StatusCodeInterface::STATUS_NOT_FOUND);
        }

        try {
            $stmt = $this->dbh->prepare('UPDATE `courses` SET `status` = ? WHERE `id` = ?');
            $stmt->execute([$req->status, $courseId]);
        } catch (PDOException $e) {
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        return $response;
    }

    /**
     * getClasses GET /api/courses/:courseId/classes 科目に紐づく講義一覧の取得
     */
    public function getClasses(Request $request, Response $response, array $params): Response
    {
        [$userId, , , $err] = $this->getUserInfo();
        if ($err !== '') {
            $this->logger->error($err);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        $courseId = $params['courseId'];

        try {
            $stmt = $this->dbh->prepare('SELECT COUNT(*) FROM `courses` WHERE `id` = ?');
            $stmt->execute([$courseId]);
            $count = $stmt->fetch()[0];
        } catch (PDOException $e) {
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        if ($count == 0) {
            $response->getBody()->write('No such course.');

            return $response->withStatus(StatusCodeInterface::STATUS_NOT_FOUND);
        }

        /** @var array<ClassWithSubmitted> $classes */
        $classes = [];
        $query = 'SELECT `classes`.*, `submissions`.`user_id` IS NOT NULL AS `submitted`' .
            ' FROM `classes`' .
            ' LEFT JOIN `submissions` ON `classes`.`id` = `submissions`.`class_id` AND `submissions`.`user_id` = ?' .
            ' WHERE `classes`.`course_id` = ?' .
            ' ORDER BY `classes`.`part`';
        try {
            $stmt = $this->dbh->prepare($query);
            $stmt->execute([$userId, $courseId]);
            while ($row = $stmt->fetch()) {
                $classes[] = ClassWithSubmitted::fromDbRow($row);
            }
        } catch (PDOException $e) {
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        // 結果が0件の時は空配列を返却
        /** @var array<GetClassResponse> $res */
        $res = [];
        foreach ($classes as $class) {
            $res[] = new GetClassResponse(
                id: $class->id,
                part: $class->part,
                title: $class->title,
                description: $class->description,
                submissionClosed: $class->submissionClosed,
                submitted: $class->submitted,
            );
        }

        return $this->jsonResponse($response, $res);
    }

    /**
     * submitAssignment POST /api/courses/:courseId/classes/:classId/assignments 課題の提出
     */
    public function submitAssignment(Request $request, Response $response, array $params): Response
    {
        [$userId, , , $err] = $this->getUserInfo();
        if ($err !== '') {
            $this->logger->error($err);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        $courseId = $params['courseId'];
        $classId = $params['classId'];

        $this->dbh->beginTransaction();

        try {
            $stmt = $this->dbh->prepare('SELECT `status` FROM `courses` WHERE `id` = ? FOR SHARE');
            $stmt->execute([$courseId]);
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            $this->dbh->rollBack();
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }
        if ($row === false) {
            $this->dbh->rollBack();
            $response->getBody()->write('No such course.');

            return $response->withStatus(StatusCodeInterface::STATUS_NOT_FOUND);
        }
        if ($row['status'] !== self::COURSE_STATUS_IN_PROGRESS) {
            $this->dbh->rollBack();
            $response->getBody()->write('This course is not in progress.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        try {
            $stmt = $this->dbh->prepare('SELECT COUNT(*) FROM `registrations` WHERE `user_id` = ? AND `course_id` = ?');
            $stmt->execute([$userId, $courseId]);
            $registrationCount = $stmt->fetch()[0];
        } catch (PDOException $e) {
            $this->dbh->rollBack();
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }
        if ($registrationCount == 0) {
            $this->dbh->rollBack();
            $response->getBody()->write('You have not taken this course.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        try {
            $stmt = $this->dbh->prepare('SELECT `submission_closed` FROM `classes` WHERE `id` = ? FOR SHARE');
            $stmt->execute([$classId]);
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            $this->dbh->rollBack();
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }
        if ($row === false) {
            $this->dbh->rollBack();
            $response->getBody()->write('No such class.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }
        if ($row['submission_closed']) {
            $this->dbh->rollBack();
            $response->getBody()->write('Submission has been closed for this class.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        /** @var \Psr\Http\Message\UploadedFileInterface|null $file */
        $file = $request->getUploadedFiles()['file'] ?? null;
        if (is_null($file) || $file->getError() !== UPLOAD_ERR_OK) {
            $this->dbh->rollBack();
            $response->getBody()->write('"Invalid file.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        try {
            $stmt = $this->dbh->prepare('INSERT INTO `submissions` (`user_id`, `class_id`, `file_name`) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE `file_name` = VALUES(`file_name`)');
            $stmt->execute([$userId, $classId, $file->getClientFilename()]);
        } catch (PDOException $e) {
            $this->dbh->rollBack();
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        // TODO: stream の扱いについて要検証
        if (file_put_contents(self::ASSIGNMENTS_DIRECTORY . $classId . '-' . $userId . 'pdf', $file->getStream()) === false) {
            $this->dbh->rollBack();
            $this->logger->error('failed to create file: ' . self::ASSIGNMENTS_DIRECTORY . $classId . '-' . $userId . 'pdf');

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        $this->dbh->commit();

        return $response->withStatus(StatusCodeInterface::STATUS_NO_CONTENT);
    }

    /**
     * registerScores PUT /api/courses/:courseId/classes/:classId/assignments/scores 成績登録
     */
    public function registerScores(Request $request, Response $response, array $params): Response
    {
        $classId = $params['classId'];

        $this->dbh->beginTransaction();

        try {
            $stmt = $this->dbh->prepare('SELECT `submission_closed` FROM `classes` WHERE `id` = ? FOR SHARE');
            $stmt->execute([$classId]);
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            $this->dbh->rollBack();
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        if ($row === false) {
            $this->dbh->rollBack();
            $response->getBody()->write('No such class.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }
        if (!$row['submission_closed']) {
            $this->dbh->rollBack();
            $response->getBody()->write('This assignment is not closed yet.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        try {
            $req = Score::listFromJson((string)$request->getBody());
        } catch (UnexpectedValueException) {
            $this->dbh->rollBack();
            $response->getBody()->write('Invalid format.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        foreach ($req as $score) {
            try {
                $stmt = $this->dbh->prepare('UPDATE `submissions` JOIN `users` ON `users`.`id` = `submissions`.`user_id` SET `score` = ? WHERE `users`.`code` = ? AND `class_id` = ?');
                $stmt->execute([$score->score, $score->userCode, $classId]);
            } catch (PDOException $e) {
                $this->dbh->rollBack();
                $this->logger->error('db error: ' . $e->errorInfo[2]);

                return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
            }
        }

        $this->dbh->commit();

        return $response->withStatus(StatusCodeInterface::STATUS_NO_CONTENT);
    }

    /**
     * downloadSubmittedAssignments GET /api/courses/:courseId/classes/:classId/assignments/export 提出済みの課題ファイルをzip形式で一括ダウンロード
     */
    public function downloadSubmittedAssignments(Request $request, Response $response, array $params): Response
    {
        $classId = $params['classId'];

        $this->dbh->beginTransaction();

        try {
            $stmt = $this->dbh->prepare('SELECT COUNT(*) FROM `classes` WHERE `id` = ? FOR UPDATE');
            $stmt->execute([$classId]);
            $classCount = $stmt->fetch()[0];
        } catch (PDOException $e) {
            $this->dbh->rollBack();
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }
        if ($classCount == 0) {
            $this->dbh->rollBack();
            $response->getBody()->write('No such class.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        /** @var array<Submission> $submissions */
        $submissions = [];
        $query = 'SELECT `submissions`.`user_id`, `submissions`.`file_name`, `users`.`code` AS `user_code`' .
            ' FROM `submissions`' .
            ' JOIN `users` ON `users`.`id` = `submissions`.`user_id`' .
            ' WHERE `class_id` = ? FOR SHARE';
        try {
            $stmt = $this->dbh->prepare($query);
            $stmt->execute([$classId]);
            while ($row = $stmt->fetch()) {
                $submissions[] = Submission::fromDbRow($row);
            }
        } catch (PDOException $e) {
            $this->dbh->rollBack();
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        $zipFilePath = self::ASSIGNMENTS_DIRECTORY . $classId . '.zip';
        $err = $this->createSubmissionsZip($zipFilePath, $classId, $submissions);
        if ($err !== '') {
            $this->dbh->rollBack();
            $this->logger->error($err);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        try {
            $stmt = $this->dbh->prepare('UPDATE `classes` SET `submission_closed` = true WHERE `id` = ?');
            $stmt->execute([$classId]);
        } catch (PDOException $e) {
            $this->dbh->rollBack();
            $this->logger->error($e->getMessage());

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        $this->dbh->commit();

        return $response->withHeader('Content-Type', 'application/octet-stream')
            ->withHeader('Content-Disposition', 'attachment; filename=' . $classId . '.zip')
            ->withBody(new Stream(fopen($zipFilePath, 'rb')));
    }

    /**
     * @param array<Submission> $submissions
     */
    private function createSubmissionsZip(string $zipFilePath, string $classId, array $submissions): string
    {
        $tmpDir = self::ASSIGNMENTS_DIRECTORY . $classId . '/';
        if (exec(sprintf('rm -rf %s', escapeshellarg($tmpDir))) === false) {
            return 'failed to remove directory: ' . $tmpDir;
        }
        if (exec(sprintf('mkdir %s', escapeshellarg($tmpDir))) === false) {
            return 'failed to make directory: ' . $tmpDir;
        }

        // ファイル名を指定の形式に変更
        foreach ($submissions as $submission) {
            if (
                exec(sprintf(
                    'cp %s %s',
                    escapeshellarg(self::ASSIGNMENTS_DIRECTORY . $classId . '-' . $submission->userId . 'pdf'),
                    escapeshellarg($tmpDir . $submission->userCode . '-' . $submission->fileName),
                )) === false
            ) {
                return 'failed to copy file: ' . $classId . '-' . $submission->userId . 'pdf';
            }
        }

        // -i 'tmpDir/*': 空zipを許す
        if (exec(sprintf('zip -j -r %s  %s -i %s*', $zipFilePath, $tmpDir, $tmpDir)) === false) {
            return 'failed to zip';
        }

        return '';
    }

    /**
     * addClass POST /api/courses/:courseId/classes 新規講義(&課題)追加
     */
    public function addClass(Request $request, Response $response, array $params): Response
    {
        $courseId = $params['courseId'];

        try {
            $req = AddClassRequest::fromJson((string)$request->getBody());
        } catch (UnexpectedValueException) {
            $response->getBody()->write('Invalid format.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        $this->dbh->beginTransaction();

        try {
            $stmt = $this->dbh->prepare('SELECT * FROM `courses` WHERE `id` = ? FOR SHARE');
            $stmt->execute([$courseId]);
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            $this->dbh->rollBack();
            $this->logger->error('db error: ' . $e->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }
        if ($row === false) {
            $this->dbh->rollBack();
            $response->getBody()->write('No such course.');

            return $response->withStatus(StatusCodeInterface::STATUS_NOT_FOUND);
        }
        $course = Course::fromDbRow($row);
        if ($course->status !== self::COURSE_STATUS_IN_PROGRESS) {
            $this->dbh->rollBack();
            $response->getBody()->write('This course is not in-progress.');

            return $response->withStatus(StatusCodeInterface::STATUS_BAD_REQUEST);
        }

        $classId = util\newUlid();
        try {
            $stmt = $this->dbh->prepare('INSERT INTO `classes` (`id`, `course_id`, `part`, `title`, `description`) VALUES (?, ?, ?, ?, ?)');
            $stmt->execute([$classId, $courseId, $req->part, $req->title, $req->description]);
        } catch (PDOException $exception) {
            $this->dbh->rollBack();
            if ($exception->errorInfo[1] === self::MYSQL_ERR_NUM_DUPLICATE_ENTRY) {
                try {
                    $stmt = $this->dbh->prepare('SELECT * FROM `classes` WHERE `course_id` = ? AND `part` = ?');
                    $stmt->execute([$courseId, $req->part]);
                    $row = $stmt->fetch();
                } catch (PDOException $e) {
                    $this->logger->error('db error: ' . $e->errorInfo[2]);

                    return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
                }

                if ($row === false) {
                    $this->logger->error('db error: no rows');

                    return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
                }
                $class = Klass::fromDbRow($row);

                if ($req->title !== $class->title || $req->description !== $class->description) {
                    $response->getBody()->write('A class with the same part already exists.');

                    return $response->withStatus(StatusCodeInterface::STATUS_CONFLICT);
                }

                return $this->jsonResponse($response, new AddClassResponse(classId: $class->id), StatusCodeInterface::STATUS_CREATED);
            }

            $this->logger->error('db error: ' . $exception->errorInfo[2]);

            return $response->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        $this->dbh->commit();

        return $this->jsonResponse($response, new AddClassResponse(classId: $classId), StatusCodeInterface::STATUS_CREATED);
    }

    /**
     * getAnnouncementList GET /api/announcements お知らせ一覧取得
     */
    public function getAnnouncementList(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
    }

    /**
     * getAnnouncementDetail GET /api/announcements/:announcementId お知らせ詳細取得
     */
    public function getAnnouncementDetail(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
    }

    /**
     * addAnnouncement POST /api/announcements 新規お知らせ追加
     */
    public function addAnnouncement(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
    }

    /**
     * @throws UnexpectedValueException
     */
    private function jsonResponse(Response $response, JsonSerializable|array $data, int $statusCode = StatusCodeInterface::STATUS_OK): Response
    {
        $responseBody = json_encode($data, JSON_UNESCAPED_UNICODE);
        if ($responseBody === false) {
            throw new UnexpectedValueException('failed to json_encode');
        }

        $response->getBody()->write($responseBody);

        return $response->withStatus($statusCode)
            ->withHeader('Content-Type', 'application/json; charset=UTF-8');
    }
}
