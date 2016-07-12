// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api

import (
	l4g "github.com/alecthomas/log4go"
	"github.com/mattermost/platform/model"
	"github.com/mattermost/platform/utils"
)

func InitStatus() {
	l4g.Debug(utils.T("api.status.init.debug"))
}

func SetStatusOnline(userId string, sessionId string) {
	broadcast := false

	var status *model.Status
	var err *model.AppError
	if status, err = GetStatus(userId); err != nil {
		status = &model.Status{userId, model.STATUS_ONLINE, model.GetMillis()}
		broadcast = true
	} else {
		if status.Status != model.STATUS_ONLINE {
			broadcast = true
		}
		status.Status = model.STATUS_ONLINE
		status.LastActivityAt = model.GetMillis()
	}

	statusCache.Add(status.UserId, status)

	achan := Srv.Store.Session().UpdateLastActivityAt(sessionId, model.GetMillis())
	schan := Srv.Store.Status().SaveOrUpdate(status)

	if result := <-achan; result.Err != nil {
		l4g.Error(utils.T("api.web_conn.new_web_conn.last_activity.error"), userId, sessionId, result.Err)
	}

	if result := <-schan; result.Err != nil {
		l4g.Error(utils.T("api.web_conn.new_web_conn.save_status.error"), userId, result.Err)
	}

	if broadcast {
		event := model.NewWebSocketEvent("", "", status.UserId, model.WEBSOCKET_EVENT_STATUS_CHANGE)
		event.Add("status", model.STATUS_ONLINE)
		go Publish(event)
	}
}

func SetStatusOffline(userId string) {
	status := &model.Status{userId, model.STATUS_OFFLINE, model.GetMillis()}

	statusCache.Add(status.UserId, status)

	if result := <-Srv.Store.Status().SaveOrUpdate(status); result.Err != nil {
		l4g.Error(utils.T("api.web_conn.new_web_conn.save_status.error"), userId, result.Err)
	}

	event := model.NewWebSocketEvent("", "", status.UserId, model.WEBSOCKET_EVENT_STATUS_CHANGE)
	event.Add("status", model.STATUS_OFFLINE)
	go Publish(event)
}

func SetStatusAwayIfNeeded(userId string) {
	status, err := GetStatus(userId)
	if err != nil {
		status = &model.Status{userId, model.STATUS_OFFLINE, 0}
	}

	if status.Status == model.STATUS_AWAY {
		return
	}

	if model.GetMillis()-status.LastActivityAt <= model.STATUS_AWAY_TIMEOUT {
		return
	}

	status.Status = model.STATUS_AWAY

	statusCache.Add(status.UserId, status)

	if result := <-Srv.Store.Status().SaveOrUpdate(status); result.Err != nil {
		l4g.Error(utils.T("api.web_conn.new_web_conn.save_status.error"), userId, result.Err)
	}

	event := model.NewWebSocketEvent("", "", status.UserId, model.WEBSOCKET_EVENT_STATUS_CHANGE)
	event.Add("status", model.STATUS_AWAY)
	go Publish(event)
}

func GetStatus(userId string) (*model.Status, *model.AppError) {
	if status, ok := statusCache.Get(userId); ok {
		return status.(*model.Status), nil
	}

	if result := <-Srv.Store.Status().Get(userId); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.(*model.Status), nil
	}
}
