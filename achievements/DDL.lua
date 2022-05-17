-- SPDX-License-Identifier: BUSL-1.1

box.execute([[CREATE TABLE IF NOT EXISTS global  (
                    key STRING primary key,
                    value SCALAR NOT NULL
                    ) WITH ENGINE = 'vinyl';]])

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
box.execute([[CREATE TABLE IF NOT EXISTS tasks  (
                    name STRING primary key,
                    index UNSIGNED NOT NULL DEFAULT 0
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[CREATE TABLE IF NOT EXISTS user_achievements  (
                    user_id STRING primary key,
                    balance UNSIGNED NOT NULL DEFAULT 0,
                    level UNSIGNED NOT NULL DEFAULT 1,
                    role STRING NOT NULL DEFAULT 'PIONEER',
                    t1_referrals UNSIGNED NOT NULL DEFAULT 0
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[CREATE TABLE IF NOT EXISTS achieved_user_badges  (
                    user_id STRING primary key REFERENCES user_achievements(user_id) ON DELETE CASCADE,
                    badge_name STRING NOT NULL REFERENCES badges(name) ON DELETE CASCADE,
                    achieved_at UNSIGNED NOT NULL
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[CREATE TABLE IF NOT EXISTS achieved_user_tasks  (
                    user_id STRING primary key REFERENCES user_achievements(user_id) ON DELETE CASCADE,
                    task_name STRING NOT NULL REFERENCES tasks(name) ON DELETE CASCADE,
                    achieved_at UNSIGNED NOT NULL
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[CREATE TABLE IF NOT EXISTS consecutive_user_mining_sessions  (
                    user_id STRING primary key REFERENCES user_achievements(user_id) ON DELETE CASCADE,
                    last_mining_started_at UNSIGNED NOT NULL DEFAULT 0,
                    max_count UNSIGNED NOT NULL DEFAULT 0
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[CREATE TABLE IF NOT EXISTS user_referral_pings  (
                    user_id STRING primary key REFERENCES user_achievements(user_id) ON DELETE CASCADE,
                    max_count UNSIGNED NOT NULL DEFAULT 0
                    ) WITH ENGINE = 'vinyl';]])

-- TODO: still need to find out how to do Achievement Dictionary>Levels>9-14 (from README.md)

-- TODO will add indexes later on