package main

import (
	"fmt"

	janus "github.com/notedit/janus-go"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/ice"
)

func watch_handle(handle *janus.Handle) {
	// wait for event
	for {
		msg := <-handle.Events
		switch msg := msg.(type) {
		case *janus.SlowLinkMsg:
			fmt.Print("SlowLinkMsg type ", handle.Id)
		case *janus.MediaMsg:
			fmt.Print("MediaEvent type", msg.Type, " receiving ", msg.Receiving)
		case *janus.WebRTCUpMsg:
			fmt.Print("WebRTCUp type ", handle.Id)
		case *janus.HangupMsg:
			fmt.Print("HangupEvent type ", handle.Id)
		case *janus.EventMsg:
			fmt.Print("EventMsg %+v", msg.Plugindata.Data)
		}

	}

}

func main() {

	webrtc.RegisterDefaultCodecs()

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.New(webrtc.RTCConfiguration{
		ICEServers: []webrtc.RTCICEServer{
			{
				URLs: []string{"stun:101.201.141.179"},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	peerConnection.OnICEConnectionStateChange = func(connectionState ice.ConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	}

	peerConnection.Ontrack = func(track *webrtc.RTCTrack) {
		codec := track.Codec
		fmt.Printf("Track has started, of type %d: %s \n", track.PayloadType, codec.Name)
		for {
			<-track.Packets
			fmt.Print("Got Packet")
		}
	}

	// Janus
	gateway, err := janus.Connect("ws://39.106.248.166:8188/")

	if err != nil {
		panic(err)
	}

	mess, _ := gateway.Info()

	fmt.Print("janus info", mess)

	// Create session
	session, err := gateway.Create()

	if err != nil {
		panic(err)
	}

	// Create handle
	handle, err := session.Attach("janus.plugin.streaming")

	go watch_handle(handle)

	// Get streaming list
	message, err := handle.Request(map[string]interface{}{
		"request": "list",
	})

	if err != nil {
		panic(err)
	}

	fmt.Print("streams ", message.PluginData.Data)

	// Watch the second stream
	msg, err := handle.Message(map[string]interface{}{
		"request": "watch",
		"id":      1,
	}, nil)

	if err != nil {
		fmt.Print("message", msg)
		panic(err)
	}

	if msg.Jsep != nil {
		offer := msg.Jsep["sdp"].(string)

		fmt.Println("offer", offer)

		sdp := webrtc.RTCSessionDescription{
			Type: webrtc.RTCSdpTypeAnswer,
			Sdp:  offer,
		}

		err := peerConnection.SetRemoteDescription(sdp)

		if err != nil {
			panic(err)
		}

		answer, err := peerConnection.CreateAnswer(nil)

		if err != nil {
			panic(err)
		}

		// now we start

		_, err = handle.Message(map[string]interface{}{
			"request": "start",
		}, map[string]string{
			"type": "answer",
			"sdp":  answer.Sdp,
		})

		if err != nil {
			panic(err)
		}

	}

	select {}

}
