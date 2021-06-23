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
    `phase` ENUM ('registration', 'term-time', 'exam-period') DEFAULT 'registration'
);

CREATE TABLE `users`
(
    `id`              CHAR(36) PRIMARY KEY,
    `name`            VARCHAR(255)                NOT NULL,
    `mail_address`    VARCHAR(255)                NOT NULL,
    `hashed_password` BINARY(60)                  NOT NULL,
    `type`            ENUM ('student', 'faculty') NOT NULL
);

CREATE TABLE `courses`
(
    `id`          CHAR(36) PRIMARY KEY,
    `name`        VARCHAR(255)     NOT NULL,
    `description` TEXT             NOT NULL,
    `credit`      TINYINT UNSIGNED NOT NULL,
    `classroom`   VARCHAR(255)     NOT NULL,
    `capacity`    INT UNSIGNED
);

CREATE TABLE `course_requirements`
(
    `course_id`          CHAR(36),
    `required_course_id` CHAR(36),
    PRIMARY KEY (`course_id`, `required_course_id`),
    FOREIGN KEY FK_course_id (`course_id`) REFERENCES `courses` (`id`),
    FOREIGN KEY FK_required_course_id (`required_course_id`) REFERENCES `courses` (`id`)
);

CREATE TABLE `schedules`
(
    `id`          CHAR(36) PRIMARY KEY,
    `period`      TINYINT UNSIGNED                                                                    NOT NULL,
    `day_of_week` ENUM ('sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday') NOT NULL,
    `semester`    ENUM ('first', 'second')                                                            NOT NULL,
    `year`        INT UNSIGNED                                                                        NOT NULL
);

CREATE TABLE `course_schedules`
(
    `course_id`   CHAR(36),
    `schedule_id` CHAR(36),
    PRIMARY KEY (`course_id`, `schedule_id`),
    FOREIGN KEY FK_course_id (`course_id`) REFERENCES `courses` (`id`),
    FOREIGN KEY FK_schedule_id (`schedule_id`) REFERENCES `schedules` (`id`)
);

CREATE TABLE `registrations`
(
    `course_id`  CHAR(36),
    `user_id`    CHAR(36),
    `created_at` DATETIME(6) NOT NULL,
    `deleted_at`  DATETIME(6),
    PRIMARY KEY (`course_id`, `user_id`),
    FOREIGN KEY FK_course_id (`course_id`) REFERENCES `courses` (`id`),
    FOREIGN KEY FK_user_id (`user_id`) REFERENCES `users` (`id`)
);

CREATE TABLE `grades`
(
    `id`        CHAR(36) PRIMARY KEY,
    `user_id`   CHAR(36)     NOT NULL,
    `course_id` CHAR(36)     NOT NULL,
    `grade`     INT UNSIGNED NOT NULL,
    FOREIGN KEY FK_user_id (`user_id`) REFERENCES `users` (`id`),
    FOREIGN KEY FK_course_id (`course_id`) REFERENCES `courses` (`id`)
);

CREATE TABLE `classes`
(
    `id`              CHAR(36) PRIMARY KEY,
    `course_id`       CHAR(36)     NOT NULL,
    `title`           VARCHAR(255) NOT NULL,
    `description`     TEXT         NOT NULL,
    `attendance_code` VARCHAR(255) NOT NULL,
    FOREIGN KEY FK_course_id (`course_id`) REFERENCES `courses` (`id`)
);

CREATE TABLE `documents`
(
    `id`         CHAR(36) PRIMARY KEY,
    `class_id`   CHAR(36)    NOT NULL,
    `name`       TEXT        NOT NULL,
    `created_at` DATETIME(6) NOT NULL,
    FOREIGN KEY FK_class_id (`class_id`) REFERENCES `classes` (`id`)
);

CREATE TABLE `attendances`
(
    `class_id`   CHAR(36),
    `user_id`    CHAR(36),
    `created_at` DATETIME(6) NOT NULL,
    PRIMARY KEY (`class_id`, `user_id`),
    FOREIGN KEY FK_class_id (`class_id`) REFERENCES `classes` (`id`),
    FOREIGN KEY FK_user_id (`user_id`) REFERENCES `users` (`id`)
);

CREATE TABLE `assignments`
(
    `id`          CHAR(36) PRIMARY KEY,
    `class_id`    CHAR(36)     NOT NULL,
    `name`        VARCHAR(255) NOT NULL,
    `description` TEXT         NOT NULL,
    `deadline`    DATETIME(6)  NOT NULL,
    `created_at`  DATETIME(6)  NOT NULL,
    FOREIGN KEY FK_class_id (`class_id`) REFERENCES `classes` (`id`)
);

CREATE TABLE `submissions`
(
    `id`            CHAR(36) PRIMARY KEY,
    `user_id`       CHAR(36)     NOT NULL,
    `assignment_id` CHAR(36)     NOT NULL,
    `name`          VARCHAR(255) NOT NULL,
    `created_at`    DATETIME(6)  NOT NULL,
    FOREIGN KEY FK_user_id (`user_id`) REFERENCES `users` (`id`),
    FOREIGN KEY FK_assignment_id (`assignment_id`) REFERENCES `assignments` (`id`)
);

CREATE TABLE `announcements`
(
    `id`         CHAR(36) PRIMARY KEY,
    `course_id`  CHAR(36)     NOT NULL,
    `title`      VARCHAR(255) NOT NULL,
    `message`    TEXT         NOT NULL,
    `created_at` DATETIME(6)  NOT NULL,
    FOREIGN KEY FK_course_id (`course_id`) REFERENCES `courses` (`id`)
);

CREATE TABLE `unread_announcements`
(
    `announcement_id` CHAR(36)    NOT NULL,
    `user_id`         CHAR(36)    NOT NULL,
    `created_at`      DATETIME(6) NOT NULL,
    `deleted_at`       DATETIME(6),
    PRIMARY KEY (`announcement_id`, `user_id`),
    FOREIGN KEY FK_announcement_id (`announcement_id`) REFERENCES `announcements` (`id`),
    FOREIGN KEY FK_user_id (`user_id`) REFERENCES `users` (`id`)
)
