package server

import (
	"fmt"
	"l3/bfd/bfddCommonDefs"
	"math/rand"
	"net"
	"strconv"
	"time"
)

func (server *BFDServer) processSessionConfig(sessionConfig bfddCommonDefs.BfdSessionConfig) error {
	var sessionMgmt BfdSessionMgmt
	sessionMgmt.DestIp = sessionConfig.DestIp
	sessionMgmt.Protocol = sessionConfig.Protocol
	if sessionConfig.Operation == bfddCommonDefs.CREATE {
		server.CreateSessionCh <- sessionMgmt
	}
	if sessionConfig.Operation == bfddCommonDefs.DELETE {
		server.DeleteSessionCh <- sessionMgmt
	}
	return nil
}

func (server *BFDServer) StartSessionHandler() error {
	server.CreateSessionCh = make(chan BfdSessionMgmt)
	server.DeleteSessionCh = make(chan BfdSessionMgmt)
	for {
		select {
		case sessionMgmt := <-server.CreateSessionCh:
			session, err := server.CreateBfdSession(sessionMgmt)
			if err == nil {
				session.TxTimeoutCh = make(chan *BfdSession)
				session.SessionTimeoutCh = make(chan *BfdSession)
				session.SessionDeleteCh = make(chan bool)
				go session.StartSessionClient()
				go session.StartSessionServer()
			}
		case sessionMgmt := <-server.DeleteSessionCh:
			server.DeleteBfdSession(sessionMgmt)
		}
	}
	return nil
}

func (server *BFDServer) GetNewSessionId() int32 {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	sessionId := r1.Int31n(MAX_NUM_SESSIONS)
	server.bfdGlobal.SessionsIdSlice = append(server.bfdGlobal.SessionsIdSlice, sessionId)
	return sessionId
}

func (server *BFDServer) GetIfIndexAndLocalIpFromDestIp(DestIp string) (int32, string) {
	reachabilityInfo, err := server.ribdClient.ClientHdl.GetRouteReachabilityInfo(DestIp)
	if err != nil {
		server.logger.Info(fmt.Sprintf("%s is not reachable", DestIp))
		return int32(0), ""
	}
	return int32(reachabilityInfo.NextHopIfIndex), reachabilityInfo.NextHopIp
}

func (server *BFDServer) NewBfdSession(DestIp string, protocol int) *BfdSession {
	bfdSession := &BfdSession{}
	bfdSession.state.SessionId = server.GetNewSessionId()
	bfdSession.state.RemoteIpAddr = DestIp
	ifIndex, localIp := server.GetIfIndexAndLocalIpFromDestIp(DestIp)
	bfdSession.state.LocalIpAddr = localIp
	bfdSession.state.InterfaceId = ifIndex
	bfdSession.state.RegisteredProtocols = make([]bool, bfddCommonDefs.MAX_NUM_PROTOCOLS)
	bfdSession.state.RegisteredProtocols[protocol] = true
	bfdSession.state.SessionState = STATE_DOWN
	bfdSession.state.RemoteSessionState = STATE_DOWN
	bfdSession.state.LocalDiscriminator = uint32(bfdSession.state.SessionId)
	bfdSession.state.LocalDiagType = DIAG_NONE
	intf, exist := server.bfdGlobal.Interfaces[bfdSession.state.InterfaceId]
	if exist {
		bfdSession.state.LocalIpAddr = intf.property.IpAddr.String()
		bfdSession.state.DesiredMinTxInterval = intf.conf.DesiredMinTxInterval
		bfdSession.state.RequiredMinRxInterval = intf.conf.RequiredMinRxInterval
		bfdSession.state.DetectionMultiplier = intf.conf.LocalMultiplier
		bfdSession.state.DemandMode = intf.conf.DemandEnabled
	}
	server.logger.Info(fmt.Sprintln("new sesstion : ", bfdSession))
	return bfdSession
}

func (server *BFDServer) FindBfdSession(DestIp string) (sessionId int32, found bool) {
	found = false
	for sessionId, session := range server.bfdGlobal.Sessions {
		if session.state.RemoteIpAddr == DestIp {
			return sessionId, true
		}
	}
	return sessionId, found
}

func NewBfdControlPacketDefault() *BfdControlPacket {
	bfdControlPacket := &BfdControlPacket{
		Version:    DEFAULT_BFD_VERSION,
		Diagnostic: DIAG_NONE,
		State:      STATE_DOWN,
		Poll:       false,
		Final:      false,
		ControlPlaneIndependent:   false,
		AuthPresent:               false,
		Demand:                    false,
		Multipoint:                false,
		DetectMult:                DEFAULT_DETECT_MULTI,
		MyDiscriminator:           0,
		YourDiscriminator:         0,
		DesiredMinTxInterval:      DEFAULT_DESIRED_MIN_TX_INTERVAL,
		RequiredMinRxInterval:     DEFAULT_REQUIRED_MIN_RX_INTERVAL,
		RequiredMinEchoRxInterval: DEFAULT_REQUIRED_MIN_ECHO_RX_INTERVAL,
		AuthHeader:                nil,
	}
	return bfdControlPacket
}

func (session *BfdSession) UpdateBfdSessionControlPacket() error {
	session.bfdPacket.Diagnostic = session.state.LocalDiagType
	session.bfdPacket.State = session.state.SessionState
	session.bfdPacket.DetectMult = uint8(session.state.DetectionMultiplier)
	session.bfdPacket.MyDiscriminator = session.state.LocalDiscriminator
	session.bfdPacket.YourDiscriminator = session.state.RemoteDiscriminator
	session.bfdPacket.DesiredMinTxInterval = time.Duration(session.state.DesiredMinTxInterval)
	session.bfdPacket.RequiredMinRxInterval = time.Duration(session.state.RequiredMinRxInterval)
	return nil
}

// CreateBfdSession initializes a session and starts cpntrol packets exchange.
// This function is called when a protocol registers with BFD to monitor a destination IP.
func (server *BFDServer) CreateBfdSession(sessionMgmt BfdSessionMgmt) (*BfdSession, error) {
	var bfdSession *BfdSession
	DestIp := sessionMgmt.DestIp
	Protocol := sessionMgmt.Protocol
	sessionId, found := server.FindBfdSession(DestIp)
	if !found {
		server.logger.Info(fmt.Sprintln("CreateSession ", DestIp, Protocol))
		bfdSession = server.NewBfdSession(DestIp, Protocol)
		bfdSession.bfdPacket = NewBfdControlPacketDefault()
		server.bfdGlobal.Sessions[bfdSession.state.SessionId] = bfdSession
		server.logger.Info(fmt.Sprintln("Bfd session created ", bfdSession))
	} else {
		server.logger.Info(fmt.Sprintln("Bfd session already exists ", DestIp, Protocol, sessionId))
		bfdSession = server.bfdGlobal.Sessions[sessionId]
		if !bfdSession.state.RegisteredProtocols[Protocol] {
			bfdSession.state.RegisteredProtocols[Protocol] = true
		}
	}
	return bfdSession, nil
}

// DeleteBfdSession ceases the session.
// A session down control packet is sent to BFD neighbor before deleting the session.
// This function is called when a protocol decides to stop monitoring the destination IP.
func (server *BFDServer) DeleteBfdSession(sessionMgmt BfdSessionMgmt) error {
	DestIp := sessionMgmt.DestIp
	Protocol := sessionMgmt.Protocol
	server.logger.Info(fmt.Sprintln("DeleteSession ", DestIp, Protocol))
	sessionId, found := server.FindBfdSession(DestIp)
	if found {
		server.bfdGlobal.Sessions[sessionId].txTimer.Stop()
		server.bfdGlobal.Sessions[sessionId].sessionTimer.Stop()
		server.bfdGlobal.Sessions[sessionId].SessionDeleteCh <- true
		delete(server.bfdGlobal.Sessions, sessionId)
	} else {
		server.logger.Info(fmt.Sprintln("Bfd session not found ", sessionId))
	}
	return nil
}

// This function handles NextHop change from RIB.
// Subsequent control packets will be sent using the BFD attributes configuration on the new IfIndex.
// A Poll control packet will be sent to BFD neighbor and expact a Final control packet.
func (server *BFDServer) HandleNextHopChange(DestIp string) error {
	return nil
}

/* State Machine
                             +--+
                             |  | UP, ADMIN DOWN, TIMER
                             |  V
                     DOWN  +------+  INIT
              +------------|      |------------+
              |            | DOWN |            |
              |  +-------->|      |<--------+  |
              |  |         +------+         |  |
              |  |                          |  |
              |  |               ADMIN DOWN,|  |
              |  |ADMIN DOWN,          DOWN,|  |
              |  |TIMER                TIMER|  |
              V  |                          |  V
            +------+                      +------+
       +----|      |                      |      |----+
   DOWN|    | INIT |--------------------->|  UP  |    |INIT, UP
       +--->|      | INIT, UP             |      |<---+
            +------+                      +------+
*/
// EventHandler is called after receiving a BFD packet from remote.
func (session *BfdSession) EventHandler(event BfdSessionEvent) error {
	switch session.state.SessionState {
	case STATE_ADMIN_DOWN, STATE_DOWN:
		switch event {
		case REMOTE_DOWN:
			session.MoveToInitState()
		case REMOTE_INIT:
			session.MoveToUpState()
		case ADMIN_DOWN, TIMEOUT, REMOTE_UP:
			fmt.Printf("Received %d event in DOWN state. No change in state", event)
		}
	case STATE_INIT:
		switch event {
		case REMOTE_INIT, REMOTE_UP:
			session.MoveToUpState()
		case ADMIN_DOWN, TIMEOUT:
			session.MoveToDownState()
		case REMOTE_DOWN:
			fmt.Printf("Received %d event in INIT state. No change in state", event)
		}
	case STATE_UP:
		switch event {
		case REMOTE_DOWN, ADMIN_DOWN, TIMEOUT:
			session.MoveToDownState()
		case REMOTE_INIT, REMOTE_UP:
			fmt.Printf("Received %d event in UP state. No change in state", event)
		}
	}
	return nil
}

func (session *BfdSession) MoveToDownState() error {
	session.state.SessionState = STATE_DOWN
	return nil
}

func (session *BfdSession) MoveToInitState() error {
	session.state.SessionState = STATE_INIT
	return nil
}

func (session *BfdSession) MoveToUpState() error {
	session.state.SessionState = STATE_UP
	return nil
}

func (session *BfdSession) StartSessionClient() error {
	destAddr := session.state.RemoteIpAddr + ":" + strconv.Itoa(DEST_PORT)
	ServerAddr, err := net.ResolveUDPAddr("udp", destAddr)
	if err != nil {
		fmt.Println("Failed ResolveUDPAddr ", destAddr, err)
	}
	localAddr := session.state.LocalIpAddr + ":" + strconv.Itoa(SRC_PORT)
	ClientAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		fmt.Println("Failed ResolveUDPAddr ", localAddr, err)
	}
	Conn, err := net.DialUDP("udp", ClientAddr, ServerAddr)
	if err != nil {
		fmt.Println("Failed DialUDP ", ClientAddr, ServerAddr, err)
	}
	sessionTimeoutMS := time.Duration((session.state.RequiredMinRxInterval * session.state.DetectionMultiplier) / 1000)
	txTimerMS := time.Duration(session.state.DesiredMinTxInterval / 1000)
	session.sessionTimer = time.AfterFunc(time.Millisecond*sessionTimeoutMS, func() { session.SessionTimeoutCh <- session })
	session.txTimer = time.AfterFunc(time.Millisecond*txTimerMS, func() { session.TxTimeoutCh <- session })
	defer Conn.Close()
	for {
		select {
		case session := <-session.TxTimeoutCh:
			session.UpdateBfdSessionControlPacket()
			buf, err := session.bfdPacket.CreateBfdControlPacket()
			if err != nil {
				fmt.Println("Failed to create control packet for session ", session.state.SessionId)
			} else {
				_, err = Conn.Write(buf)
				if err != nil {
					fmt.Println("failed to send control packet for session ", session.state.SessionId)
				}
				session.txTimer = time.AfterFunc(time.Millisecond*txTimerMS, func() { session.TxTimeoutCh <- session })
			}
		case session := <-session.SessionTimeoutCh:
			session.EventHandler(TIMEOUT)
			session.sessionTimer = time.AfterFunc(time.Millisecond*sessionTimeoutMS, func() { session.SessionTimeoutCh <- session })
		case <-session.SessionDeleteCh:
			return nil
		}
	}
}

func (session *BfdSession) StartSessionServer() error {
	destAddr := session.state.RemoteIpAddr + ":" + strconv.Itoa(DEST_PORT)
	ServerAddr, err := net.ResolveUDPAddr("udp", destAddr)
	if err != nil {
		fmt.Println("Failed ResolveUDPAddr ", destAddr, err)
	}
	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	if err != nil {
		fmt.Println("Failed ListenUDP ", err)
	}
	defer ServerConn.Close()
	buf := make([]byte, 1024)
	for {
		n, addr, err := ServerConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Failed to read from ", ServerAddr)
		} else {
			bfdPacket, _ := DecodeBfdControlPacket(buf[0:n])
			session.ProcessBfdPacket(bfdPacket)
			fmt.Println("Received ", string(buf[0:n]), " from ", addr, " bfdPacket ", bfdPacket)
		}
	}
	return nil
}

func (session *BfdSession) ProcessBfdPacket(bfdPacket *BfdControlPacket) error {
	var event BfdSessionEvent
	session.state.RemoteSessionState = bfdPacket.State
	session.state.RemoteDiscriminator = bfdPacket.MyDiscriminator
	session.state.RemoteMinRxInterval = int32(bfdPacket.RequiredMinRxInterval)
	if bfdPacket.Demand {
		session.state.RemoteDemandMode = true
	}
	switch session.state.RemoteSessionState {
	case STATE_DOWN:
		event = REMOTE_DOWN
	case STATE_INIT:
		event = REMOTE_INIT
	case STATE_UP:
		event = REMOTE_UP
	}
	session.EventHandler(event)
	return nil
}