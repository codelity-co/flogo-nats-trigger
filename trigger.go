package nats

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"

	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
)

var triggerMd = trigger.NewMetadata(&Settings{}, &HandlerSettings{}, &Output{})

func init() {
	_ = trigger.Register(&Trigger{}, &Factory{})
}

// Factory struct
type Factory struct {
}

// New trigger method of Factory
func (*Factory) New(config *trigger.Config) (trigger.Trigger, error) {

	return &Trigger{triggerConfig: config}, nil

}

// Metadata method of Factory
func (f *Factory) Metadata() *trigger.Metadata {
	return triggerMd
}

// Trigger struct
type Trigger struct {
	triggerConfig   *trigger.Config
	natsHandlers 	  []*Handler
}

// Metadata implements trigger.Trigger.Metadata
func (t *Trigger) Metadata() *trigger.Metadata {
	return triggerMd
}

// Initialize method of trigger
func (t *Trigger) Initialize(ctx trigger.InitContext) error {
	logger := ctx.Logger()

	s := &Settings{}
	err := s.FromMap(t.triggerConfig.Settings)
	if err != nil {
		logger.Debugf("Settings: %v", s)
	}

	for _, handler := range ctx.GetHandlers() {

		// Create handler settings
		logger.Debugf("Mapping handler settings...")
		handlerSettings := &HandlerSettings{}
		if err := handlerSettings.FromMap(handler.Settings()); err != nil {
			return err
		}
		logger.Debugf("Mapped handler settings successfully")

		// Get NATS Connection
		logger.Debugf("Getting NATS connection...")
		nc, err := getNatsConnection(logger, s)
		if err != nil {
			return err
		}
		logger.Debugf("Got NATS connection")

		// Create Stop Channel
		logger.Debugf("Registering trigger handler...")
		stopChannel := make(chan bool)

		// Create Trigger Handler
		natsHandler := &Handler{
			handlerSettings: handlerSettings,
			logger:          logger,
			natsConn:        nc,
			stopChannel:     stopChannel,
			triggerHandler:  handler,
		}

		// Check NATS Streaming
		if enableStreaming, ok := s.Streaming["enableStreaming"]; ok {
			natsHandler.natsStreaming = enableStreaming.(bool)
			if natsHandler.natsStreaming {
				natsHandler.stanConn, err = getStanConnection(s, nc) // Create STAN connection
				if err != nil {
					return err
				}
				natsHandler.stanMsgChannel = make(chan *stan.Msg) // Create STAN message channel
			}
		} else {
			natsHandler.natsMsgChannel = make(chan *nats.Msg) // Create NATS message channel
		}

		// Append handler
		t.natsHandlers = append(t.natsHandlers, natsHandler)
		logger.Debugf("Registered trigger handler successfully")

	}

	return nil
}

// Start implements util.Managed.Start
func (t *Trigger) Start() error {
	for _, handler := range t.natsHandlers {
		_ = handler.Start()
	}
	return nil
}

// Stop implements util.Managed.Stop
func (t *Trigger) Stop() error {
	for _, handler := range t.natsHandlers {
		_ = handler.Stop()
	}
	return nil
}

// Handler is a NATS subject handler
type Handler struct {
	handlerSettings  *HandlerSettings
	logger           log.Logger
	natsConn         *nats.Conn
	natsMsgChannel   chan *nats.Msg
	natsStreaming    bool
	natsSubscription *nats.Subscription
	stanConn         stan.Conn
	stanMsgChannel   chan *stan.Msg
	stanSubscription stan.Subscription
	stopChannel      chan bool
	triggerHandler   trigger.Handler
}

func (h *Handler) handleMessage() {
	for {
		select {
		case done := <-h.stopChannel: // Receive message from Stop Channel

			if done {
				return
			}

		case msg := <-h.natsMsgChannel: // Receive NATS Msg from NATS message channel

			var (
				err error
				// results map[string]interface{}
			)
			out := &Output{}
			out.Payload, err = getPayloadData(msg.Data)
			if err != nil {
				h.logger.Errorf("Cannot parse NATS message data: %v", msg.Data)
				continue
			}
			out.PayloadFormat = fmt.Sprintf("%v", reflect.TypeOf(out.Payload))

			var result map[string]interface{}
			result, err = h.triggerHandler.Handle(context.Background(), out.ToMap)
			if err != nil {
				h.logger.Errorf("Trigger handler error: %v", err)
				continue
			}

			var replyBytes []byte
			replyBytes, err = createReply(h.logger, result)
			if err != nil {
				h.logger.Errorf("Cannot create reply struct: %v", err)
				continue
			}
			err = msg.Respond(replyBytes)
			if err != nil {
				h.logger.Errorf("Cannot respond NATS publisher: %v", err)
				continue
			}

		case msg := <-h.stanMsgChannel: // Receive STAN Msg from STAN message channel

			var (
				err error
			)

			// Create Output
			out := &Output{}

			// Get Payload content
			out.Payload, err = getPayloadData(msg.Data)
			if err != nil {
				h.logger.Errorf("Cannot parse STAN message data: %v", msg.Data)
				continue
			}
			out.PayloadFormat = fmt.Sprintf("%v", reflect.TypeOf(out.Payload))

			// Pass output to Handler
			_, err = h.triggerHandler.Handle(context.Background(), out.ToMap())
			if err != nil {
				h.logger.Errorf("Trigger handler error: %v", err)
				continue
			}

			if !h.handlerSettings.EnableAutoAcknowledgement {
				err = msg.Ack()
				if err != nil {
					h.logger.Errorf("Cannot acknowledge message: %v", err)
					continue
				}
			}

		}
	}
}

// Start starts the handler
func (h *Handler) Start() error {
	var err error
	go h.handleMessage()

	if len(h.handlerSettings.Queue) > 0 { // if Queue info is set

		if !h.natsStreaming { // if NATS connection

			// NATS Queue Subcribe
			h.natsSubscription, err = h.natsConn.QueueSubscribe(h.handlerSettings.Subject, h.handlerSettings.Queue, func(m *nats.Msg) {
				h.natsMsgChannel <- m
			})
			if err != nil {
				return err
			}

		} else { // if NATS Streaming Connection

			// Prepare Queue subscription option
			subscriptionOptions := make([]stan.SubscriptionOption, 0)
			if !h.handlerSettings.EnableAutoAcknowledgement {
				subscriptionOptions = append(subscriptionOptions, stan.SetManualAckMode())
				actWait, _ := time.ParseDuration(fmt.Sprintf("%vs", h.handlerSettings.AckWaitInSeconds))
				subscriptionOptions = append(subscriptionOptions, stan.AckWait(actWait))
			}
			if h.handlerSettings.EnableStartWithLastReceived {
				subscriptionOptions = append(subscriptionOptions, stan.StartWithLastReceived())
			}

			// STAN Queue Subscribe
			h.stanSubscription, err = h.stanConn.QueueSubscribe(h.handlerSettings.ChannelID, h.handlerSettings.Queue, func(m *stan.Msg) {
				h.stanMsgChannel <- m
			}, subscriptionOptions...)
			if err != nil {
				return err
			}
		}

	} else { // If no Queueu info

		if !h.natsStreaming { // If NATS connection

			// NATS Subscribe
			h.natsSubscription, err = h.natsConn.Subscribe(h.handlerSettings.Subject, func(m *nats.Msg) {
				h.natsMsgChannel <- m
			})
			if err != nil {
				return err
			}

		} else { // if NATS Streaming connection

			// Prepare subscription option
			subscriptionOptions := make([]stan.SubscriptionOption, 0)
			if !h.handlerSettings.EnableAutoAcknowledgement {
				subscriptionOptions = append(subscriptionOptions, stan.SetManualAckMode())
				actWait, _ := time.ParseDuration(fmt.Sprintf("%vs", h.handlerSettings.AckWaitInSeconds))
				subscriptionOptions = append(subscriptionOptions, stan.AckWait(actWait))
			}
			if h.handlerSettings.EnableStartWithLastReceived {
				subscriptionOptions = append(subscriptionOptions, stan.StartWithLastReceived())
			}

			// STAN Subscribe
			h.stanSubscription, err = h.stanConn.Subscribe(h.handlerSettings.ChannelID, func(m *stan.Msg) {
				h.stanMsgChannel <- m
			}, subscriptionOptions...)
			if err != nil {
				return err
			}

		}

	}
	return nil
}

// Stop stops the handler
func (h *Handler) Stop() error {

	h.stopChannel <- true

	if !h.natsStreaming {
		close(h.natsMsgChannel)
	} else {
		close(h.stanMsgChannel)
		h.stanConn.Close()
	}

	close(h.stopChannel)

	_ = h.natsConn.Drain()
	h.natsConn.Close()

	return nil
}

func getNatsConnection(logger log.Logger, settings *Settings) (*nats.Conn, error) {
	var (
		err           error
		authOpts      []nats.Option
		reconnectOpts []nats.Option
		sslConfigOpts []nats.Option
		urlString     string
	)

	// Check ClusterUrls
	if err := checkClusterUrls(settings); err != nil {
		return nil, err
	}

	urlString = settings.ClusterUrls

	authOpts, err = getNatsConnAuthOpts(settings)
	if err != nil {
		return nil, err
	}

	reconnectOpts, err = getNatsConnReconnectOpts(settings)
	if err != nil {
		return nil, err
	}

	sslConfigOpts, err = getNatsConnSslConfigOpts(settings)
	if err != nil {
		return nil, err
	}

	natsOptions := append(authOpts, reconnectOpts...)
	natsOptions = append(natsOptions, sslConfigOpts...)

	// Check ConnName
	if len(settings.ConnName) > 0 {
		natsOptions = append(natsOptions, nats.Name(settings.ConnName))
	}

	return nats.Connect(urlString, natsOptions...)

}

// checkClusterUrls is function to all valid NATS cluster urls
func checkClusterUrls(settings *Settings) error {
	// Check ClusterUrls
	clusterUrls := strings.Split(settings.ClusterUrls, ",")
	if len(clusterUrls) < 1 {
		return fmt.Errorf("ClusterUrl [%v] is invalid, require at least one url", settings.ClusterUrls)
	}
	for _, v := range clusterUrls {
		if err := validateClusterURL(v); err != nil {
			return err
		}
	}
	return nil
}

// validateClusterUrl is function to check NATS cluster url specificaiton
func validateClusterURL(url string) error {
	hostPort := strings.Split(url, ":")
	if len(hostPort) < 2 || len(hostPort) > 3 {
		return fmt.Errorf("ClusterUrl must be composed of sections like \"{nats|tls}://host[:port]\"")
	}
	if len(hostPort) == 3 {
		i, err := strconv.Atoi(hostPort[2])
		if err != nil || i < 0 || i > 32767 {
			return fmt.Errorf("port specification [%v] is not numeric and between 0 and 32767", hostPort[2])
		}
	}
	if (hostPort[0] != "nats") && (hostPort[0] != "tls") {
		return fmt.Errorf("protocol schema [%v] is not nats or tls", hostPort[0])
	}

	return nil
}

// getNatsConnAuthOps return slice of nats.Option specific for NATS authentication
func getNatsConnAuthOpts(settings *Settings) ([]nats.Option, error) {
	opts := make([]nats.Option, 0)
	// Check auth setting
	if settings.Auth != nil {
		if username, ok := settings.Auth["username"]; ok { // Check if usename is defined
			password, ok := settings.Auth["password"] // check if password is defined
			if !ok {
				return nil, fmt.Errorf("Missing password")
			}

			// Create UserInfo NATS option
			opts = append(opts, nats.UserInfo(username.(string), password.(string)))

		} else if token, ok := settings.Auth["token"]; ok { // Check if token is defined
			opts = append(opts, nats.Token(token.(string)))
		} else if nkeySeedfile, ok := settings.Auth["nkeySeedfile"]; ok { // Check if nkey seed file is defined
			nkey, err := nats.NkeyOptionFromSeed(nkeySeedfile.(string))
			if err != nil {
				return nil, err
			}
			opts = append(opts, nkey)
		} else if credfile, ok := settings.Auth["credfile"]; ok { // Check if credential file is defined
			opts = append(opts, nats.UserCredentials(credfile.(string)))
		}
	}
	return opts, nil
}

func getNatsConnReconnectOpts(settings *Settings) ([]nats.Option, error) {
	opts := make([]nats.Option, 0)
	// Check reconnect setting
	if settings.Reconnect != nil {

		// Enable autoReconnect
		if autoReconnect, ok := settings.Reconnect["autoReconnect"]; ok {
			if !autoReconnect.(bool) {
				opts = append(opts, nats.NoReconnect())
			}
		}

		// Max reconnect attempts
		if maxReconnects, ok := settings.Reconnect["maxReconnects"]; ok {
			opts = append(opts, nats.MaxReconnects(maxReconnects.(int)))
		}

		// Don't randomize
		if dontRandomize, ok := settings.Reconnect["dontRandomize"]; ok {
			if dontRandomize.(bool) {
				opts = append(opts, nats.DontRandomize())
			}
		}

		// Reconnect wait in seconds
		if reconnectWait, ok := settings.Reconnect["reconnectWait"]; ok {
			duration, err := time.ParseDuration(fmt.Sprintf("%vs", reconnectWait))
			if err != nil {
				return nil, err
			}
			opts = append(opts, nats.ReconnectWait(duration))
		}

		// Reconnect buffer size in bytes
		if reconnectBufSize, ok := settings.Reconnect["reconnectBufSize"]; ok {
			opts = append(opts, nats.ReconnectBufSize(reconnectBufSize.(int)))
		}
	}
	return opts, nil
}

func getNatsConnSslConfigOpts(settings *Settings) ([]nats.Option, error) {
	opts := make([]nats.Option, 0)

	// Check sslConfig setting
	if settings.SslConfig != nil {

		// Skip verify
		if skipVerify, ok := settings.SslConfig["skipVerify"]; ok {
			opts = append(opts, nats.Secure(&tls.Config{
				InsecureSkipVerify: skipVerify.(bool),
			}))
		}

		// CA Root
		if caFile, ok := settings.SslConfig["caFile"]; ok {
			opts = append(opts, nats.RootCAs(caFile.(string)))
			// Cert file
			if certFile, ok := settings.SslConfig["certFile"]; ok {
				if keyFile, ok := settings.SslConfig["keyFile"]; ok {
					opts = append(opts, nats.ClientCert(certFile.(string), keyFile.(string)))
				} else {
					return nil, fmt.Errorf("Missing keyFile setting")
				}
			} else {
				return nil, fmt.Errorf("Missing certFile setting")
			}
		} else {
			return nil, fmt.Errorf("Missing caFile setting")
		}

	}
	return opts, nil
}

func getStanConnection(ts *Settings, conn *nats.Conn) (stan.Conn, error) {

	var (
		err       error
		clusterID interface{}
		ok        bool
		hostname  string
		sc        stan.Conn
	)

	clusterID, ok = ts.Streaming["clusterId"]
	if !ok {
		return nil, fmt.Errorf("clusterId not found")
	}

	hostname, err = os.Hostname()
	hostname = strings.Split(hostname, ".")[0]
	hostname = strings.Split(hostname, ":")[0]

	fmt.Println(hostname)

	if err != nil {
		return nil, err
	}

	sc, err = stan.Connect(clusterID.(string), hostname, stan.NatsConn(conn))
	if err != nil {
		return nil, err
	}

	return sc, nil
}

func getPayloadData(data []byte) (interface{}, error) {
	var outputVar interface{}

	err := json.Unmarshal(data, &outputVar)
	if err != nil {
		return nil, err
	}

	return outputVar, nil
}

func createReply(logger log.Logger, result map[string]interface{}) ([]byte, error) {
	var err error

	reply := &Reply{}
	err = reply.FromMap(result)
	if err != nil {
		return nil, err
	}

	if reply.Data != nil {
		dataJSON, err := json.Marshal(reply.Data)
		if err != nil {
			return nil, err
		}
		return dataJSON, nil
	}

	return nil, nil
}
