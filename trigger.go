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
	triggerConfig *trigger.Config
	natsHandlers  []*Handler
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
		return err
	}

	logger.Debugf("Settings: %v", s)

	// Resolving auth settings
	// if s.Auth != nil {
	// 	ctx.Logger().Debugf("auth settings being resolved: %v", s.Auth)
	// 	auth, err := resolveObject(s.Auth)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	s.Auth = auth
	// 	ctx.Logger().Debugf("auth settings resolved: %v", s.Auth)
	// }

	// Resolving reconnect settings
	// if s.Reconnect != nil {
	// 	ctx.Logger().Debugf("reconnect settings being resolved: %v", s.Reconnect)
	// 	reconnect, err := resolveObject(s.Reconnect)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	s.Reconnect = reconnect
	// 	ctx.Logger().Debugf("reconnect settings resolved: %v", s.Reconnect)
	// }

	// Resolving sslConfig settings
	// if s.SslConfig != nil {
	// 	ctx.Logger().Debugf("sslConfig settings being resolved: %v", s.SslConfig)
	// 	sslConfig, err := resolveObject(s.SslConfig)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	s.SslConfig = sslConfig
	// 	ctx.Logger().Debugf("sslConfig settings resolved: %v", s.SslConfig)
	// }

	// Resolving sslConfig settings
	// if s.Streaming != nil {
	// 	ctx.Logger().Debugf("streaming settings being resolved: %v", s.Streaming)
	// 	streaming, err := resolveObject(s.Streaming)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	s.Streaming = streaming
	// 	ctx.Logger().Debugf("streaming settings resolved: %v", s.Streaming)
	// }

	for _, handler := range ctx.GetHandlers() {

		// Create handler settings
		logger.Infof("Mapping handler settings...")
		handlerSettings := &HandlerSettings{}
		if err := handlerSettings.FromMap(handler.Settings()); err != nil {
			return err
		}
		logger.Debugf("handlerSettings: %v", handlerSettings)
		logger.Infof("Mapped handler settings successfully")

		// Get NATS Connection
		logger.Infof("Getting NATS connection...")
		nc, err := getNatsConnection(logger, s)
		if err != nil {
			return err
		}
		logger.Infof("Got NATS connection")

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
		logger.Infof("Checking NATS Streaming ...")
		if s.EnableStreaming {
			logger.Infof("NATS Streaming is enabled")
			natsHandler.natsStreaming = s.EnableStreaming
			if natsHandler.natsStreaming {
				natsHandler.stanConn, err = getStanConnection(logger, s, nc) // Create STAN connection
				if err != nil {
					return err
				}
				logger.Infof("Got STAN connection")
				natsHandler.stanMsgChannel = make(chan *stan.Msg) // Create STAN message channel
			}
		} else {
			logger.Infof("NATS Streaming is disabled")
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
			result, err = h.triggerHandler.Handle(context.Background(), out.ToMap())
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

	urlString = settings.NatsClusterUrls

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
	if len(settings.NatsConnName) > 0 {
		natsOptions = append(natsOptions, nats.Name(settings.NatsConnName))
	}

	return nats.Connect(urlString, natsOptions...)

}

// checkClusterUrls is function to all valid NATS cluster urls
func checkClusterUrls(settings *Settings) error {
	// Check ClusterUrls
	clusterUrls := strings.Split(settings.NatsClusterUrls, ",")
	if len(clusterUrls) < 1 {
		return fmt.Errorf("ClusterUrl [%v] is invalid, require at least one url", settings.NatsClusterUrls)
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

	if settings.NatsUserName != "" { // Check if usename is defined
	  // check if password is defined
		if settings.NatsUserPassword == "" {
			return nil, fmt.Errorf("Missing password")
		} else {
			// Create UserInfo NATS option
			opts = append(opts, nats.UserInfo(settings.NatsUserName, settings.NatsUserPassword))
		}
	} else if settings.NatsToken != "" { // Check if token is defined
		opts = append(opts, nats.Token(settings.NatsToken))
	} else if settings.NatsNkeySeedfile != "" { // Check if nkey seed file is defined
		nkey, err := nats.NkeyOptionFromSeed(settings.NatsNkeySeedfile)
		if err != nil {
			return nil, err
		}
		opts = append(opts, nkey)
	} else if settings.NatsCredentialFile != "" { // Check if credential file is defined
		opts = append(opts, nats.UserCredentials(settings.NatsCredentialFile))
	}
	return opts, nil
}

func getNatsConnReconnectOpts(settings *Settings) ([]nats.Option, error) {
	opts := make([]nats.Option, 0)

	// Enable autoReconnect
	if !settings.AutoReconnect {
		opts = append(opts, nats.NoReconnect())
	}
	
	// Max reconnect attempts
	if settings.MaxReconnects > 0 {
		opts = append(opts, nats.MaxReconnects(settings.MaxReconnects))
	}

	// Don't randomize
	if settings.EnableRandomReconnection {
		opts = append(opts, nats.DontRandomize())
	}

	// Reconnect wait in seconds
	if settings.ReconnectWait > 0 {
		duration, err := time.ParseDuration(fmt.Sprintf("%vs", settings.ReconnectWait))
		if err != nil {
			return nil, err
		}
		opts = append(opts, nats.ReconnectWait(duration))
	}

	// Reconnect buffer size in bytes
	if settings.ReconnectBufferSize > 0 {
		opts = append(opts, nats.ReconnectBufSize(settings.ReconnectBufferSize))
	}
	return opts, nil
}

func getNatsConnSslConfigOpts(settings *Settings) ([]nats.Option, error) {
	opts := make([]nats.Option, 0)

	if settings.CertFile != "" && settings.KeyFile != "" {
		// Skip verify
		if settings.SkipVerify {
			opts = append(opts, nats.Secure(&tls.Config{
				InsecureSkipVerify: settings.SkipVerify,
			}))
		}
		// CA Root
		if settings.CaFile != "" {
			opts = append(opts, nats.RootCAs(settings.CaFile))
			// Cert file
			if settings.CertFile != "" {
				if settings.KeyFile != "" {
					opts = append(opts, nats.ClientCert(settings.CertFile, settings.KeyFile))
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

func getStanConnection(logger log.Logger, settings *Settings, conn *nats.Conn) (stan.Conn, error) {

	if settings.StanClusterID == "" {
		return nil, fmt.Errorf("missing stanClusterId")
	}

	logger.Debugf("clusterID: %v", settings.StanClusterID)
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	hostname = strings.Split(hostname, ".")[0]
	hostname = strings.Split(hostname, ":")[0]
	logger.Debugf("hostname: %v", hostname)
	logger.Debugf("natsConn: %v", conn)

	sc, err := stan.Connect(settings.StanClusterID, hostname, stan.NatsConn(conn))
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

// func resolveObject(object map[string]interface{}) (map[string]interface{}, error) {
// 	var err error

// 	mapperFactory := mapper.NewFactory(resolver)
// 	valuesMapper, err := mapperFactory.NewMapper(object)
// 	if err != nil {
// 		return nil, err
// 	}

// 	objectValues, err := valuesMapper.Apply(data.NewSimpleScope(map[string]interface{}{}, nil))
// 	if err != nil {
// 		return nil, err
// 	}

// 	return objectValues, nil
// }
