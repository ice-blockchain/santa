-- SPDX-License-Identifier: ice License 1.0
--************************************************************************************************************************************
-- levels_and_roles_progress
box.execute([[CREATE TABLE IF NOT EXISTS levels_and_roles_progress (
                        enabled_roles               STRING,
                        completed_levels            STRING,
                        user_id                     STRING NOT NULL PRIMARY KEY,
                        phone_number_hash           STRING,
                        mining_streak               UNSIGNED NOT NULL DEFAULT 0,
                        pings_sent                  UNSIGNED NOT NULL DEFAULT 0,
                        agenda_contacts_joined      UNSIGNED NOT NULL DEFAULT 0,
                        friends_invited             UNSIGNED NOT NULL DEFAULT 0,
                        completed_tasks             UNSIGNED NOT NULL DEFAULT 0,
                        hide_level                  BOOLEAN,
                        hide_role                   BOOLEAN
                    )
                     WITH ENGINE = 'memtx';]])
box.execute([[CREATE INDEX IF NOT EXISTS levels_and_roles_progress_phone_number_hash_ix ON levels_and_roles_progress (phone_number_hash);]])
--************************************************************************************************************************************
-- agenda_phone_number_hashes
box.execute([[CREATE TABLE IF NOT EXISTS agenda_phone_number_hashes (
                        user_id                     STRING NOT NULL,
                        agenda_phone_number_hash    STRING NOT NULL,
                        PRIMARY KEY(user_id, agenda_phone_number_hash)
                    )
                     WITH ENGINE = 'memtx';]])
--************************************************************************************************************************************
-- pings
box.execute([[CREATE TABLE IF NOT EXISTS pings (
                        user_id    STRING NOT NULL,
                        pinged_by  STRING NOT NULL,
                        PRIMARY KEY(user_id, pinged_by)
                    )
                     WITH ENGINE = 'memtx';]])


