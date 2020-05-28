package nats

import (	
	"github.com/project-flogo/core/app/resolve"
	"github.com/project-flogo/core/data/coerce"
)

// Settings of trigger
type Settings struct {
	ClusterUrls string                 `md:"clusterUrls,required"` // The NATS cluster to connect to
	ConnName    string                 `md:"connName"`
	Auth        map[string]interface{} `md:"auth"`      // Auth setting
	Reconnect   map[string]interface{} `md:"reconnect"` // Reconnect setting
	SslConfig   map[string]interface{} `md:"sslConfig"` // SSL config setting
	Streaming   map[string]interface{} `md:"streaming"` // NATS streaming config
}

// FromMap method of Settings
func (s *Settings) FromMap(values map[string]interface{}) error {

	var (
		err error
	)
	s.ClusterUrls, err = coerce.ToString(values["clusterUrls"])
	if err != nil {
		return err
	}

	s.ConnName, err = coerce.ToString(values["connName"])
	if err != nil {
		return err
	}

	if values["auth"] != nil {
		s.Auth = make(map[string]interface{})
		for k, v := range values["auth"].(map[string]interface{}) {
			s.Auth[k], err = s.MapValue(v)
			if err != nil {
				return err
			}
		}
	}

	if values["reconnect"] != nil {
		s.Reconnect = make(map[string]interface{})
		for k, v := range values["reconnect"].(map[string]interface{}) {
			s.Reconnect[k], err = s.MapValue(v)
			if err != nil {
				return err
			}
		}
	}

	if values["sslConfig"] != nil {
		s.SslConfig = make(map[string]interface{})
		for k, v := range values["sslConfig"].(map[string]interface{}) {
			s.SslConfig[k], err = s.MapValue(v)
			if err != nil {
				return err
			}
		}
	}

	if values["streaming"] != nil {
		s.Streaming = make(map[string]interface{})
		for k, v := range values["streaming"].(map[string]interface{}) {
			s.Streaming[k], err = s.MapValue(v)
			if err != nil {
				return err
			}
		}
	}

	return nil

}

// ToMap method of Settings
func (s *Settings) ToMap() map[string]interface{} {

	return map[string]interface{}{
		"clusterUrls": s.ClusterUrls,
		"connName":    s.ConnName,
		"auth":        s.Auth,
		"reconnect":   s.Reconnect,
		"sslConfig":   s.SslConfig,
		"streaming":   s.Streaming,
	}

}

func (s *Settings) MapValue(value interface{}) (interface{}, error) {
	var (
		err      error
		anyValue interface{}
	)

	switch val := value.(type) {
	case string:
		if len(val) > 0 && val[0] == '=' {
			anyValue, err = resolve.Resolve(val[1:], nil)
			if err != nil {
				return nil, err
			}
		} else {
			anyValue, err = coerce.ToAny(val)
			if err != nil {
				return nil, err
			}
		}
		
	case map[string]interface{}:
		dataMap := make(map[string]interface{})
		for k, v := range val {
			dataMap[k], err = s.MapValue(v)
			if err != nil {
				return nil, err
			}
		}
		anyValue = dataMap

	default:
		anyValue, err = coerce.ToAny(val)
		if err != nil {
			return nil, err
		}
	}

	return anyValue, nil
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