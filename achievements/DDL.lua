-- SPDX-License-Identifier: BUSL-1.1

box.execute([[CREATE TABLE IF NOT EXISTS global  (
                    key STRING primary key,
                    value SCALAR NOT NULL
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[CREATE TABLE IF NOT EXISTS user_progress  (
                    user_id STRING primary key,
                    balance UNSIGNED NOT NULL DEFAULT 0,
                    t1_referrals UNSIGNED NOT NULL DEFAULT 0,
                    agenda_phone_number_hashes STRING,
                    last_mining_started_at UNSIGNED NOT NULL DEFAULT 0,
                    max_consecutive_user_mining_sessions UNSIGNED NOT NULL DEFAULT 0,
                    total_user_referral_pings UNSIGNED NOT NULL DEFAULT 0
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[CREATE TABLE IF NOT EXISTS agenda_referrals  (
                    user_id STRING NOT NULL REFERENCES user_progress(user_id) ON DELETE CASCADE,
                    agenda_user_id STRING NOT NULL REFERENCES user_progress(user_id) ON DELETE CASCADE,
                    primary key(user_id,agenda_user_id) ) WITH ENGINE = 'vinyl';]])

-- BADGES
box.execute([[CREATE TABLE IF NOT EXISTS badges  (
                    name STRING primary key,
                    type STRING NOT NULL,
                    from_inclusive UNSIGNED NOT NULL DEFAULT 0,
                    to_inclusive UNSIGNED NOT NULL DEFAULT 0
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[INSERT INTO badges (name, type, from_inclusive, to_inclusive)
                          VALUES ('LEVEL1', 'LEVEL', 1, 1),
                                 ('LEVEL2', 'LEVEL', 2, 5),
                                 ('LEVEL3', 'LEVEL', 6, 10),
                                 ('LEVEL4', 'LEVEL', 11, 15),
                                 ('LEVEL5', 'LEVEL', 16, 20),
                                 ('LEVEL6', 'LEVEL', 21, 20000),
                                 ('ICE1', 'ICE', 0, 1000),
                                 ('ICE2', 'ICE', 1001, 5000),
                                 ('ICE3', 'ICE', 5001, 10000),
                                 ('ICE4', 'ICE', 10001, 40000),
                                 ('ICE5', 'ICE', 40001, 80000),
                                 ('ICE6', 'ICE', 80001, 160000),
                                 ('ICE7', 'ICE', 160001, 320000),
                                 ('ICE8', 'ICE', 320001, 640000),
                                 ('ICE9', 'ICE', 640001, 1280000),
                                 ('ICE10', 'ICE', 1280001, 1280000000000),
                                 ('SOCIAL1', 'SOCIAL', 0, 5),
                                 ('SOCIAL2', 'SOCIAL', 6, 15),
                                 ('SOCIAL3', 'SOCIAL', 16, 30),
                                 ('SOCIAL4', 'SOCIAL', 31, 100),
                                 ('SOCIAL5', 'SOCIAL', 101, 250),
                                 ('SOCIAL6', 'SOCIAL', 251, 500),
                                 ('SOCIAL7', 'SOCIAL', 501, 1000),
                                 ('SOCIAL8', 'SOCIAL', 1001, 2000),
                                 ('SOCIAL9', 'SOCIAL', 2001, 10000),
                                 ('SOCIAL10', 'SOCIAL', 10001, 1000000000)
          ]])

box.execute([[CREATE TABLE IF NOT EXISTS achieved_user_badges  (
                    user_id STRING NOT NULL REFERENCES user_progress(user_id) ON DELETE CASCADE,
                    badge_name STRING NOT NULL REFERENCES badges(name) ON DELETE CASCADE,
                    achieved_at UNSIGNED NOT NULL,
                    primary key(user_id, badge_name) ) WITH ENGINE = 'vinyl';]])

-- LEVELS
box.execute([[CREATE TABLE IF NOT EXISTS levels (name STRING primary key, description STRING NOT NULL) WITH ENGINE = 'vinyl';]])

box.execute([[INSERT INTO levels (name, description)
                           VALUES ('L1', '5 consecutive mining sessions ( no more than 10 hours pause between the mining sessions )'),
                                  ('L2', '10 consecutive mining sessions ( no more than 10 hours pause between the mining sessions )'),
                                  ('L3', '30 consecutive mining sessions ( no more than 10 hours pause between the mining sessions )'),
                                  ('L4', '60 consecutive mining sessions ( no more than 10 hours pause between the mining sessions )'),
                                  ('L5', '90 consecutive mining sessions ( no more than 10 hours pause between the mining sessions )'),
                                  ('L6', 'TASK1: Completed tasks from home screen'),
                                  ('L7', 'TASK2: Completed tasks from home screen'),
                                  ('L8', 'TASK3: Completed tasks from home screen'),
                                  ('L9', 'TASK4: Completed tasks from home screen'),
                                  ('L10', 'TASK5: Completed tasks from home screen'),
                                  ('L11', 'TASK6: Completed tasks from home screen'),
                                  ('L12', 'TASK7: Completed tasks from home screen'),
                                  ('L13', 'Confirm phone number'),
                                  ('L14', '1 Friend from agenda joined ICE'),
                                  ('L15', '5 Friends from agenda joined ICE'),
                                  ('L16', '10 Friends from agenda joined ICE'),
                                  ('L17', 'Opened the app 5 times'),
                                  ('L18', 'Opened the app 10 times'),
                                  ('L19', 'Opened the app more than 30 times in the last 30 days'),
                                  ('L20', 'First ping to referral'),
                                  ('L21', '10 pings to referrals')
           ]])

box.execute([[CREATE TABLE IF NOT EXISTS achieved_user_levels  (
                    user_id STRING NOT NULL REFERENCES user_progress(user_id) ON DELETE CASCADE,
                    level_name STRING NOT NULL REFERENCES levels(name) ON DELETE CASCADE,
                    achieved_at UNSIGNED NOT NULL,
                    primary key(user_id, level_name)) WITH ENGINE = 'vinyl';]])

box.execute([[CREATE TABLE IF NOT EXISTS current_user_levels  (
                    user_id STRING primary key REFERENCES user_progress(user_id) ON DELETE CASCADE,
                    level UNSIGNED NOT NULL DEFAULT 1,
                    updated_at UNSIGNED NOT NULL
                    ) WITH ENGINE = 'vinyl';]])

-- TASKS

box.execute([[CREATE TABLE IF NOT EXISTS tasks  (
                    name STRING primary key,
                    task_index UNSIGNED NOT NULL DEFAULT 0
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[INSERT INTO tasks (name, task_index)
                           VALUES ('TASK1', 1),
                                  ('TASK2', 2),
                                  ('TASK3', 3),
                                  ('TASK4', 4),
                                  ('TASK5', 5),
                                  ('TASK6', 6),
                                  ('TASK7', 7),
                                  ('PRIZE', 8)
           ]])

box.execute([[CREATE TABLE IF NOT EXISTS achieved_user_tasks  (
                    user_id STRING NOT NULL REFERENCES user_progress(user_id) ON DELETE CASCADE,
                    task_name STRING NOT NULL REFERENCES tasks(name) ON DELETE CASCADE,
                    achieved_at UNSIGNED NOT NULL,
                    primary key(user_id, task_name)) WITH ENGINE = 'vinyl';]])

-- ROLES

box.execute([[CREATE TABLE IF NOT EXISTS current_user_roles  (
                    user_id STRING primary key REFERENCES user_progress(user_id) ON DELETE CASCADE,
                    role_name STRING NOT NULL DEFAULT 'PIONEER' CHECK (role_name == 'PIONEER' OR role_name == 'AMBASSADOR'),
                    updated_at UNSIGNED NOT NULL
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[DROP VIEW IF EXISTS user_achievements;]])
box.execute([[CREATE VIEW IF NOT EXISTS user_achievements AS
            SELECT
                up.user_id,
                (
                    SELECT '[' || group_concat('{"name": "' || b.name ||
                                             '", "type": "' || b.type ||
                                             '", "fromInclusive": ' || cast(b.from_inclusive as string) ||
                                             ', "toInclusive": ' || cast(b.to_inclusive as string) ||
                                             ', "achievedAt": ' || cast(COALESCE(aub.achieved_at, 0) as string) || '}') || ']'
                    FROM badges AS b
                        LEFT JOIN achieved_user_badges AS aub
                            ON aub.badge_name = b.name
                            AND aub.user_id = up.user_id
                    ORDER BY b.type, b.from_inclusive
                ) as achieved_badges,
                (
                    SELECT '[' || group_concat('{"name": "' || t.name ||
                                            '","taskIndex": ' || cast(t.task_index as string) ||
                                            ',"achievedAt": ' || cast(coalesce(aut.achieved_at, 0) as string) || '}') || ']'
                    FROM tasks AS t
                        LEFT JOIN achieved_user_tasks aut
                            ON aut.task_name = t.name
                            AND aut.user_id = up.user_id
                    ORDER BY task_index
                ) as achieved_tasks,
                (
                    SELECT count(1)
                    FROM achieved_user_levels
                    WHERE user_id = up.user_id
                ) as levels,
                cur.role_name as user_role,
                (
                    SELECT '[' || group_concat('{"type": "' || type ||
                                            '","count": ' || CAST(CNT as string) || '}') || ']'
                    FROM (
                        SELECT
                            type, COUNT(*) as cnt
                        FROM
                            badges
                        GROUP BY type
                    )
                ) as badge_types
            FROM user_progress as up
                LEFT JOIN current_user_roles as cur ON cur.user_id = up.user_id;]])

-- TODO will add indexes later on
