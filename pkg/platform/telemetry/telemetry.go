package telemetry

import (
	"bytes"
	"net/http"
	"time"

	"go.keploy.io/server/pkg/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

type Telemetry struct {
	TelemetryDB DB
	Enabled     bool
	logger      *zap.Logger
}

func NewTelemetry(col DB, enabled bool, logger *zap.Logger) *Telemetry {
	adb := Telemetry{
		Enabled:     enabled,
		logger:      logger,
		TelemetryDB: col,
	}
	return &adb
}

func (ac *Telemetry) Ping() {
	if !ac.Enabled {
		return
	}
	go func() {
		for {
			count, err := ac.TelemetryDB.Count()
			if err != nil {
				ac.logger.Fatal("failed to countDocuments in analytics collection", zap.Error(err))
			}
			event := models.Event{
				EventType: "Ping",
				CreatedAt: time.Now().Unix(),
			}
			if count == 0 {
				bin, err := marshalEvent(event, ac.logger)
				if err != nil {
					break
				}
				resp, err := http.Post("http://localhost:3030/analytics", "application/json", bytes.NewBuffer(bin))
				if err != nil {
					ac.logger.Fatal("failed to send request for analytics", zap.Error(err))
					break
				}
				id, err := unmarshalResp(resp, ac.logger)
				if err != nil {
					break
				}
				ac.TelemetryDB.Insert(id)
			} else {
				ac.SendTelemetry("Ping")
			}
			time.Sleep(5 * time.Minute)
		}
	}()

}

func (ac *Telemetry) Normalize() {
	go func() {
		ac.SendTelemetry("NormaliseTC")
	}()
}

func (ac *Telemetry) DeleteTc() {
	go func() {
		ac.SendTelemetry("DeleteTC")
	}()
}

func (ac *Telemetry) EditTc() {
	go func() {
		ac.SendTelemetry("EditTC")
	}()
}

func (ac *Telemetry) Testrun(success int, failure int) {
	go func() {
		ac.SendTelemetry("TestRun", map[string]interface{}{"Passed-Tests": success, "Failed-Tests": failure})
	}()
}

func (ac *Telemetry) GetApps(apps int) {
	go func() {
		ac.SendTelemetry("GetApps", map[string]interface{}{"Apps": apps})
	}()
}

func (ac *Telemetry) SendTelemetry(eventType string, output ...map[string]interface{}) {
	if ac.Enabled {
		event := models.Event{
			EventType: eventType,
			CreatedAt: time.Now().Unix(),
		}
		// only 1 or no meta is passed in output array parameter
		if len(output) == 1 {
			event.Meta = output[0]
		}
		sr := ac.TelemetryDB.Find()
		if sr == nil {
			ac.logger.Error("")
			return
		}
		doc := bson.D{}
		err := sr.Decode(&doc)
		if err != nil {
			ac.logger.Error("failed to decode transactionID", zap.Error(err))
			return
		}
		m := doc.Map()
		tid, ok := m["InstallationID"].(string)
		if !ok {
			ac.logger.Error("InstallationID not present")
			return
		}
		event.InstallationID = tid
		bin, err := marshalEvent(event, ac.logger)
		if err != nil {
			return
		}
		_, err = http.Post("http://localhost:3030/analytics", "application/json", bytes.NewBuffer(bin))
		if err != nil {
			ac.logger.Fatal("failed to send request for analytics", zap.Error(err))
			return
		}
	}
}