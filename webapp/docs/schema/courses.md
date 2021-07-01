# courses

## Description

科目一覧

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE TABLE `courses` (
  `id` char(36) COLLATE utf8mb4_bin NOT NULL,
  `code` varchar(255) COLLATE utf8mb4_bin NOT NULL,
  `type` enum('liberal-arts','major-subjects') COLLATE utf8mb4_bin NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_bin NOT NULL,
  `description` text COLLATE utf8mb4_bin NOT NULL,
  `credit` tinyint unsigned NOT NULL,
  `classroom` varchar(255) COLLATE utf8mb4_bin NOT NULL,
  `capacity` int unsigned DEFAULT NULL,
  `teacher_id` char(36) COLLATE utf8mb4_bin NOT NULL,
  `keywords` text COLLATE utf8mb4_bin NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `code` (`code`),
  KEY `FK_courses_teacher_id` (`teacher_id`),
  CONSTRAINT `FK_courses_teacher_id` FOREIGN KEY (`teacher_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin
```

</details>

## Columns

| Name        | Type                                  | Default | Nullable | Children                                                                                                                                                                                            | Parents           | Comment    |
| ----------- | ------------------------------------- | ------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------- | ---------- |
| id          | char(36)                              |         | false    | [announcements](announcements.md) [classes](classes.md) [course_requirements](course_requirements.md) [course_schedules](course_schedules.md) [grades](grades.md) [registrations](registrations.md) |                   |            |
| code        | varchar(255)                          |         | false    |                                                                                                                                                                                                     |                   |            |
| type        | enum('liberal-arts','major-subjects') |         | false    |                                                                                                                                                                                                     |                   |            |
| name        | varchar(255)                          |         | false    |                                                                                                                                                                                                     |                   | 科目名        |
| description | text                                  |         | false    |                                                                                                                                                                                                     |                   | 科目の説明      |
| credit      | tinyint unsigned                      |         | false    |                                                                                                                                                                                                     |                   | 単位数        |
| classroom   | varchar(255)                          |         | false    |                                                                                                                                                                                                     |                   | 開講場所       |
| capacity    | int unsigned                          |         | true     |                                                                                                                                                                                                     |                   | 履修定員       |
| teacher_id  | char(36)                              |         | false    |                                                                                                                                                                                                     | [users](users.md) |            |
| keywords    | text                                  |         | false    |                                                                                                                                                                                                     |                   |            |

## Constraints

| Name                  | Type        | Definition                                     |
| --------------------- | ----------- | ---------------------------------------------- |
| code                  | UNIQUE      | UNIQUE KEY code (code)                         |
| FK_courses_teacher_id | FOREIGN KEY | FOREIGN KEY (teacher_id) REFERENCES users (id) |
| PRIMARY               | PRIMARY KEY | PRIMARY KEY (id)                               |

## Indexes

| Name                  | Definition                                         |
| --------------------- | -------------------------------------------------- |
| FK_courses_teacher_id | KEY FK_courses_teacher_id (teacher_id) USING BTREE |
| PRIMARY               | PRIMARY KEY (id) USING BTREE                       |
| code                  | UNIQUE KEY code (code) USING BTREE                 |

## Relations

![er](courses.svg)

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
