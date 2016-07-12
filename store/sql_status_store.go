// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package store

import (
	"database/sql"
	"github.com/mattermost/platform/model"
)

const (
	MISSING_STATUS_ERROR = "store.sql_status.get.missing.app_error"
)

type SqlStatusStore struct {
	*SqlStore
}

func NewSqlStatusStore(sqlStore *SqlStore) StatusStore {
	s := &SqlStatusStore{sqlStore}

	for _, db := range sqlStore.GetAllConns() {
		table := db.AddTableWithName(model.Status{}, "Status").SetKeys(false, "UserId")
		table.ColMap("UserId").SetMaxSize(26)
		table.ColMap("Status").SetMaxSize(32)
	}

	return s
}

func (s SqlStatusStore) UpgradeSchemaIfNeeded() {
}

func (s SqlStatusStore) CreateIndexesIfNotExists() {
	s.CreateIndexIfNotExists("idx_status_user_id", "Status", "UserId")
	s.CreateIndexIfNotExists("idx_status_status", "Status", "Status")
}

func (s SqlStatusStore) SaveOrUpdate(status *model.Status) StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		if err := s.GetReplica().SelectOne(&model.Status{}, "SELECT * FROM Status WHERE UserId = :UserId", map[string]interface{}{"UserId": status.UserId}); err == nil {
			if _, err := s.GetMaster().Update(status); err != nil {
				result.Err = model.NewLocAppError("SqlStatusStore.SaveOrUpdate", "store.sql_status.update.app_error", nil, "")
			}
		} else {
			if err := s.GetMaster().Insert(status); err != nil {
				result.Err = model.NewLocAppError("SqlStatusStore.SaveOrUpdate", "store.sql_status.save.app_error", nil, "")
			}
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (s SqlStatusStore) Get(userId string) StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		var status model.Status

		if err := s.GetReplica().SelectOne(&status,
			`SELECT
				*
			FROM
				Status
			WHERE
				UserId = :UserId`, map[string]interface{}{"UserId": userId}); err != nil {
			if err == sql.ErrNoRows {
				result.Err = model.NewLocAppError("SqlStatusStore.Get", MISSING_STATUS_ERROR, nil, err.Error())
			} else {
				result.Err = model.NewLocAppError("SqlStatusStore.Get", "store.sql_status.get.app_error", nil, err.Error())
			}
		} else {
			result.Data = &status
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (s SqlStatusStore) GetOnlineAway() StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		var statuses []*model.Status
		if _, err := s.GetReplica().Select(&statuses, "SELECT * FROM Status WHERE Status = 'online' OR Status = 'away'"); err != nil {
			result.Err = model.NewLocAppError("SqlStatusStore.GetOnline", "store.sql_status.get_online.app_error", nil, err.Error())
		} else {
			result.Data = statuses
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (s SqlStatusStore) ResetAll() StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		if _, err := s.GetMaster().Exec("UPDATE Status SET Status = :Status", map[string]interface{}{"Status": model.STATUS_OFFLINE}); err != nil {
			result.Err = model.NewLocAppError("SqlStatusStore.ResetAll", "store.sql_status.reset_all.app_error", nil, "")
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}
