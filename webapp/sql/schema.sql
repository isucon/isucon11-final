# CREATEと逆順
DROP TABLE IF EXISTS `unread_announcements`;
DROP TABLE IF EXISTS `announcements`;
DROP TABLE IF EXISTS `submissions`;
DROP TABLE IF EXISTS `assignments`;
DROP TABLE IF EXISTS `attendances`;
DROP TABLE IF EXISTS `documents`;
DROP TABLE IF EXISTS `classes`;
DROP TABLE IF EXISTS `grades`;
DROP TABLE IF EXISTS `registrations`;
DROP TABLE IF EXISTS `course_schedules`;
DROP TABLE IF EXISTS `schedules`;
DROP TABLE IF EXISTS `course_requirements`;
DROP TABLE IF EXISTS `courses`;
DROP TABLE IF EXISTS `users`;
DROP TABLE IF EXISTS `phase`;

CREATE TABLE `phase`
(
    `phase`    ENUM ('reg', 'class', 'result') DEFAULT 'reg' NOT NULL,
    `year`     INT UNSIGNED                                  NOT NULL,
    `semester` ENUM ('first', 'second')                      NOT NULL
);

-- master data
CREATE TABLE `users`
(
    `id`              CHAR(36) PRIMARY KEY,
    `name`            VARCHAR(255)                NOT NULL,
    `mail_address`    VARCHAR(255)                NOT NULL,
    `hashed_password` BINARY(60)                  NOT NULL,
    `type`            ENUM ('student', 'faculty') NOT NULL
);

-- master data
CREATE TABLE `courses`
(
    `id`          CHAR(36) PRIMARY KEY,
    `code`        VARCHAR(255) UNIQUE                     NOT NULL,
    `type`        ENUM ('liberal-arts', 'major-subjects') NOT NULL,
    `name`        VARCHAR(255)                            NOT NULL,
    `description` TEXT                                    NOT NULL,
    `credit`      TINYINT UNSIGNED                        NOT NULL,
    `classroom`   VARCHAR(255)                            NOT NULL,
    `capacity`    INT UNSIGNED,
    `teacher_id`  CHAR(36)                                NOT NULL,
    `keywords`    TEXT                                    NOT NULL,
    CONSTRAINT FK_courses_teacher_id FOREIGN KEY (`teacher_id`) REFERENCES `users` (`id`)
);

-- master data
CREATE TABLE `course_requirements`
(
    `course_id`          CHAR(36),
    `required_course_id` CHAR(36),
    PRIMARY KEY (`course_id`, `required_course_id`),
    CONSTRAINT FK_course_requirements_course_id FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`),
    CONSTRAINT FK_course_requirements_required_course_id FOREIGN KEY (`required_course_id`) REFERENCES `courses` (`id`)
);

-- master data
CREATE TABLE `schedules`
(
    `id`          CHAR(36) PRIMARY KEY,
    `period`      TINYINT UNSIGNED                                                                    NOT NULL,
    `day_of_week` ENUM ('sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday') NOT NULL,
    `semester`    ENUM ('first', 'second')                                                            NOT NULL,
    `year`        INT UNSIGNED                                                                        NOT NULL
);

-- master data
CREATE TABLE `course_schedules`
(
    `course_id`   CHAR(36),
    `schedule_id` CHAR(36),
    PRIMARY KEY (`course_id`, `schedule_id`),
    CONSTRAINT FK_course_schedules_course_id FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`),
    CONSTRAINT FK_course_schedules_schedule_id FOREIGN KEY (`schedule_id`) REFERENCES `schedules` (`id`)
);

CREATE TABLE `registrations`
(
    `course_id`  CHAR(36),
    `user_id`    CHAR(36),
    `created_at` DATETIME(6) NOT NULL,
    `deleted_at` DATETIME(6),
    PRIMARY KEY (`course_id`, `user_id`),
    CONSTRAINT FK_registrations_course_id FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`),
    CONSTRAINT FK_registrations_user_id FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
);

CREATE TABLE `grades`
(
    `id`         CHAR(36) PRIMARY KEY,
    `user_id`    CHAR(36)     NOT NULL,
    `course_id`  CHAR(36)     NOT NULL,
    `grade`      INT UNSIGNED NOT NULL,
    `created_at` DATETIME(6)  NOT NULL,
    CONSTRAINT FK_grades_user_id FOREIGN KEY (`user_id`) REFERENCES `users` (`id`),
    CONSTRAINT FK_grades_course_id FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`)
);

-- MEMO: クラス追加しないならmaster data
CREATE TABLE `classes`
(
    `id`              CHAR(36) PRIMARY KEY,
    `course_id`       CHAR(36)            NOT NULL,
    `part`            TINYINT UNSIGNED    NOT NULL,
    `title`           VARCHAR(255)        NOT NULL,
    `description`     TEXT                NOT NULL,
    `attendance_code` VARCHAR(255) UNIQUE NOT NULL,
    CONSTRAINT FK_classes_course_id FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`)
);

CREATE TABLE `documents`
(
    `id`         CHAR(36) PRIMARY KEY,
    `class_id`   CHAR(36)    NOT NULL,
    `name`       TEXT        NOT NULL,
    `created_at` DATETIME(6) NOT NULL,
    CONSTRAINT FK_documents_class_id FOREIGN KEY (`class_id`) REFERENCES `classes` (`id`)
);

CREATE TABLE `attendances`
(
    `class_id`   CHAR(36),
    `user_id`    CHAR(36),
    `created_at` DATETIME(6) NOT NULL,
    PRIMARY KEY (`class_id`, `user_id`),
    CONSTRAINT FK_attendances_class_id FOREIGN KEY (`class_id`) REFERENCES `classes` (`id`),
    CONSTRAINT FK_attendances_user_id FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
);

CREATE TABLE `assignments`
(
    `id`          CHAR(36) PRIMARY KEY,
    `class_id`    CHAR(36)     NOT NULL,
    `name`        VARCHAR(255) NOT NULL,
    `description` TEXT         NOT NULL,
    `created_at`  DATETIME(6)  NOT NULL,
    CONSTRAINT FK_assignments_class_id FOREIGN KEY (`class_id`) REFERENCES `classes` (`id`)
);

CREATE TABLE `submissions`
(
    `id`            CHAR(36) PRIMARY KEY,
    `user_id`       CHAR(36)     NOT NULL,
    `assignment_id` CHAR(36)     NOT NULL,
    `name`          VARCHAR(255) NOT NULL,
    `created_at`    DATETIME(6)  NOT NULL,
    CONSTRAINT FK_submissions_user_id FOREIGN KEY (`user_id`) REFERENCES `users` (`id`),
    CONSTRAINT FK_submissions_assignment_id FOREIGN KEY (`assignment_id`) REFERENCES `assignments` (`id`)
);

CREATE TABLE `announcements`
(
    `id`         CHAR(36) PRIMARY KEY,
    `course_id`  CHAR(36)     NOT NULL,
    `title`      VARCHAR(255) NOT NULL,
    `message`    TEXT         NOT NULL,
    `created_at` DATETIME(6)  NOT NULL,
    CONSTRAINT FK_announcements_course_id FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`)
);

CREATE TABLE `unread_announcements`
(
    `announcement_id` CHAR(36)    NOT NULL,
    `user_id`         CHAR(36)    NOT NULL,
    `created_at`      DATETIME(6) NOT NULL,
    `deleted_at`      DATETIME(6),
    PRIMARY KEY (`announcement_id`, `user_id`),
    CONSTRAINT FK_unread_announcements_announcement_id FOREIGN KEY (`announcement_id`) REFERENCES `announcements` (`id`),
    CONSTRAINT FK_unread_announcements_user_id FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
)
