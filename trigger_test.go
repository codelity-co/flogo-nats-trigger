package nats

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"testing"

	"github.com/project-flogo/core/action"
	"github.com/project-flogo/core/support"
	"github.com/project-flogo/core/support/test"
	"github.com/project-flogo/core/trigger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type NatsTriggerTestSuite struct {
	suite.Suite
	natsTestConfig string
	stanTestConfig string
}

func TestNatsTriggerTestSuite(t *testing.T) {
	
	suite.Run(t, new(NatsTriggerTestSuite))
}

func (suite *NatsTriggerTestSuite) SetupSuite() {
	command := exec.Command("docker", "start", "nats-streaming")
	err := command.Run()
	if err != nil {
		fmt.Println(err.Error())
		command := exec.Command(
			"docker", "run", 
			"-p", "4222:4222", 
			"-p", "6222:6222", 
			"-p", "8222:8222", 
			"--name", "nats-streaming", 
			"-d", 
			"nats-streaming",
      "--addr=0.0.0.0",
      "--port=4222",
      "--http_port=8222",
      "--cluster_id=flogo",
			"--store=MEMORY",
		)
		err := command.Run()
		if err != nil {
			fmt.Println(err.Error())
			panic(err)
		}
	}
}

func (suite *NatsTriggerTestSuite) SetupTest() {
	suite.natsTestConfig = `{
		"id": "nats-trigger",
		"ref": "github.com/codelity-co/codelity-flogo-plugins/nats/trigger",
		"settings": {
			"clusterUrls": "nats://localhost:4222"
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

	suite.stanTestConfig = `{
		"id": "nats-trigger",
		"ref": "github.com/codelity-co/codelity-flogo-plugins/nats/trigger",
		"settings": {
			"clusterUrls": "nats://localhost:4222",
			"streaming": {
				"enableStreaming": true,
				"clusterId": "flogo"
			}
		},
		"handlers": [
			{
				"settings": {
					"async": true,
					"subject": "flogo",
					"queue": "flogo",
					"channelId": "flogo",
					"durableName": "flogo",
					"maxInFlight": 25
				},
				"action": {
					"id": "dummy"
				}
			}
		]
	}`
}

func (suite *NatsTriggerTestSuite) TestNatsTrigger_Register() {
	ref := support.GetRef(&Trigger{})
	f := trigger.GetFactory(ref)
	assert.NotNil(suite.T(), f)
}

func (suite *NatsTriggerTestSuite) TestNatsTrigger_NATSInitialize() {
	t := suite.T()

	factory := &Factory{}
	config := &trigger.Config{}
	err := json.Unmarshal([]byte(suite.natsTestConfig), config)
	assert.Nil(t, err)
	actions := map[string]action.Action{"dummy": test.NewDummyAction(func() {
	})}

	trg, err := test.InitTrigger(factory, config, actions)
	assert.Nil(t, err)
	assert.NotNil(t, trg)

	err = trg.Start()
	assert.Nil(t, err)

	err = trg.Stop()
	assert.Nil(t, err)
}

func (suite *NatsTriggerTestSuite) TestNatsTrigger_STANInitialize() {
	t := suite.T()

	factory := &Factory{}
	config := &trigger.Config{}
	err := json.Unmarshal([]byte(suite.stanTestConfig), config)
	assert.Nil(t, err)
	actions := map[string]action.Action{"dummy": test.NewDummyAction(func() {
	})}

	trg, err := test.InitTrigger(factory, config, actions)
	assert.Nil(t, err)
	assert.NotNil(t, trg)

	err = trg.Start()
	assert.Nil(t, err)

	err = trg.Stop()
	assert.Nil(t, err)
}