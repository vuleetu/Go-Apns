package apns

import (
	"encoding/json"
)

type Alert struct {
	Body          string   `json:"body,omitempty"`
	LockKey       string   `json:"loc-key,omitempty"`
	LockArgs      []string `json:"loc-args,omitempty"`
	ActionLockKey string   `json:"action-loc-key,omitempty"`
	LaunchImage   string   `json:"launch-image,omitempty"`
}

// If AlertStruct set to any instance, it will ignore any content in Alert when send to iOS.
// To use simple string Alert, make sure AlertStruct's value is nil.
type Aps struct {
	Alert       string `json:"alert,omitempty"`
	AlertStruct *Alert `json:"alert,omitempty"`
	Badge       int    `json:"badge,omitempty"`
	Sound       string `json:"sound,omitempty"`
}

type msgAps struct {
	Alert       string `json:"alert,omitempty"`
	Badge       int    `json:"badge,omitempty"`
	Sound       string `json:"sound,omitempty"`
}

type alertAps struct {
	AlertStruct *Alert `json:"alert,omitempty"`
	Badge       int    `json:"badge,omitempty"`
	Sound       string `json:"sound,omitempty"`
}

type Payload struct {
	Aps Aps

	customProperty map[string]interface{}
}

// Set a custom key with value, overwriting any existed key. If key is "aps", do nothing.
func (l *Payload) SetCustom(key string, value interface{}) {
	if key == "aps" {
		return
	}
	if l.customProperty == nil {
		l.customProperty = make(map[string]interface{})
	}
	l.customProperty[key] = value
}

// Get a custom key's value. If key is "aps", return nil.
func (l *Payload) GetCustom(key string) interface{} {
	if key == "aps" || l.customProperty == nil {
		return nil
	}
	return l.customProperty[key]
}

func (l Payload) MarshalJSON() ([]byte, error) {
	if l.customProperty == nil {
		l.customProperty = make(map[string]interface{})
	}

    var finalAps interface{}
	if l.Aps.AlertStruct != nil {
        m := alertAps{}
        m.AlertStruct = l.Aps.AlertStruct
        m.Badge = l.Aps.Badge
        m.Sound = l.Aps.Sound
        finalAps = m
	} else {
        m := msgAps{}
        m.Alert = l.Aps.Alert
        m.Badge = l.Aps.Badge
        m.Sound = l.Aps.Sound
        finalAps = m
    }

	l.customProperty["aps"] = finalAps
	return json.Marshal(l.customProperty)
}
