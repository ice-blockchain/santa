-- SPDX-License-Identifier: ice License 1.0
--************************************************************************************************************************************
-- levels_and_roles_progress
CREATE TABLE IF NOT EXISTS levels_and_roles_progress (
                        mining_streak               BIGINT NOT NULL DEFAULT 0,
                        pings_sent                  BIGINT NOT NULL DEFAULT 0,
                        friends_invited             BIGINT NOT NULL DEFAULT 0,
                        completed_tasks             BIGINT NOT NULL DEFAULT 0,
                        hide_level                  BOOLEAN DEFAULT false,
                        hide_role                   BOOLEAN DEFAULT false,
                        agenda_contact_user_ids     TEXT[],
                        enabled_roles               TEXT[],
                        completed_levels            TEXT[],
                        user_id                     TEXT NOT NULL PRIMARY KEY,
                        phone_number_hash           TEXT
                    ) WITH (fillfactor = 70);
--************************************************************************************************************************************
-- pings
CREATE TABLE IF NOT EXISTS pings (
                        last_ping_cooldown_ended_at  TIMESTAMP NOT NULL,
                        user_id                      TEXT NOT NULL,
                        pinged_by                    TEXT NOT NULL,
                        PRIMARY KEY(user_id, pinged_by, last_ping_cooldown_ended_at)
                    );
CREATE INDEX IF NOT EXISTS pings_last_ping_cooldown_ended_at_ix ON pings (last_ping_cooldown_ended_at);