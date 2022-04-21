-- SPDX-License-Identifier: BUSL-1.1

box.execute([[CREATE TABLE IF NOT EXISTS total_users  (
                    key STRING primary key,
                    value UNSIGNED NOT NULL
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[CREATE TABLE IF NOT EXISTS badges  (
                    name STRING primary key,
                    type STRING NOT NULL,
                    from_inclusive UNSIGNED NOT NULL DEFAULT 0,
                    to_inclusive UNSIGNED NOT NULL DEFAULT 0
                    ) WITH ENGINE = 'vinyl';]])

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
                    user_id STRING primary key REFERENCES users(user_id) ON DELETE CASCADE,
                    badge_name STRING NOT NULL REFERENCES badges(name) ON DELETE CASCADE,
                    achieved_at UNSIGNED NOT NULL
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[CREATE TABLE IF NOT EXISTS achieved_user_tasks  (
                    user_id STRING primary key REFERENCES users(user_id) ON DELETE CASCADE,
                    task_name STRING NOT NULL REFERENCES tasks(name) ON DELETE CASCADE,
                    achieved_at UNSIGNED NOT NULL
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[CREATE TABLE IF NOT EXISTS consecutive_user_mining_sessions  (
                    user_id STRING primary key REFERENCES users(user_id) ON DELETE CASCADE,
                    last_mining_started_at UNSIGNED NOT NULL DEFAULT 0,
                    max_count UNSIGNED NOT NULL DEFAULT 0
                    ) WITH ENGINE = 'vinyl';]])

box.execute([[CREATE TABLE IF NOT EXISTS user_referral_pings  (
                    user_id STRING primary key REFERENCES users(user_id) ON DELETE CASCADE,
                    max_count UNSIGNED NOT NULL DEFAULT 0
                    ) WITH ENGINE = 'vinyl';]])

-- TODO: still need to find out how to do Achievement Dictionary>Levels>9-14 (from README.md)

-- TODO will add indexes later on