// SPDX-License-Identifier: BUSL-1.1

package badges

import (
	"context"
	"encoding/json"
	"time"

	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
	appCfg "github.com/ice-blockchain/wintr/config"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/pkg/errors"
)

func newRepository(db tarantool.Connector, mb messagebroker.Client) Repository {
	appCfg.MustLoadFromKey("achievements", &cfg)

	return &repository{
		db:                         db,
		mb:                         mb,
		publishAchievedBadgesTopic: cfg.MessageBroker.Topics[1].Name,
	}
}

func (r *repository) AchieveBadge(ctx context.Context, userID UserID, badgeName BadgeName) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "achieve badge failed because context failed")
	}
	now := uint64(time.Now().UTC().UnixNano())
	sql := `INSERT INTO achieved_user_badges(USER_ID, badge_name,   ACHIEVED_AT)
                                                   VALUES(:userID,  :badgeName,  :achievedAt);`
	params := map[string]interface{}{
		"userID":     userID,
		"badgeName":  badgeName,
		"achievedAt": now,
	}
	query, err := r.db.PrepareExecute(sql, params)
	if err = storage.CheckSQLDMLErr(query, err); err != nil {
		return errors.Wrapf(err, "failed to achieve user's level for userID:%v", userID)
	}

	return errors.Wrapf(r.sendAchievedBadge(ctx, userID, badgeName, now), "failed to send achieved badge %v to message broker for userId:%v", badgeName, userID)
}

func (r *repository) sendAchievedBadge(ctx context.Context, userID UserID, badgeName BadgeName, achievedTime uint64) error {
	m := AchievedBadgeMessage{
		Name:       badgeName,
		UserID:     userID,
		AchievedAt: achievedTime,
	}

	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrapf(err, "[achieve-badge] failed to marshal %#v", m)
	}

	responder := make(chan error, 1)
	r.mb.SendMessage(ctx, &messagebroker.Message{
		Headers: map[string]string{"producer": "santa"},
		Key:     userID + badgeName,
		Topic:   r.publishAchievedBadgesTopic,
		Value:   b,
	}, responder)

	return errors.Wrapf(<-responder, "[achieve-badge] failed to send message to broker")
}

// nolint:funlen // Long SQL here
func (r *repository) AchieveBadgesWithCompletedRequirements(ctx context.Context, userProgress *progress.UserProgress) error {
	// SQL to read all unachieved badges for user (by type) based on current userProgress state (provided in params)
	// and to insert them for that user in one query. And then we fetch inserted rows based on ACHIEVED_AT = :achievedAt.
	sql := `
INSERT INTO ACHIEVED_USER_BADGES (USER_ID, BADGE_NAME, ACHIEVED_AT) 
SELECT :userID, badge_names.*, :achievedAt FROM (SELECT SOCIAL_BADGES.NAME from BADGES SOCIAL_BADGES
    WHERE SOCIAL_BADGES.TYPE = 'SOCIAL'
    and :referrals >= SOCIAL_BADGES.FROM_INCLUSIVE
    and :referrals <= SOCIAL_BADGES.TO_INCLUSIVE
UNION ALL SELECT ICE_BADGES.NAME from BADGES ICE_BADGES
    WHERE ICE_BADGES.TYPE = 'ICE'
    and :balance >= ICE_BADGES.FROM_INCLUSIVE
    and :balance <= ICE_BADGES.TO_INCLUSIVE
UNION SELECT LEVEL_BADGES.NAME from  BADGES LEVEL_BADGES
    WHERE LEVEL_BADGES.TYPE = 'LEVEL'
    and (SELECT count(*) from achieved_user_levels where USER_ID = :userID) >= LEVEL_BADGES.FROM_INCLUSIVE
    and (SELECT count(*) from achieved_user_levels where USER_ID = :userID) <= LEVEL_BADGES.TO_INCLUSIVE) badge_names
left join ACHIEVED_USER_BADGES on USER_ID = :userID and BADGE_NAME = badge_names.NAME
where ACHIEVED_USER_BADGES.BADGE_NAME IS NULL;
`
	now := uint64(time.Now().UTC().UnixNano())
	params := map[string]interface{}{
		"referrals":  userProgress.T1Referrals,
		"balance":    userProgress.Balance,
		"userID":     userProgress.UserID,
		"achievedAt": now,
	}
	err := storage.CheckSQLDMLErr(r.db.PrepareExecute(sql, params))
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return errors.Wrapf(err, "failed to achieve user's completed badges for userID:%v", userProgress.UserID)
	} else if err != nil && errors.Is(err, storage.ErrNotFound) {
		// Nothing to insert, no new badges.
		return nil
	}
	achievedBadges := []*achievedBadge{}
	queryParams := map[string]interface{}{
		"userID":     userProgress.UserID,
		"achievedAt": now,
	}
	err = r.db.PrepareExecuteTyped(`SELECT * FROM ACHIEVED_USER_BADGES WHERE ACHIEVED_AT = :achievedAt AND USER_ID = :userID;`, queryParams, &achievedBadges)
	if err != nil {
		return errors.Wrapf(err, "failed to achieve user's completed badges for userID:%v", userProgress.UserID)
	}
	for _, achievedBadgeByUser := range achievedBadges {
		if err := r.sendAchievedBadge(ctx, userProgress.UserID, achievedBadgeByUser.BadgeName, now); err != nil {
			return errors.Wrapf(err, "failed to send achieved badge %v to message broker for userId:%v", achievedBadgeByUser.BadgeName, userProgress.UserID)
		}
	}

	return nil
}
