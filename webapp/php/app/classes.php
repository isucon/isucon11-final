<?php

final class InitializeResponse implements JsonSerializable
{
    public function __construct(public string $language)
    {
    }

    /**
     * @return array{language: string}
     */
    public function jsonSerialize(): array
    {
        return ['language' => $this->language];
    }
}

final class User
{
    public function __construct(
        public ?string $id,
        public ?string $code,
        public ?string $name,
        public ?string $hashedPassword,
        public ?string $type,
    ) {
    }

    /**
     * @param array{id?: string, code?: string, name?: string, hashed_password?: string, type?: string} $dbRow
     */
    public static function fromDbRow(array $dbRow): self
    {
        return new self(
            $dbRow['id'] ?? null,
            $dbRow['code'] ?? null,
            $dbRow['name'] ?? null,
            $dbRow['hashed_password'] ?? null,
            $dbRow['type'] ?? null,
        );
    }
}

final class Course
{
    public function __construct(
        public ?string $id,
        public ?string $code,
        public ?string $type,
        public ?string $name,
        public ?string $description,
        public ?int $credit,
        public ?int $period,
        public ?string $dayOfWeek,
        public ?string $teacherId,
        public ?string $keywords,
        public ?string $status,
    ) {
    }

    /**
     * @param array{id?: string, code?: string, type?: string, name?: string, description?: string, credit?: int, period?: int, day_of_week?: string, teacher_id?: string, keywords?: string, status?: string} $dbRow
     */
    public static function fromDbRow(array $dbRow): self
    {
        return new self(
            $dbRow['id'] ?? null,
            $dbRow['code'] ?? null,
            $dbRow['type'] ?? null,
            $dbRow['name'] ?? null,
            $dbRow['description'] ?? null,
            $dbRow['credit'] ?? null,
            $dbRow['period'] ?? null,
            $dbRow['day_of_week'] ?? null,
            $dbRow['teacher_id'] ?? null,
            $dbRow['keywords'] ?? null,
            $dbRow['status'] ?? null,
        );
    }
}

// ---------- Public API ----------

final class LoginRequest
{
    public function __construct(
        public string $code,
        public string $password,
    ) {
    }

    /**
     * @throws UnexpectedValueException
     */
    public static function fromJson(string $json): self
    {
        try {
            $data = json_decode($json, true, flags: JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            throw new UnexpectedValueException();
        }

        if (!(isset($data['code']) && isset($data['password']))) {
            throw new UnexpectedValueException();
        }

        return new self($data['code'], $data['password']);
    }
}

// ---------- Users API ----------

final class GetMeResponse implements JsonSerializable
{
    public function __construct(
        public string $code,
        public string $name,
        public bool $isAdmin,
    ) {
    }

    /**
     * @return array{code: string, name: string, is_admin: bool}
     */
    public function jsonSerialize(): array
    {
        return [
            'code' => $this->code,
            'name' => $this->name,
            'is_admin' => $this->isAdmin,
        ];
    }
}

final class GetRegisteredCourseResponseContent implements JsonSerializable
{
    public function __construct(
        public string $id,
        public string $name,
        public string $teacher,
        public int $period,
        public string $dayOfWeek,
    ) {
    }

    /**
     * @return array{id: string, name: string, teacher: string, period: int, day_of_week: string}
     */
    public function jsonSerialize(): array
    {
        return [
            'id' => $this->id,
            'name' => $this->name,
            'teacher' => $this->teacher,
            'period' => $this->period,
            'day_of_week' => $this->dayOfWeek,
        ];
    }
}

final class RegisterCourseRequestContent
{
    public function __construct(public string $id)
    {
    }

    /**
     * @return array<self>
     * @throws UnexpectedValueException
     */
    public static function listFromJson(string $json): array
    {
        try {
            $data = json_decode($json, true, flags: JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            throw new UnexpectedValueException();
        }

        /** @var array<self> $list */
        $list = [];
        foreach ($data as $course) {
            if (!isset($course['id'])) {
                throw new UnexpectedValueException();
            }

            $list[] = new self($course['id']);
        }

        return $list;
    }
}

final class RegisterCoursesErrorResponse implements JsonSerializable
{
    /**
     * @param array<string> $courseNotFound
     * @param array<string> $notRegistrableStatus
     * @param array<string> $scheduleConflict
     */
    public function __construct(
        public array $courseNotFound = [],
        public array $notRegistrableStatus = [],
        public array $scheduleConflict = [],
    ) {
    }

    /**
     * @return array{course_not_found?: array<string>, not_registrable_status?: array<string>, schedule_conflict?: array<string>}
     */
    public function jsonSerialize(): array
    {
        $data = [];

        if (!empty($this->courseNotFound)) {
            $data['course_not_found'] = $this->courseNotFound;
        }

        if (!empty($this->notRegistrableStatus)) {
            $data['not_registrable_status'] = $this->notRegistrableStatus;
        }

        if (!empty($this->scheduleConflict)) {
            $data['schedule_conflict'] = $this->scheduleConflict;
        }

        return $data;
    }
}

final class Klass
{
    public function __construct(
        public ?string $id,
        public ?string $courseId,
        public ?int $part,
        public ?string $title,
        public ?string $description,
        public ?bool $submissionClosed,
    ) {
    }

    /**
     * @param array{id?: string, course_id?: string, part?: int, title?: string, description?: string, submission_closed?: int} $dbRow
     */
    public static function fromDbRow(array $dbRow): self
    {
        return new self(
            $dbRow['id'] ?? null,
            $dbRow['course_id'] ?? null,
            $dbRow['part'] ?? null,
            $dbRow['title'] ?? null,
            $dbRow['description'] ?? null,
            isset($dbRow['submission_closed']) ? (bool)$dbRow['submission_closed'] : null,
        );
    }
}

final class GetGradeResponse implements JsonSerializable
{
    /**
     * @param array<CourseResult> $courseResults
     */
    public function __construct(
        public Summary $summary,
        public array $courseResults,
    ) {
    }

    /**
     * @return array{summary: Summary, courses: array<CourseResult>}
     */
    public function jsonSerialize(): array
    {
        return [
            'summary' => $this->summary,
            'courses' => $this->courseResults,
        ];
    }
}

final class Summary implements JsonSerializable
{
    public function __construct(
        public int $credits,
        public float $gpa,
        public float $gpaTScore, // 偏差値
        public float $gpaAvg, // 平均値
        public float $gpaMax, // 最大値
        public float $gpaMin, // 最小値
    ) {
    }

    /**
     * @return array{credits: int, gpa: float, gpa_t_score: float, gpa_avg: float, gpa_max: float, gpa_min: float}
     */
    public function jsonSerialize(): array
    {
        return [
            'credits' => $this->credits,
            'gpa' => $this->gpa,
            'gpa_t_score' => $this->gpaTScore,
            'gpa_avg' => $this->gpaAvg,
            'gpa_max' => $this->gpaMax,
            'gpa_min' => $this->gpaMin,
        ];
    }
}

final class CourseResult implements JsonSerializable
{
    /**
     * @param array<ClassScore> $classScores
     */
    public function __construct(
        public string $name,
        public string $code,
        public int $totalScore,
        public float $totalScoreTScore,
        public float $totalScoreAvg,
        public int $totalScoreMax,
        public int $totalScoreMin,
        public array $classScores,
    ) {
    }

    /**
     * @return array{name: string, code: string, total_score: int, total_score_t_score: float, total_score_avg: float, total_score_max: int, total_score_min: int, class_scores: array<ClassScore>}
     */
    public function jsonSerialize(): array
    {
        return [
            'name' => $this->name,
            'code' => $this->code,
            'total_score' => $this->totalScore,
            'total_score_t_score' => $this->totalScoreTScore,
            'total_score_avg' => $this->totalScoreAvg,
            'total_score_max' => $this->totalScoreMax,
            'total_score_min' => $this->totalScoreMin,
            'class_scores' => $this->classScores,
        ];
    }
}

final class ClassScore implements JsonSerializable
{
    public function __construct(
        public string $classId,
        public string $title,
        public int $part,
        public ?int $score, // 0~100点
        public int $submitters, // 提出した学生数
    ) {
    }

    /**
     * @return array{class_id: string, title: string, part: int, score: ?int, submitters: int}
     */
    public function jsonSerialize(): array
    {
        return [
            'class_id' => $this->classId,
            'title' => $this->title,
            'part' => $this->part,
            'score' => $this->score,
            'submitters' => $this->submitters,
        ];
    }
}

// ---------- Courses API ----------

final class AddCourseRequest
{
    public function __construct(
        public string $code,
        public string $type,
        public string $name,
        public string $description,
        public int $credit,
        public int $period,
        public string $dayOfWeek,
        public string $keywords,
    ) {
    }

    /**
     * @throws UnexpectedValueException
     */
    public static function fromJson(string $json): self
    {
        try {
            $data = json_decode($json, true, flags: JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            throw new UnexpectedValueException();
        }

        if (
            !(
                isset($data['code']) &&
                isset($data['type']) &&
                isset($data['name']) &&
                isset($data['description']) &&
                isset($data['credit']) &&
                isset($data['period']) &&
                isset($data['day_of_week']) &&
                isset($data['keywords'])
            )
        ) {
            throw new UnexpectedValueException();
        }

        return new self(
            $data['code'],
            $data['type'],
            $data['name'],
            $data['description'],
            $data['credit'],
            $data['period'],
            $data['day_of_week'],
            $data['keywords'],
        );
    }
}

final class AddCourseResponse implements JsonSerializable
{
    public function __construct(public string $id)
    {
    }

    /**
     * @return array{id: string}
     */
    public function jsonSerialize(): array
    {
        return ['id' => $this->id];
    }
}

final class GetCourseDetailResponse implements JsonSerializable
{
    public function __construct(
        public ?string $id,
        public ?string $code,
        public ?string $type,
        public ?string $name,
        public ?string $description,
        public ?int $credit,
        public ?int $period,
        public ?string $dayOfWeek,
        public ?string $teacherId,
        public ?string $keywords,
        public ?string $status,
        public ?string $teacher,
    ) {
    }

    /**
     * @param array{id?: string, code?: string, type?: string, name?: string, description?: string, credit?: int, period?: int, day_of_week?: string, teacher_id?: string, keywords?: string, status?: string, teacher?: string} $dbRow
     */
    public static function fromDbRow(array $dbRow): self
    {
        return new self(
            $dbRow['id'] ?? null,
            $dbRow['code'] ?? null,
            $dbRow['type'] ?? null,
            $dbRow['name'] ?? null,
            $dbRow['description'] ?? null,
            $dbRow['credit'] ?? null,
            $dbRow['period'] ?? null,
            $dbRow['day_of_week'] ?? null,
            $dbRow['teacher_id'] ?? null,
            $dbRow['keywords'] ?? null,
            $dbRow['status'] ?? null,
            $dbRow['teacher'] ?? null,
        );
    }

    /**
     * @return array{id: string, code: string, type: string, name: string, description: string, credit: int, period: int, day_of_week: string, keywords: string, status: string, teacher: string}
     * @throws UnexpectedValueException
     */
    public function jsonSerialize(): array
    {
        $data = [
            'id' => $this->id,
            'code' => $this->code,
            'type' => $this->type,
            'name' => $this->name,
            'description' => $this->description,
            'credit' => $this->credit,
            'period' => $this->period,
            'day_of_week' => $this->dayOfWeek,
            'keywords' => $this->keywords,
            'status' => $this->status,
            'teacher' => $this->teacher,
        ];

        foreach ($data as $value) {
            if (is_null($value)) {
                throw new UnexpectedValueException();
            }
        }

        return $data;
    }
}

final class SetCourseStatusRequest
{
    public function __construct(public string $status)
    {
    }

    /**
     * @throws UnexpectedValueException
     */
    public static function fromJson(string $json): self
    {
        try {
            $data = json_decode($json, true, flags: JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            throw new UnexpectedValueException();
        }

        if (!isset($data['status'])) {
            throw new UnexpectedValueException();
        }

        return new self($data['status']);
    }
}

final class ClassWithSubmitted
{
    public function __construct(
        public ?string $id,
        public ?string $courseId,
        public ?int $part,
        public ?string $title,
        public ?string $description,
        public ?bool $submissionClosed,
        public ?bool $submitted,
    ) {
    }

    /**
     * @param array{id?: string, course_id?: string, part?: int, title?: string, description?: string, submission_closed?: int, submitted?: int} $dbRow
     */
    public static function fromDbRow(array $dbRow): self
    {
        return new self(
            $dbRow['id'] ?? null,
            $dbRow['course_id'] ?? null,
            $dbRow['part'] ?? null,
            $dbRow['title'] ?? null,
            $dbRow['description'] ?? null,
            isset($dbRow['submission_closed']) ? (bool)$dbRow['submission_closed'] : null,
            isset($dbRow['submitted']) ? (bool)$dbRow['submitted'] : null,
        );
    }
}

final class GetClassResponse implements JsonSerializable
{
    public function __construct(
        public string $id,
        public int $part,
        public string $title,
        public string $description,
        public bool $submissionClosed,
        public bool $submitted,
    ) {
    }

    /**
     * @return array{id: string, part: int, title: string, description: string, submission_closed: bool, submitted: bool}
     */
    public function jsonSerialize(): array
    {
        return [
            'id' => $this->id,
            'part' => $this->part,
            'title' => $this->title,
            'description' => $this->description,
            'submission_closed' => $this->submissionClosed,
            'submitted' => $this->submitted,
        ];
    }
}

final class AddClassRequest
{
    public function __construct(
        public int $part,
        public string $title,
        public string $description,
    ) {
    }

    /**
     * @throws UnexpectedValueException
     */
    public static function fromJson(string $json): self
    {
        try {
            $data = json_decode($json, true, flags: JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            throw new UnexpectedValueException();
        }

        if (
            !(
                isset($data['part']) &&
                isset($data['title']) &&
                isset($data['description'])
            )
        ) {
            throw new UnexpectedValueException();
        }

        return new self($data['part'], $data['title'], $data['description']);
    }
}

final class AddClassResponse implements JsonSerializable
{
    public function __construct(public string $classId)
    {
    }

    /**
     * @return array{class_id: string}
     */
    public function jsonSerialize(): array
    {
        return ['class_id' => $this->classId];
    }
}

final class Score
{
    public function __construct(
        public string $userCode,
        public int $score,
    ) {
    }

    /**
     * @return array<Score>
     * @throws UnexpectedValueException
     */
    public static function listFromJson(string $json): array
    {
        try {
            $data = json_decode($json, true, flags: JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            throw new UnexpectedValueException();
        }

        /** @var array<self> $list */
        $list = [];
        foreach ($data as $score) {
            if (!(isset($score['user_code']) && isset($score['score']))) {
                throw new UnexpectedValueException();
            }

            $list[] = new self($score['user_code'], $score['score']);
        }

        return $list;
    }
}

final class Submission
{
    public function __construct(
        public ?string $userId,
        public ?string $userCode,
        public ?string $fileName,
    ) {
    }

    /**
     * @param array{user_id?: string, user_code?: string, file_name?: string} $dbRow
     */
    public static function fromDbRow(array $dbRow): self
    {
        return new self(
            $dbRow['user_id'] ?? null,
            $dbRow['user_code'] ?? null,
            $dbRow['file_name'] ?? null,
        );
    }
}

// ---------- Announcement API ----------

final class AnnouncementWithoutDetail implements JsonSerializable
{
    public function __construct(
        public ?string $id,
        public ?string $courseId,
        public ?string $courseName,
        public ?string $title,
        public ?bool $unread,
    ) {
    }

    /**
     * @param array{id?: string, course_id?: string, course_name?: string, title?: string, unread?: int} $dbRow
     */
    public static function fromDbRow(array $dbRow): self
    {
        return new self(
            $dbRow['id'] ?? null,
            $dbRow['course_id'] ?? null,
            $dbRow['course_name'] ?? null,
            $dbRow['title'] ?? null,
            isset($dbRow['unread']) ? (bool)$dbRow['unread'] : null,
        );
    }

    /**
     * @return array{id: string, course_id: string, course_name: string, title: string, unread: bool}
     * @throws UnexpectedValueException
     */
    public function jsonSerialize(): array
    {
        $data = [
            'id' => $this->id,
            'course_id' => $this->courseId,
            'course_name' => $this->courseName,
            'title' => $this->title,
            'unread' => $this->unread,
        ];

        foreach ($data as $value) {
            if (is_null($value)) {
                throw new UnexpectedValueException();
            }
        }

        return $data;
    }
}

final class GetAnnouncementsResponse implements JsonSerializable
{
    /**
     * @param array<AnnouncementWithoutDetail> $announcements
     */
    public function __construct(
        public int $unreadCount,
        public array $announcements,
    ) {
    }

    /**
     * @return array{unread_count: int, announcements: array<AnnouncementWithoutDetail>}
     */
    public function jsonSerialize(): array
    {
        return [
            'unread_count' => $this->unreadCount,
            'announcements' => $this->announcements,
        ];
    }
}

final class Announcement
{
    public function __construct(
        public ?string $id,
        public ?string $courseId,
        public ?string $title,
        public ?string $message,
    ) {
    }

    /**
     * @param array{id?: string, course_id?: string, title?: string, message?: string} $dbRow
     */
    public static function fromDbRow(array $dbRow): self
    {
        return new self(
            $dbRow['id'] ?? null,
            $dbRow['course_id'] ?? null,
            $dbRow['title'] ?? null,
            $dbRow['message'] ?? null,
        );
    }
}

final class AddAnnouncementRequest
{
    public function __construct(
        public string $id,
        public string $courseId,
        public string $title,
        public string $message,
    ) {
    }

    /**
     * @throws UnexpectedValueException
     */
    public static function fromJson(string $json): self
    {
        try {
            $data = json_decode($json, true, flags: JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            throw new UnexpectedValueException();
        }

        if (
            !(
                isset($data['id']) &&
                isset($data['course_id']) &&
                isset($data['title']) &&
                isset($data['message'])
            )
        ) {
            throw new UnexpectedValueException();
        }

        return new self(
            $data['id'],
            $data['course_id'],
            $data['title'],
            $data['message'],
        );
    }
}

final class AnnouncementDetail implements JsonSerializable
{
    public function __construct(
        public ?string $id,
        public ?string $courseId,
        public ?string $courseName,
        public ?string $title,
        public ?string $message,
        public ?bool $unread,
    ) {
    }

    /**
     * @param array{id?: string, course_id?: string, course_name?: string, title?: string, message?: string, unread?: int} $dbRow
     */
    public static function fromDbRow(array $dbRow): self
    {
        return new self(
            $dbRow['id'] ?? null,
            $dbRow['course_id'] ?? null,
            $dbRow['course_name'] ?? null,
            $dbRow['title'] ?? null,
            $dbRow['message'] ?? null,
            isset($dbRow['unread']) ? (bool)$dbRow['unread'] : null,
        );
    }

    /**
     * @return array{id: string, course_id: string, course_name: string, title: string, message: string, unread: bool}
     * @throws UnexpectedValueException
     */
    public function jsonSerialize(): array
    {
        $data = [
            'id' => $this->id,
            'course_id' => $this->courseId,
            'course_name' => $this->courseName,
            'title' => $this->title,
            'message' => $this->message,
            'unread' => $this->unread,
        ];

        foreach ($data as $value) {
            if (is_null($value)) {
                throw new UnexpectedValueException();
            }
        }

        return $data;
    }
}
