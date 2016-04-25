package server

import (
	"fmt"
)

func (server *BFDServer) createDefaultSessionParam() error {
	paramName := "default"
	sessionParam, exist := server.bfdGlobal.SessionParams[paramName]
	if !exist {
		sessionParam.state.Name = paramName
		sessionParam.state.LocalMultiplier = DEFAULT_DETECT_MULTI
		sessionParam.state.DesiredMinTxInterval = DEFAULT_DESIRED_MIN_TX_INTERVAL
		sessionParam.state.RequiredMinRxInterval = DEFAULT_REQUIRED_MIN_RX_INTERVAL
		sessionParam.state.RequiredMinEchoRxInterval = DEFAULT_REQUIRED_MIN_ECHO_RX_INTERVAL
		sessionParam.state.DemandEnabled = false
		sessionParam.state.AuthenticationEnabled = false
		server.UpdateBfdSessionsUsingParam(sessionParam.state.Name)
	}
	server.logger.Info(fmt.Sprintln("Created default session param"))
	return nil
}

func (server *BFDServer) processSessionParamConfig(paramConfig SessionParamConfig) error {
	sessionParam, exist := server.bfdGlobal.SessionParams[paramConfig.Name]
	if !exist {
		server.logger.Info(fmt.Sprintln("Creating session param: ", paramConfig.Name))
	} else {
		server.logger.Info(fmt.Sprintln("Updating session param: ", paramConfig.Name))
	}
	sessionParam.state.Name = paramConfig.Name
	sessionParam.state.LocalMultiplier = paramConfig.LocalMultiplier
	sessionParam.state.DesiredMinTxInterval = paramConfig.DesiredMinTxInterval * 1000
	sessionParam.state.RequiredMinRxInterval = paramConfig.RequiredMinRxInterval * 1000
	sessionParam.state.RequiredMinEchoRxInterval = paramConfig.RequiredMinEchoRxInterval * 1000
	sessionParam.state.DemandEnabled = paramConfig.DemandEnabled
	sessionParam.state.AuthenticationEnabled = paramConfig.AuthenticationEnabled
	sessionParam.state.AuthenticationType = paramConfig.AuthenticationType
	sessionParam.state.AuthenticationKeyId = paramConfig.AuthenticationKeyId
	sessionParam.state.AuthenticationData = paramConfig.AuthenticationData
	server.UpdateBfdSessionsUsingParam(sessionParam.state.Name)
	return nil
}

func (server *BFDServer) processSessionParamDelete(paramName string) error {
	_, exist := server.bfdGlobal.SessionParams[paramName]
	if exist {
		server.logger.Info(fmt.Sprintln("Deleting session param: ", paramName))
		delete(server.bfdGlobal.SessionParams, paramName)
		server.UpdateBfdSessionsUsingParam(paramName)
	}
	return nil
}
