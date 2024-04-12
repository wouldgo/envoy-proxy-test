package main

import (
	"fmt"
	"log"
	"net"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/network/source/go/pkg/network"
)

const (
	configServerAddr = "server_addr"
	configServerPort = "server_port"
)

var (
	_ network.ConfigFactory = (*configFactory)(nil)
	_ network.FilterFactory = (*filterFactory)(nil)
	_ api.DownstreamFilter  = (*downFilter)(nil)

	cf = &configFactory{}
)

type configFactory struct{}

func (f *configFactory) CreateFactoryFromConfig(config interface{}) network.FilterFactory {
	configMessage, ok := config.(*anypb.Any)
	if !ok {
		log.Fatalf("failed to load configuration: %+v", config)
		return nil
	}
	configStruct := &xds.TypedStruct{}
	err := configMessage.UnmarshalTo(configStruct)
	if err != nil {
		log.Fatalf("failed to read configuration %+v", configMessage)
		return nil
	}
	configurationMap := configStruct.Value.AsMap()

	serverAddr, ok := configurationMap[configServerAddr]
	if !ok {
		log.Fatalf("failed to read configuration %s property: %+v", configServerAddr, configurationMap)
		return nil
	}

	serverPort, ok := configurationMap[configServerPort]
	if !ok {
		log.Fatalf("failed to read configuration %s property: %+v", configServerPort, configurationMap)
		return nil
	}

	serverAddressStr, ok := serverAddr.(string)
	if !ok {
		log.Fatalf("failed to convert %s into string value", serverAddr)
		return nil
	}

	serverPortStr, ok := serverPort.(string)
	if !ok {
		log.Fatalf("failed to convert %s into string value", serverPort)
	}

	addr, err := net.LookupHost(serverAddressStr)
	if len(addr) == 0 && err != nil {
		fmt.Printf("fail to resolve: %v, err: %v\n", serverAddressStr, err)
		return nil
	}

	choosenAddr := addr[0]
	return &filterFactory{
		upstreamAddr: net.JoinHostPort(choosenAddr, serverPortStr),
	}
}

type filterFactory struct {
	upstreamAddr string
}

func (f *filterFactory) CreateFilter(cb api.ConnectionCallback) api.DownstreamFilter {
	return &downFilter{
		upstreamAddr: f.upstreamAddr,
		cb:           cb,
	}
}

type downFilter struct {
	api.EmptyDownstreamFilter

	cb           api.ConnectionCallback
	upstreamAddr string
	upFilter     *upFilter
}

func (f *downFilter) OnNewConnection() api.FilterStatus {
	localAddr, _ := f.cb.StreamInfo().UpstreamLocalAddress()
	remoteAddr, _ := f.cb.StreamInfo().UpstreamRemoteAddress()
	fmt.Printf("OnNewConnection, local: %v, remote: %v, connect to: %v\n", localAddr, remoteAddr, f.upstreamAddr)
	f.upFilter = &upFilter{
		downFilter: f,
		ch:         make(chan []byte, 1),
	}
	network.CreateUpstreamConn(f.upstreamAddr, f.upFilter)
	return api.NetworkFilterContinue
}

func (f *downFilter) OnData(buffer []byte, endOfStream bool) api.FilterStatus {
	remoteAddr, _ := f.cb.StreamInfo().UpstreamRemoteAddress()
	fmt.Printf("OnData, addr: %v, buffer: %v, endOfStream: %v\n", remoteAddr, string(buffer), endOfStream)
	f.upFilter.ch <- buffer
	return api.NetworkFilterContinue
}

func (f *downFilter) OnEvent(event api.ConnectionEvent) {
	remoteAddr, _ := f.cb.StreamInfo().UpstreamRemoteAddress()
	fmt.Printf("OnEvent, addr: %v, event: %v\n", remoteAddr, event)
}

func (f *downFilter) OnWrite(buffer []byte, endOfStream bool) api.FilterStatus {
	fmt.Printf("OnWrite, buffer: %v, endOfStream: %v\n", string(buffer), endOfStream)
	return api.NetworkFilterContinue
}

type upFilter struct {
	api.EmptyUpstreamFilter

	cb         api.ConnectionCallback
	downFilter *downFilter
	ch         chan []byte
}

func (f *upFilter) OnPoolReady(cb api.ConnectionCallback) {
	f.cb = cb
	localAddr, _ := f.cb.StreamInfo().UpstreamLocalAddress()
	remoteAddr, _ := f.cb.StreamInfo().UpstreamRemoteAddress()
	fmt.Printf("OnPoolReady, local: %v, remote: %v\n", localAddr, remoteAddr)
	go func() {
		for {
			buf, ok := <-f.ch
			if !ok {
				return
			}
			f.cb.Write(buf, false)
		}
	}()
}

func (f *upFilter) OnPoolFailure(poolFailureReason api.PoolFailureReason, transportFailureReason string) {
	fmt.Printf("OnPoolFailure, reason: %v, transportFailureReason: %v\n", poolFailureReason, transportFailureReason)
}

func (f *upFilter) OnData(buffer []byte, endOfStream bool) {
	remoteAddr, _ := f.cb.StreamInfo().UpstreamRemoteAddress()
	fmt.Printf("OnData, addr: %v, buffer: %v, endOfStream: %v\n", remoteAddr, string(buffer), endOfStream)
	f.downFilter.cb.Write(buffer, endOfStream)
}

func (f *upFilter) OnEvent(event api.ConnectionEvent) {
	remoteAddr, _ := f.cb.StreamInfo().UpstreamRemoteAddress()
	fmt.Printf("OnEvent, addr: %v, event: %v\n", remoteAddr, event)
	if event == api.LocalClose || event == api.RemoteClose {
		close(f.ch)
	}
}

func init() {
	network.RegisterNetworkFilterConfigFactory("simple", cf)
}

func main() {}
