<?php

declare(strict_types=1);

use Fig\Http\Message\StatusCodeInterface;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Server\RequestHandlerInterface as RequestHandler;
use Psr\Log\LoggerInterface;
use Slim\App;
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
        // TODO: 実装

        return $response;
    }

    /**
     * getGrades GET /api/users/me/grades 成績取得
     */
    public function getGrades(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
    }

    /**
     * searchCourses GET /api/courses 科目検索
     */
    public function searchCourses(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
    }

    /**
     * getCourseDetail GET /api/courses/:courseId 科目詳細の取得
     */
    public function getCourseDetail(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
    }

    /**
     * addCourse POST /api/courses 新規科目登録
     */
    public function addCourse(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
    }

    /**
     * setCourseStatus PUT /api/courses/:courseId/status 科目のステータスを変更
     */
    public function setCourseStatus(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
    }

    /**
     * getClasses GET /api/courses/:courseId/classes 科目に紐づく講義一覧の取得
     */
    public function getClasses(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
    }

    /**
     * submitAssignment POST /api/courses/:courseId/classes/:classId/assignments 課題の提出
     */
    public function submitAssignment(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
    }

    /**
     * registerScores PUT /api/courses/:courseId/classes/:classId/assignments/scores 成績登録
     */
    public function registerScores(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
    }

    /**
     * downloadSubmittedAssignments GET /api/courses/:courseId/classes/:classId/assignments/export 提出済みの課題ファイルをzip形式で一括ダウンロード
     */
    public function downloadSubmittedAssignments(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
    }

    /**
     * @throws RuntimeException
     */
    private function createSubmissionsZip(string $zipFilePath, string $classId, array $submissions): void
    {
    }

    /**
     * addClass POST /api/courses/:courseId/classes 新規講義(&課題)追加
     */
    public function addClass(Request $request, Response $response): Response
    {
        // TODO: 実装

        return $response;
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
