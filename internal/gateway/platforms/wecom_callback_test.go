package platforms

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWecomCallbackAdapter_Defaults(t *testing.T) {
	a := NewWecomCallbackAdapter(map[string]any{
		"corp_id":          "testcorp",
		"corp_secret":      "secret",
		"encoding_aes_key": "aeskey",
	})
	require.NotNil(t, a)
	assert.Equal(t, "wecom_callback", a.AdapterName)
	assert.Equal(t, wecomDefaultHost, a.host)
	assert.Equal(t, wecomDefaultPort, a.port)
	assert.Equal(t, wecomDefaultPath, a.path)
}

func TestNewWecomCallbackAdapter_CustomValues(t *testing.T) {
	a := NewWecomCallbackAdapter(map[string]any{
		"host":             "0.0.0.0",
		"port":             9999,
		"path":             "/custom/path",
		"corp_id":          "testcorp",
		"corp_secret":      "secret",
		"encoding_aes_key": "aeskey",
	})
	require.NotNil(t, a)
	assert.Equal(t, "0.0.0.0", a.host)
	assert.Equal(t, 9999, a.port)
	assert.Equal(t, "/custom/path", a.path)
}

func TestNewWecomCallbackAdapter_PortAsFloat(t *testing.T) {
	a := NewWecomCallbackAdapter(map[string]any{
		"port":    float64(8080),
		"corp_id": "testcorp",
	})
	require.NotNil(t, a)
	assert.Equal(t, 8080, a.port)
}

func TestNewWecomCallbackAdapter_PortAsString(t *testing.T) {
	a := NewWecomCallbackAdapter(map[string]any{
		"port":    "7777",
		"corp_id": "testcorp",
	})
	require.NotNil(t, a)
	assert.Equal(t, 7777, a.port)
}

func TestNewWecomCallbackAdapter_MultipleApps(t *testing.T) {
	apps := []any{
		map[string]any{
			"name":             "app1",
			"corp_id":          "corp1",
			"corp_secret":      "secret1",
			"agent_id":         "1",
			"token":            "token1",
			"encoding_aes_key": "aeskey1",
		},
		map[string]any{
			"name":             "app2",
			"corp_id":          "corp2",
			"corp_secret":      "secret2",
			"agent_id":         "2",
			"token":            "token2",
			"encoding_aes_key": "aeskey2",
		},
	}
	a := NewWecomCallbackAdapter(map[string]any{"apps": apps})
	require.NotNil(t, a)
	assert.Len(t, a.apps, 2)
}

func TestNormalizeWecomApps_Empty(t *testing.T) {
	result := normalizeWecomApps(map[string]any{})
	assert.Nil(t, result)
}

func TestNormalizeWecomApps_SingleApp(t *testing.T) {
	result := normalizeWecomApps(map[string]any{
		"corp_id":     "c1",
		"corp_secret": "s1",
	})
	require.Len(t, result, 1)
	assert.Equal(t, "c1", result[0]["corp_id"])
	assert.Equal(t, "s1", result[0]["corp_secret"])
	assert.Equal(t, "default", result[0]["name"])
}

func TestWecomCallbackAdapter_BuildEvent_TextMessage(t *testing.T) {
	a := NewWecomCallbackAdapter(map[string]any{
		"corp_id": "testcorp",
	})
	app := map[string]string{"corp_id": "testcorp", "name": "app1"}

	xmlText := `<xml>
		<ToUserName>testcorp</ToUserName>
		<FromUserName>user1</FromUserName>
		<MsgType>text</MsgType>
		<Content>hello world</Content>
		<MsgId>msg-001</MsgId>
		<CreateTime>1234567890</CreateTime>
	</xml>`

	event := a.buildEvent(app, xmlText)
	require.NotNil(t, event)
	assert.Equal(t, "hello world", event.Content)
	assert.Contains(t, event.ChatID, "user1")
}

func TestWecomCallbackAdapter_BuildEvent_EventSubscribe(t *testing.T) {
	a := NewWecomCallbackAdapter(map[string]any{"corp_id": "testcorp"})
	app := map[string]string{"corp_id": "testcorp", "name": "app1"}

	xmlText := `<xml>
		<ToUserName>testcorp</ToUserName>
		<FromUserName>user1</FromUserName>
		<MsgType>event</MsgType>
		<Event>subscribe</Event>
		<MsgId>msg-002</MsgId>
		<CreateTime>1234567890</CreateTime>
	</xml>`

	// subscribe events should return nil
	event := a.buildEvent(app, xmlText)
	assert.Nil(t, event)
}

func TestWecomCallbackAdapter_BuildEvent_InvalidXML(t *testing.T) {
	a := NewWecomCallbackAdapter(map[string]any{"corp_id": "testcorp"})
	app := map[string]string{"corp_id": "testcorp", "name": "app1"}

	event := a.buildEvent(app, "not valid xml")
	assert.Nil(t, event)
}

func TestWecomCallbackAdapter_BuildEvent_Deduplication(t *testing.T) {
	a := NewWecomCallbackAdapter(map[string]any{"corp_id": "testcorp"})
	app := map[string]string{"corp_id": "testcorp", "name": "app1"}

	xmlText := `<xml>
		<ToUserName>testcorp</ToUserName>
		<FromUserName>user1</FromUserName>
		<MsgType>text</MsgType>
		<Content>hello</Content>
		<MsgId>dup-msg</MsgId>
		<CreateTime>1234567890</CreateTime>
	</xml>`

	event1 := a.buildEvent(app, xmlText)
	event2 := a.buildEvent(app, xmlText)
	assert.NotNil(t, event1)
	assert.Nil(t, event2) // duplicate
}

func TestWecomCallbackAdapter_ResolveAppForChat_NoMatch(t *testing.T) {
	a := NewWecomCallbackAdapter(map[string]any{
		"corp_id": "testcorp",
	})
	app := a.resolveAppForChat("unknown-chat")
	assert.NotNil(t, app)
}

func TestWecomCallbackAdapter_ResolveAppForChat_WithUserMap(t *testing.T) {
	a := NewWecomCallbackAdapter(map[string]any{
		"apps": []any{
			map[string]any{
				"name":    "myapp",
				"corp_id": "c1",
			},
		},
	})
	a.userAppMap["c1:user123"] = "myapp"

	app := a.resolveAppForChat("c1:user123")
	assert.Equal(t, "myapp", app["name"])
}

func TestUserAppKey(t *testing.T) {
	assert.Equal(t, "corp1:user1", userAppKey("corp1", "user1"))
	assert.Equal(t, "user1", userAppKey("", "user1"))
}

func TestGetStr(t *testing.T) {
	m := map[string]any{"key": "value"}
	assert.Equal(t, "value", getStr(m, "key", "default"))
	assert.Equal(t, "default", getStr(m, "missing", "default"))
}

func TestWecomCallbackXML_Unmarshal(t *testing.T) {
	xmlText := `<xml>
		<Encrypt>enc123</Encrypt>
		<ToUserName>to1</ToUserName>
		<FromUserName>from1</FromUserName>
		<CreateTime>1234567890</CreateTime>
		<MsgType>text</MsgType>
		<Content>hello</Content>
		<MsgId>msg001</MsgId>
	</xml>`

	var msg WecomCallbackXML
	err := xml.Unmarshal([]byte(xmlText), &msg)
	require.NoError(t, err)
	assert.Equal(t, "enc123", msg.Encrypt)
	assert.Equal(t, "to1", msg.ToUserName)
	assert.Equal(t, "from1", msg.FromUserName)
	assert.Equal(t, "text", msg.MsgType)
	assert.Equal(t, "hello", msg.Content)
}
