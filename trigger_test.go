package nats

import (
	"encoding/json"
	"time"
	"testing"

	"github.com/project-flogo/core/action"
	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/support/test"
	"github.com/project-flogo/core/trigger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/nats-io/gnatsd/server"
	natsserver "github.com/nats-io/nats-server/test"
)

type NatsTriggerTestSuite struct {
	suite.Suite
	natsTestConfig string
}

func (suite *NatsTriggerTestSuite) SetupTest() {
	suite.natsTestConfig = `{
		"id": "flogo-nats-trigger",
		"ref": "github.com/codelity-co/flogo-nats-trigger",
		"settings": {
			"natsClusterUrls": "nats://localhost:4222",
			"natsConnName": "NATS Connection"
		},
		"handlers": [
			{
				"settings": {
					"subject": "flogo"
				},
				"action": {
					"id": "dummy"
				}
			}
		]
	}`
}

func (suite *NatsTriggerTestSuite) TestFactoryNew() {
	ref := support.GetRef(&Trigger{})
	f := trigger.GetFactory(ref)
	assert.NotNil(suite.T(), f, "Should return factory instance")
}

func (suite *NatsTriggerTestSuite) TestFactoryMetadata() {
	t := suite.T()

	ref := support.GetRef(&Trigger{})
	f := trigger.GetFactory(ref)
	assert.NotNil(t, f, "Should return factory instance")

	m := f.Metadata()
	assert.NotNil(t, m, "Factory should return its metadata")
}

func (suite *NatsTriggerTestSuite) TestTriggerInitialize() {
	t := suite.T()

	s := RunServerWithOptions()
	defer s.Shutdown()

	ref := support.GetRef(&Trigger{})
	f := trigger.GetFactory(ref)
	assert.NotNil(t, f, "Should return factory instance")

	// Test trigger configuration
	config := &trigger.Config{}
	err := json.Unmarshal([]byte(suite.natsTestConfig), config)
	assert.Nil(t, err, "Invalid trigger config")

	triggerSettings := &Settings{}
	err = metadata.MapToStruct(config.Settings, triggerSettings, true)
	assert.Nil(t, err, "MapToStruct error when converting json to Settings")
	
	// Create dummy action
	actions := map[string]action.Action{"dummy": test.NewDummyAction(func() {
		//do nothing
	})}

	// Test trigger instance
	trg, err := test.InitTrigger(f, config, actions)
	assert.Nil(t, err, "InitTrigger return error")
	assert.NotNil(t, trg, "Should return trigger instance")
}

func (suite *NatsTriggerTestSuite) TestHandlerGetConnection() {
	t := suite.T()

	s := RunServerWithOptions()
	defer s.Shutdown()

	ref := support.GetRef(&Trigger{})
	f := trigger.GetFactory(ref)
	assert.NotNil(t, f, "Should return factory instance")

	// Test trigger configuration
	config := &trigger.Config{}
	err := json.Unmarshal([]byte(suite.natsTestConfig), config)
	assert.Nil(t, err, "Invalid trigger config")

	triggerSettings := &Settings{}
	err = metadata.MapToStruct(config.Settings, triggerSettings, true)
	assert.Nil(t, err, "MapToStruct error when converting json to Settings")

	// Test getConnection
	h := &Handler{
		logger: log.RootLogger(),
		triggerSettings: triggerSettings,
		handlerSettings: &HandlerSettings{
			Subject: "flogo",
			Queue: "flogo",
		},
	}

	err = h.getConnection()
	assert.Nil(t, err, "getConnection error")
}

func (suite *NatsTriggerTestSuite) TestHandlerStartChannel() {
	t := suite.T()

	s := RunServerWithOptions()
	defer s.Shutdown()

	// Test trigger configuration
	config := &trigger.Config{}
	err := json.Unmarshal([]byte(suite.natsTestConfig), config)
	assert.Nil(t, err, "Invalid trigger config")

	triggerSettings := &Settings{}
	err = metadata.MapToStruct(config.Settings, triggerSettings, true)
	assert.Nil(t, err, "MapToStruct error when converting json to Settings")

	// Test getConnection
	h := &Handler{
		logger: log.RootLogger(),
		triggerSettings: triggerSettings,
		handlerSettings: &HandlerSettings{
			Subject: "flogo",
			Queue: "flogo",
		},
	}

	err = h.getConnection()
	assert.Nil(t, err, "getConnection error")

	// Test startChannel
	err = h.startChannel()
	assert.Nil(t, err, "startChannel error")
}

func (suite *NatsTriggerTestSuite) TestTriggerStartStop() {
	t := suite.T()

	s := RunServerWithOptions()
	defer s.Shutdown()

	ref := support.GetRef(&Trigger{})
	f := trigger.GetFactory(ref)
	assert.NotNil(t, f, "Should return factory instance")

	// Test trigger configuration
	config := &trigger.Config{}
	err := json.Unmarshal([]byte(suite.natsTestConfig), config)
	assert.Nil(t, err, "Invalid trigger config")

	triggerSettings := &Settings{}
	err = metadata.MapToStruct(config.Settings, triggerSettings, true)
	assert.Nil(t, err, "MapToStruct error when converting json to Settings")

	// Test trigger instance
	trg, err := f.New(config)
	assert.Nil(t, err, "InitTrigger return error")
	assert.NotNil(t, trg, "Should return trigger instance")

	err = trg.Start()
	assert.Nil(t, err, "Trigger start return error")

	if duration, err := time.ParseDuration("5s"); err == nil {
		time.Sleep(duration)
	}
	
	err = trg.Stop()
	assert.Nil(t, err, "Trigger stop return error")

}

func TestNatsTriggerTestSuite(t *testing.T) {
	suite.Run(t, new(NatsTriggerTestSuite))
}

func RunServerWithOptions() *server.Server {
	return natsserver.RunServer(&natsserver.DefaultTestOptions)
}