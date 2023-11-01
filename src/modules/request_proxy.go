package modules

import (
	"github.com/pkg/errors"
	"github.com/project-nano/framework"
	"log"
	"time"
)

type ProxyRequest struct {
	Request      framework.Message
	Timeout      time.Duration
	ResponseChan chan ProxyResult
}

type ProxyResult struct {
	Response framework.Message
	Error    error
}

type RequestProxy struct {
	Module       string
	ResponseChan chan framework.Message
	RequestChan  chan ProxyRequest
	sender       framework.MessageSender
	notifyChan   chan bool
	exitChan     chan bool
}

const (
	APIModuleName = "API"
)

func CreateRequestProxy(sender framework.MessageSender) (*RequestProxy, error) {
	const (
		queueLength = 1 << 10
	)
	proxy := RequestProxy{APIModuleName, make(chan framework.Message, queueLength),
		make(chan ProxyRequest, queueLength), sender, make(chan bool), make(chan bool)}
	return &proxy, nil
}

func (proxy *RequestProxy) Start() error {
	go proxy.routine()
	return nil
}

func (proxy *RequestProxy) Stop() error {
	proxy.notifyChan <- true
	<-proxy.exitChan
	return nil
}

func (proxy *RequestProxy) routine() {
	type proxySession struct {
		Allocated bool
		Elapse    time.Time
		Chan      chan ProxyResult
	}

	const (
		MinSessionID   = 1
		SessionIDRange = 1 << 8 //256
		MaxSessionID   = MinSessionID + SessionIDRange
	)
	var sessions = map[framework.SessionID]proxySession{}
	var id framework.SessionID
	for id = MinSessionID; id < MaxSessionID; id++ {
		sessions[id] = proxySession{Allocated: false}
	}

	var timeoutTicker = time.NewTicker(1 * time.Second)
	var lastID framework.SessionID = MaxSessionID
	exitFlag := false
	for !exitFlag {
		select {
		case <-proxy.notifyChan:
			exitFlag = true
		case request := <-proxy.RequestChan:
			//allocate session
			seed := lastID
			var try framework.SessionID
			var processed = false
			for try = 0; try < SessionIDRange; try++ {
				id := (seed+try)%SessionIDRange + MinSessionID
				session, exists := sessions[id]
				if !exists {
					log.Printf("<proxy> invalid session [%08X]", id)
					break
				}
				if session.Allocated {
					continue
				}
				//available
				//log.Printf("<proxy> [%08X] session allocated", id)
				msg := request.Request
				msg.SetSender(proxy.Module)
				msg.SetFromSession(id)
				processed = true
				if err := proxy.sender.SendToSelf(msg); err != nil {
					log.Printf("<proxy> send message fail: %s", err.Error())
					break
				}
				sessions[id] = proxySession{true, time.Now().Add(request.Timeout), request.ResponseChan}
				break
			}
			if !processed {
				log.Println("<proxy> warning: no session avaialble")
			}

		case resp := <-proxy.ResponseChan:
			id = resp.GetToSession()
			session, exists := sessions[id]
			if !exists {
				log.Printf("<proxy> invalid session id [%08X] with response message [%08X]", id, resp.GetID())
				break
			}
			if !session.Allocated {
				log.Printf("<proxy> warning: response message [%08X] ignored due to session [%08X] deallocated", resp.GetID(), id)
				break
			}
			//log.Printf("<proxy> [%08X] session finished with response [%08X]", id, resp.GetID())
			session.Chan <- ProxyResult{resp, nil}
			sessions[id] = proxySession{Allocated: false}

		case <-timeoutTicker.C:
			//timeout check
			now := time.Now()
			var timeoutList []framework.SessionID
			var session proxySession
			for id, session = range sessions {
				if !session.Allocated {
					continue
				}
				if session.Elapse.Before(now) {
					//timeout
					log.Printf("<proxy> [%08X] timeout", id)
					session.Chan <- ProxyResult{Error: errors.New("timeout")}
					timeoutList = append(timeoutList, id)
					continue
				}
			}
			if 0 != len(timeoutList) {
				for _, id = range timeoutList {
					//reset
					sessions[id] = proxySession{Allocated: false}
				}
			}
		}
	}
	proxy.exitChan <- true
}

func (proxy *RequestProxy) SendRequest(request framework.Message, respChan chan ProxyResult) error {
	proxy.RequestChan <- ProxyRequest{request, GetConfigurator().GetOperateTimeout(), respChan}
	return nil
}

func (proxy *RequestProxy) SendRequestTimeout(request framework.Message, timeout time.Duration, respChan chan ProxyResult) error {
	proxy.RequestChan <- ProxyRequest{request, timeout, respChan}
	return nil
}
