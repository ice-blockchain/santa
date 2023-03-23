-- SPDX-License-Identifier: ice License 1.0
--************************************************************************************************************************************
-- badge_progress
box.execute([[CREATE TABLE IF NOT EXISTS badge_progress (
                        achieved_badges  STRING,
                        balance          STRING,
                        user_id          STRING NOT NULL PRIMARY KEY,
                        friends_invited  UNSIGNED NOT NULL DEFAULT 0,
                        completed_levels UNSIGNED NOT NULL DEFAULT 0,
                        hide_badges      BOOLEAN
                    )
                     WITH ENGINE = 'memtx';]])
--************************************************************************************************************************************
-- badge_statistics
box.execute([[CREATE TABLE IF NOT EXISTS badge_statistics (
                     badge_type         STRING NOT NULL PRIMARY KEY,
                     badge_group_type   STRING NOT NULL,
                     achieved_by        UNSIGNED NOT NULL DEFAULT 0
                 )
                  WITH ENGINE = 'memtx';]])
box.execute([[CREATE INDEX IF NOT EXISTS badge_statistics_badge_group_type_ix ON badge_statistics (badge_group_type);]])
box.execute([[INSERT INTO badge_statistics (badge_group_type,badge_type)
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
                                    ]])
