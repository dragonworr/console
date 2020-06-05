package api

import (
	"context"
	"fmt"
	"github.com/cloudhut/kowl/backend/pkg/kafka"
	"github.com/cloudhut/kowl/backend/pkg/owl"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/cloudhut/common/rest"
	"github.com/gorilla/websocket"
)

func allowAll(r *http.Request) bool { return true }

// websocket upgrader options
var upgrader = websocket.Upgrader{EnableCompression: true, CheckOrigin: allowAll}

// GetTopicMessagesResponse is a wrapper for an array of TopicMessage
type GetTopicMessagesResponse struct {
	KafkaMessages *owl.ListMessageResponse `json:"kafkaMessages"`
}

// This thing is given to 'ListMessages' so it can send updates about the search to client
type progressReporter struct {
	logger  *zap.Logger
	request *owl.ListMessageRequest

	wsMutex   *sync.Mutex
	websocket *websocket.Conn

	debugDelayTicker *time.Ticker
}

func (p *progressReporter) OnPhase(name string) {
	p.wsMutex.Lock()
	defer p.wsMutex.Unlock()

	p.websocket.EnableWriteCompression(false)
	p.websocket.WriteJSON(struct {
		Type  string `json:"type"`
		Phase string `json:"phase"`
	}{"phase", name})
}

func (p *progressReporter) OnMessage(message *kafka.TopicMessage) {
	p.wsMutex.Lock()
	defer p.wsMutex.Unlock()

	//<-p.debugDelayTicker.C
	p.websocket.EnableWriteCompression(true)
	p.websocket.WriteJSON(struct {
		Type    string              `json:"type"`
		Message *kafka.TopicMessage `json:"message"`
	}{"message", message})
}

func (p *progressReporter) OnComplete(elapsedMs float64, isCancelled bool) {
	p.wsMutex.Lock()
	defer p.wsMutex.Unlock()

	p.websocket.EnableWriteCompression(false)
	p.websocket.WriteJSON(struct {
		Type        string  `json:"type"`
		ElapsedMs   float64 `json:"elapsedMs"`
		IsCancelled bool    `json:"isCancelled"`
	}{"done", elapsedMs, isCancelled})
}

func (p *progressReporter) OnError(message string) {
	p.wsMutex.Lock()
	defer p.wsMutex.Unlock()

	p.websocket.EnableWriteCompression(false)
	p.websocket.WriteJSON(struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}{"error", message})
}

type listMessagesRequest struct {
	TopicName   string `json:"topicName"`
	StartOffset int64  `json:"startOffset"` // -1 for newest, -2 for oldest offset
	PartitionID int32  `json:"partitionId"` // -1 for all partition ids
	MaxResults  uint16 `json:"maxResults"`
}

func (l *listMessagesRequest) OK() error {
	if l.TopicName == "" {
		return fmt.Errorf("topic name is required")
	}

	if l.StartOffset < -2 {
		return fmt.Errorf("start offset is smaller than -2")
	}

	if l.PartitionID < -1 {
		return fmt.Errorf("partitionID is smaller than -1")
	}

	if l.MaxResults <= 0 || l.MaxResults > 500 {
		return fmt.Errorf("max results must be between 1 and 500")
	}

	return nil
}

func (api *API) handleGetMessages() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		logger := api.Logger

		// Websocket upgrade
		logger.Debug("starting websocket connection upgrade")

		wsConnection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("failed to upgrade websocket in messages endpoint", zap.Error(err))
			return
		}
		wsConnection.SetCloseHandler(func(code int, text string) error {
			logger.Debug("connection has been closed by client")
			r.Context().Done()
			return nil
		})
		logger.Debug("websocket connection upgrade complete")

		defer func() {
			// Close connection gracefully!
			err = wsConnection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil && err != websocket.ErrCloseSent {
				logger.Debug("failed to send 'CloseNormalClosure' to ws connection", zap.Error(err))
			} else {
				//logger.Debug("graceful WS close message sent")
				// the example in github.com/gorilla/websocket also does this
				time.Sleep(2 * time.Second)
			}
		}()

		// Get search parameters. Close connection if search parameters are invalid
		maxMessageSize := int64(16 * 1024) // 16kb
		wsConnection.SetReadLimit(maxMessageSize)
		var req listMessagesRequest
		err = wsConnection.ReadJSON(&req)
		if err != nil {
			rest.SendRESTError(w, r, api.Logger, &rest.Error{
				Err:      err,
				Status:   http.StatusBadRequest,
				Message:  "Failed to parse list message request",
				IsSilent: false,
			})
			return
		}

		err = req.OK()
		if err != nil {
			rest.SendRESTError(w, r, api.Logger, &rest.Error{
				Err:      err,
				Status:   http.StatusBadRequest,
				Message:  fmt.Sprintf("Failed to validate list message request: %v", err),
				IsSilent: false,
			})
			return
		}

		// Check if logged in user is allowed to list messages for the given topic
		canViewMessages, restErr := api.Hooks.Owl.CanViewTopicMessages(r.Context(), req.TopicName)
		if restErr != nil {
			rest.SendRESTError(w, r, logger, restErr)
			return
		}
		if !canViewMessages {
			restErr := &rest.Error{
				Err:      fmt.Errorf("requester has no permissions to view messages in the requested topic"),
				Status:   http.StatusForbidden,
				Message:  "You don't have permissions to view messages in that topic",
				IsSilent: false,
			}
			rest.SendRESTError(w, r, logger, restErr)
			return
		}

		// Request messages from kafka and return them once we got all the messages or the context is done
		listReq := owl.ListMessageRequest{
			TopicName:    req.TopicName,
			PartitionID:  req.PartitionID,
			StartOffset:  req.StartOffset,
			MessageCount: req.MaxResults,
		}

		progress := &progressReporter{api.Logger, &listReq, &sync.Mutex{}, wsConnection, time.NewTicker(300 * time.Millisecond)}

		ctx, cancelCtx := context.WithTimeout(r.Context(), 18*time.Second)
		defer cancelCtx()

		err = api.OwlSvc.ListMessages(ctx, listReq, progress)
		if err != nil {
			progress.OnError(err.Error())
		}
	}
}
