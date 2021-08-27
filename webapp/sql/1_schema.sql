-- CREATEと逆順
DROP TABLE IF EXISTS `unread_announcements`;
DROP TABLE IF EXISTS `announcements`;
DROP TABLE IF EXISTS `submissions`;
DROP TABLE IF EXISTS `classes`;
DROP TABLE IF EXISTS `registrations`;
DROP TABLE IF EXISTS `courses`;
DROP TABLE IF EXISTS `users`;

-- master data
CREATE TABLE `users`
(
    `id`              CHAR(36) PRIMARY KEY,
    `code`            CHAR(6) UNIQUE              NOT NULL,
    `name`            VARCHAR(255)                NOT NULL,
    `hashed_password` VARCHAR(255)                NOT NULL,
    `type`            ENUM ('student', 'teacher') NOT NULL
);

CREATE TABLE `courses`
(
    `id`          CHAR(36) PRIMARY KEY,
    `code`        VARCHAR(255) UNIQUE                                                                 NOT NULL,
    `type`        ENUM ('liberal-arts', 'major-subjects')                                             NOT NULL,
    `name`        VARCHAR(255)                                                                        NOT NULL,
    `description` TEXT                                                                                NOT NULL,
    `credit`      TINYINT UNSIGNED                                                                    NOT NULL,
    `period`      TINYINT UNSIGNED                                                                    NOT NULL,
    `day_of_week` ENUM ('sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday') NOT NULL,
    `teacher_id`  CHAR(36)                                                                            NOT NULL,
    `keywords`    TEXT                                                                                NOT NULL,
    `status`      ENUM ('registration', 'in-progress', 'closed')                                      NOT NULL DEFAULT 'registration',
    CONSTRAINT FK_courses_teacher_id FOREIGN KEY (`teacher_id`) REFERENCES `users` (`id`)
);

CREATE TABLE `registrations`
(
    `course_id` CHAR(36),
    `user_id`   CHAR(36),
    PRIMARY KEY (`course_id`, `user_id`),
    CONSTRAINT FK_registrations_course_id FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`),
    CONSTRAINT FK_registrations_user_id FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
);

CREATE TABLE `classes`
(
    `id`                CHAR(36) PRIMARY KEY,
    `course_id`         CHAR(36)         NOT NULL,
    `part`              TINYINT UNSIGNED NOT NULL,
    `title`             VARCHAR(255)     NOT NULL,
    `description`       TEXT             NOT NULL,
    `submission_closed` TINYINT(1)       NOT NULL DEFAULT false,
    CONSTRAINT FK_classes_course_id FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`)
);

CREATE TABLE `submissions`
(
    `user_id`   CHAR(36)     NOT NULL,
    `class_id`  CHAR(36)     NOT NULL,
    `file_name` VARCHAR(255) NOT NULL,
    `score`     TINYINT UNSIGNED,
    PRIMARY KEY (`user_id`, `class_id`),
    CONSTRAINT FK_submissions_user_id FOREIGN KEY (`user_id`) REFERENCES `users` (`id`),
    CONSTRAINT FK_submissions_class_id FOREIGN KEY (`class_id`) REFERENCES `classes` (`id`)
);

CREATE TABLE `announcements`
(
    `id`         CHAR(36) PRIMARY KEY,
    `course_id`  CHAR(36)     NOT NULL,
    `title`      VARCHAR(255) NOT NULL,
    `message`    TEXT         NOT NULL,
    `created_at` DATETIME     NOT NULL,
    CONSTRAINT FK_announcements_course_id FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`)
);

CREATE TABLE `unread_announcements`
(
    `announcement_id` CHAR(36) NOT NULL,
    `user_id`         CHAR(36) NOT NULL,
    `deleted_at`      DATETIME,
    PRIMARY KEY (`announcement_id`, `user_id`),
    CONSTRAINT FK_unread_announcements_announcement_id FOREIGN KEY (`announcement_id`) REFERENCES `announcements` (`id`),
    CONSTRAINT FK_unread_announcements_user_id FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
);
