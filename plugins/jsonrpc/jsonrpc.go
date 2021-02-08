// +build plugin

package main

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strings"

	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/v3/pkg/plugin"
)

// https://medium.com/better-programming/rpc-in-golang-19661033942
var client *rpc.Client

var unsupported = make(map[string]bool)

func cantFindMethodErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "rpc: can't find method")
}

func call(method string, req interface{}, resp interface{}) error {
	if unsupported[method] {
		return nil
	}
	if client == nil {
		c, err := jsonrpc.Dial("tcp", "localhost:12345")
		if err != nil {
			return err
		}
		client = c
	}
	log.Infof("calling %q", method)
	err := client.Call(method, req, resp)
	log.Infof("returned: %v, %v", resp, err)
	if cantFindMethodErr(err) {
		unsupported[method] = true
		return nil
	}
	return err
}

func Init(req plugin.InitReq, resp *plugin.InitResp) error {
	return call("init", req, resp)
}

func ExecuteNode(req plugin.ExecuteNodeReq, resp *wfv1.NodeStatus) error {
	return call("executeNode", req, resp)
}

func ReconcileNode(req plugin.ReconcileNodeReq, resp *wfv1.NodeStatus) error {
	return call("reconcileNode", req, resp)
}

var notify func(namespace, workflowName string)

type Plugin struct{}

func (p *Plugin) Notify(req plugin.NotifyReq, resp *plugin.NotifyResp) error {
	notify(req.Namespace, req.Name)
	return nil
}

func Run(req plugin.RunReq) {
	notify = req.Notify
	addy, err := net.ResolveTCPAddr("tcp", "0.0.0.0:54321")
	if err != nil {
		panic(err)
	}
	inbound, err := net.ListenTCP("tcp", addy)
	if err != nil {
		panic(err)
	}
	if err := rpc.Register(new(Plugin)); err != nil {
		panic(err)
	}
	for {
		select {
		case <-req.Done:
			return
		default:
			conn, err := inbound.Accept()
			if err != nil {
				println(err.Error())
				continue
			}
			jsonrpc.ServeConn(conn)
		}
	}
}

func main() {
	err := Init(plugin.InitReq{}, &plugin.InitResp{})
	if err != nil {
		panic(err)
	}
}
