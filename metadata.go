package nats

import (	
	"github.com/project-flogo/core/app/resolve"
	"github.com/project-flogo/core/data/coerce"
)

type Settings struct {
	ClusterUrls string                 `md:"clusterUrls,required"` // The NATS cluster to connect to
	ConnName    string                 `md:"connName"`
	Auth        map[string]interface{} `md:"auth"`      // Auth setting
	Reconnect   map[string]interface{} `md:"reconnect"` // Reconnect setting
	SslConfig   map[string]interface{} `md:"sslConfig"` // SSL config setting
	Streaming   map[string]interface{} `md:"streaming"` // NATS streaming config
}

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

type HandlerSettings struct {
	Subject string `md:"subject,required"`
	Queue string `md:"queue"`
	ChannelId string `md:"channelId"`
	DurableName string `md:"durableName"`
	MaxInFlight int `md:"maxInFlight"`
	DataFormat string `md:"dataFormat"`
}

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

	h.ChannelId, err = coerce.ToString(values["channelId"])
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

	h.DataFormat, err = coerce.ToString(values["dataFormat"])
	if err != nil {
		return err
	}

	return nil
}

func (h *HandlerSettings) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"subject": h.Subject,
		"queue": h.Queue,
		"channelId": h.ChannelId,
		"durableName": h.DurableName,
		"maxInFlight": h.MaxInFlight,
		"dataFormat": h.DataFormat,
	}
}

type Output struct {
	PayloadFormat string `md:"payloadFormat"`
	Payload interface{} `md:"payload"`
}

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

func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"payloadFormat": o.PayloadFormat,
		"payload": o.Payload,
	}
}

