-- SPDX-License-Identifier: ice License 1.0
--************************************************************************************************************************************
-- badge_progress
CREATE TABLE IF NOT EXISTS badge_progress (
                        friends_invited  BIGINT NOT NULL DEFAULT 0,
                        completed_levels BIGINT NOT NULL DEFAULT 0,
                        hide_badges      BOOLEAN,
                        achieved_badges  TEXT[],
                        user_id          TEXT NOT NULL PRIMARY KEY,
                        balance          TEXT
                    );
--************************************************************************************************************************************
-- badge_statistics
CREATE TABLE IF NOT EXISTS badge_statistics (
                        achieved_by        BIGINT NOT NULL DEFAULT 0,
                        badge_type         TEXT NOT NULL PRIMARY KEY,
                        badge_group_type   TEXT NOT NULL
                 );
CREATE INDEX IF NOT EXISTS badge_statistics_badge_group_type_ix ON badge_statistics (badge_group_type);
INSERT INTO badge_statistics (badge_group_type,badge_type)
                                    VALUES ('level','level'),
                                           ('level','l1'),
                                           ('level','l2'),
                                           ('level','l3'),
                                           ('level','l4'),
                                           ('level','l5'),
                                           ('level','l6'),
                                           ('coin','coin'),
                                           ('coin','c1'),
                                           ('coin','c2'),
                                           ('coin','c3'),
                                           ('coin','c4'),
                                           ('coin','c5'),
                                           ('coin','c6'),
                                           ('coin','c7'),
                                           ('coin','c8'),
                                           ('coin','c9'),
                                           ('coin','c10'),
                                           ('social','social'),
                                           ('social','s1'),
                                           ('social','s2'),
                                           ('social','s3'),
                                           ('social','s4'),
                                           ('social','s5'),
                                           ('social','s6'),
                                           ('social','s7'),
                                           ('social','s8'),
                                           ('social','s9'),
                                           ('social','s10')
ON CONFLICT DO NOTHING;
