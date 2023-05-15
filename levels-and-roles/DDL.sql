-- SPDX-License-Identifier: ice License 1.0
--************************************************************************************************************************************
-- levels_and_roles_progress
CREATE TABLE IF NOT EXISTS levels_and_roles_progress (
                        mining_streak               BIGINT NOT NULL DEFAULT 0,
                        pings_sent                  BIGINT NOT NULL DEFAULT 0,
                        agenda_contacts_joined      BIGINT NOT NULL DEFAULT 0,
                        friends_invited             BIGINT NOT NULL DEFAULT 0,
                        completed_tasks             BIGINT NOT NULL DEFAULT 0,
                        hide_level                  BOOLEAN DEFAULT false,
                        hide_role                   BOOLEAN DEFAULT false,
                        contact_user_ids            TEXT[],
                        enabled_roles               TEXT[],
                        completed_levels            TEXT[],
                        user_id                     TEXT NOT NULL PRIMARY KEY,
                        phone_number_hash           TEXT
                    );
CREATE INDEX IF NOT EXISTS levels_and_roles_progress_phone_number_hash_ix ON levels_and_roles_progress (phone_number_hash);
--************************************************************************************************************************************
-- pings
CREATE TABLE IF NOT EXISTS pings (
                        user_id    TEXT NOT NULL,
                        pinged_by  TEXT NOT NULL,
                        last_ping_cooldown_ended_at  TIMESTAMP NOT NULL,
                        PRIMARY KEY(user_id, pinged_by, last_ping_cooldown_ended_at)
                    );
