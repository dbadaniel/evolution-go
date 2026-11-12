package whatsmeow

import (
	"reflect"
	"testing"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waE2E"
)

func TestExpertsaExplicitListTransport(t *testing.T) {
	msg := &waE2E.Message{ListMessage: &waE2E.ListMessage{}}
	if got := getTypeFromMessage(msg); got != "text" {
		t.Fatalf("list stanza type = %q, want text", got)
	}
	if got := getMediaTypeFromMessage(msg); got != "list" {
		t.Fatalf("encrypted list media type = %q, want list", got)
	}
	biz := waBinary.Node{Tag: "biz", Attrs: waBinary.Attrs{"actual_actors": "2"}, Content: []waBinary.Node{{Tag: "list", Attrs: waBinary.Attrs{"v": "2", "type": "product_list"}}}}
	nodes := []waBinary.Node{biz}
	cli := &Client{}
	got := cli.getMessageContent(waBinary.Node{Tag: "enc"}, msg, waBinary.Attrs{"type": "text"}, false, nodeExtraParams{additionalNodes: &nodes})
	if len(got) != 2 || !reflect.DeepEqual(got[1], biz) {
		t.Fatalf("explicit list business node duplicated/changed: %#v", got)
	}
	got = cli.getMessageContent(waBinary.Node{Tag: "enc"}, msg, nil, false, nodeExtraParams{})
	if len(got) != 2 || got[1].Tag != "biz" {
		t.Fatalf("default list behavior must remain available: %#v", got)
	}
	interactive := &waE2E.Message{InteractiveMessage: &waE2E.InteractiveMessage{}}
	if getTypeFromMessage(interactive) != "text" {
		t.Fatal("interactive stanza type changed")
	}
	got = cli.getMessageContent(waBinary.Node{Tag: "enc"}, interactive, nil, false, nodeExtraParams{additionalNodes: &nodes})
	if len(got) != 2 || !reflect.DeepEqual(got[1], biz) {
		t.Fatal("native-flow explicit nodes changed")
	}
}
