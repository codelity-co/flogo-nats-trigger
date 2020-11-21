package nats

import (	

	"github.com/project-flogo/core/data/coerce"
)

// Settings struct
type Settings struct {
	NatsClusterUrls          string `md:"natsClusterUrls,required"`
	NatsConnName             string `md:"natsConnName"`
	PayloadFormat						 string `md:"payloadFormat"`
	NatsUserName             string `md:"natsUserName"`
	NatsUserPassword         string `md:"natsUserPassword"`
	NatsToken                string `md:"natsToken"`
	NatsNkeySeedfile         string `md:"natsNkeySeedfile"`
	NatsCredentialFile       string `md:"natsCredentialFile"`
	AutoReconnect            bool   `md:"autoReconnect"`
	MaxReconnects            int    `md:"maxReconnects"`
	EnableRandomReconnection bool   `md:"enableRandomReconnection"`
	ReconnectWait            int    `md:"reconnectWait"`
	ReconnectBufferSize      int  `md:"reconnectBufferSize"`
	SkipVerify               bool   `md:"skipVerify"`
	CaFile                   string `md:"caFile"`
	CertFile                 string `md:"certFile"`
	KeyFile                  string `md:"keyFile"`
	EnableStreaming          bool   `md:"enableStreaming"`
	StanClusterID            string `md:"stanClusterID"`
}

// FromMap method of Settings
func (s *Settings) FromMap(values map[string]interface{}) error {

	var (
		err error
	)
	s.NatsClusterUrls, err = coerce.ToString(values["natsClusterUrls"])
	if err != nil {
		return err
	}

	s.NatsConnName, err = coerce.ToString(values["natsConnName"])
	if err != nil {
		return err
	}

	s.PayloadFormat, err = coerce.ToString(values["payloadFormat"])
	if err != nil {
		return err
	}

	s.NatsUserName, err = coerce.ToString(values["natsUserName"])
	if err != nil {
		return err
	}

	s.NatsUserPassword, err = coerce.ToString(values["natsUserPassword"])
	if err != nil {
		return err
	}

	s.NatsToken, err = coerce.ToString(values["natsToken"])
	if err != nil {
		return err
	}

	s.NatsNkeySeedfile, err = coerce.ToString(values["natsNkeySeedfile"])
	if err != nil {
		return err
	}

	s.NatsCredentialFile, err = coerce.ToString(values["natsCredentialFile"])
	if err != nil {
		return err
	}

	s.AutoReconnect, err = coerce.ToBool(values["autoReconnect"])
	if err != nil {
		return err
	}

	s.MaxReconnects, err = coerce.ToInt(values["maxReconnects"])
	if err != nil {
		return err
	}

	s.EnableRandomReconnection, err = coerce.ToBool(values["enableRandomReconnection"])
	if err != nil {
		return err
	}

	s.ReconnectWait, err = coerce.ToInt(values["reconnectWait"])
	if err != nil {
		return err
	}

	s.ReconnectBufferSize, err = coerce.ToInt(values["reconnectBufferSize"])
	if err != nil {
		return err
	}

	s.SkipVerify, err = coerce.ToBool(values["skipVerify"])
	if err != nil {
		return err
	}

	s.CaFile, err = coerce.ToString(values["caFile"])
	if err != nil {
		return err
	}

	s.CertFile, err = coerce.ToString(values["certFile"])
	if err != nil {
		return err
	}

	s.KeyFile, err = coerce.ToString(values["keyFile"])
	if err != nil {
		return err
	}

	s.EnableStreaming, err = coerce.ToBool(values["enableStreaming"])
	if err != nil {
		return err
	}

	s.StanClusterID, err = coerce.ToString(values["stanClusterID"])
	if err != nil {
		return err
	}

	return nil

}

// ToMap method of Settings
func (s *Settings) ToMap() map[string]interface{} {

	return map[string]interface{}{
		"natsClusterUrls": s.NatsClusterUrls,
		"natsConnName":    s.NatsConnName,
		"payloadFormat":    s.PayloadFormat,
		"natsUserName": s.NatsUserName,
		"natsUserPassword": s.NatsUserPassword,
		"natsToken": s.NatsToken,
		"natsNkeySeedfile": s.NatsNkeySeedfile,
		"natsCredentialFile": s.NatsNkeySeedfile,
		"autoReconnect": s.AutoReconnect,
		"maxReconnects": s.MaxReconnects,
		"enableRandomReconnection": s.EnableRandomReconnection,
		"reconnectWait": s.ReconnectWait,
		"reconnectBufferSize": s.ReconnectBufferSize,
		"skipVerify": s.SkipVerify,
		"caFile": s.CaFile,
		"certFile": s.CertFile,
		"keyFile": s.KeyFile,
		"enableStreaming": s.EnableStreaming,
		"stanClusterID": s.StanClusterID,
	}

}

// HandlerSettings of Trigger
type HandlerSettings struct {
	Subject string `md:"subject,required"`
	Queue string `md:"queue"`
	ChannelID string `md:"channelId"`
	DurableName string `md:"durableName"`
	MaxInFlight int `md:"maxInFlight"`
	EnableAutoAcknowledgement bool `md:"enableAutoAcknowledgement"`
	AckWaitInSeconds int `md:"ackWaitInSeconds"`
	EnableStartWithLastReceived bool `md:"enableStartWithLastReceived"`

}

// FromMap method of HandlerSettings
func (h *HandlerSettings) FromMap(values map[string]interface{}) error {
	var err error
	h.Subject, err = coerce.ToString(values["subject"])
	if err != nil {
		return err
	}

	h.Queue, err = coerce.ToString(values["queue"])
	if err != nil {
		return err
	}

	h.ChannelID, err = coerce.ToString(values["channelId"])
	if err != nil {
		return err
	}

	h.DurableName, err = coerce.ToString(values["durableName"])
	if err != nil {
		return err
	}

	h.MaxInFlight, err = coerce.ToInt(values["maxInFlight"])
	if err != nil {
		return err
	}

	h.EnableAutoAcknowledgement, err = coerce.ToBool(values["enableAutoAcknowledgement"])
	if err != nil {
		return err
	}

	h.AckWaitInSeconds, err = coerce.ToInt(values["ackWaitInSeconds"])
	if err != nil {
		return err
	}

	h.EnableStartWithLastReceived, err = coerce.ToBool(values["enableStartWithLastReceived"])
	if err != nil {
		return err
	}

	return nil
}

// ToMap method of HandlerSettings
func (h *HandlerSettings) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"subject": h.Subject,
		"queue": h.Queue,
		"channelId": h.ChannelID,
		"durableName": h.DurableName,
		"maxInFlight": h.MaxInFlight,
		"enableAutoAcknowledgement": h.EnableAutoAcknowledgement,
		"ackWaitInSeconds": h.AckWaitInSeconds,
		"enableStartWithLastReceived": h.EnableStartWithLastReceived,
	}
}

// Output of Trigger
type Output struct {
	PayloadFormat string `md:"payloadFormat"`
	Payload interface{} `md:"payload"`
}

// FromMap method of Output
func (o *Output) FromMap(values map[string]interface{}) error {
	var err error 

	o.PayloadFormat, err = coerce.ToString(values["payloadFormat"])
	if err != nil {
		return err
	}

	o.Payload, err = coerce.ToAny(values["payload"])
	if err != nil {
		return err
	}

	return nil
}

// ToMap method of Output
func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"payloadFormat": o.PayloadFormat,
		"payload": o.Payload,
	}
}

// Reply of Trigger
type Reply struct {
	Data interface{} `md:"data"`
}

// FromMap method of Reply
func (r *Reply) FromMap(values map[string]interface{}) error {
	var err error

	r.Data, err = coerce.ToAny(values["data"])
	if err != nil {
		return err
	}

	return nil
}

// ToMap method of Reply
func (r *Reply) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"data": r.Data,
	}
}