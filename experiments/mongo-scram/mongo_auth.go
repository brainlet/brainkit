// Package mongoscram implements MongoDB SCRAM-SHA-256 authentication at the
// Go level. This bypasses the JS driver's auth flow which fails in QuickJS
// due to issues in the async iterator chain during the handshake.
//
// The approach: after a GoSocket connects to MongoDB, if credentials are
// present in the URL, Go performs the hello + SCRAM handshake natively.
// The JS driver then sees an already-authenticated connection.
package mongoscram

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"

	"github.com/xdg-go/scram"
)

// MongoAuth performs SCRAM-SHA-256 authentication on a raw TCP connection
// to MongoDB. It sends the hello command, then the saslStart/saslContinue
// commands using the SCRAM protocol.
func MongoAuth(conn net.Conn, username, password, authSource string) error {
	if authSource == "" {
		authSource = "admin"
	}

	// 1. Send hello command
	helloDoc := map[string]any{
		"hello":   1,
		"helloOk": true,
		"client": map[string]any{
			"driver":   map[string]string{"name": "brainkit-go", "version": "0.0.1"},
			"os":       map[string]string{"type": "darwin"},
			"platform": "go",
		},
		"saslSupportedMechs": authSource + "." + username,
	}

	helloResp, err := sendCommand(conn, authSource, helloDoc)
	if err != nil {
		return fmt.Errorf("hello: %w", err)
	}

	ok, _ := helloResp["ok"].(float64)
	if ok != 1 {
		return fmt.Errorf("hello failed: %v", helloResp)
	}

	// 2. SCRAM-SHA-256 authentication
	client, err := scram.SHA256.NewClient(username, password, "")
	if err != nil {
		return fmt.Errorf("scram client: %w", err)
	}
	conv := client.NewConversation()

	// First SCRAM message
	firstMsg, err := conv.Step("")
	if err != nil {
		return fmt.Errorf("scram step1: %w", err)
	}

	saslStartDoc := map[string]any{
		"saslStart":    1,
		"mechanism":    "SCRAM-SHA-256",
		"payload":      bsonBinary([]byte(firstMsg)),
		"autoAuthorize": 1,
		"options":      map[string]any{"skipEmptyExchange": true},
	}

	startResp, err := sendCommand(conn, authSource, saslStartDoc)
	if err != nil {
		return fmt.Errorf("saslStart: %w", err)
	}
	if ok, _ := startResp["ok"].(float64); ok != 1 {
		errmsg, _ := startResp["errmsg"].(string)
		return fmt.Errorf("saslStart rejected: %s", errmsg)
	}

	// Extract server response
	serverPayload := extractBinaryPayload(startResp["payload"])
	conversationId := startResp["conversationId"]

	// Second SCRAM message (client proof)
	secondMsg, err := conv.Step(string(serverPayload))
	if err != nil {
		return fmt.Errorf("scram step2: %w", err)
	}

	saslContinueDoc := map[string]any{
		"saslContinue":  1,
		"conversationId": conversationId,
		"payload":       bsonBinary([]byte(secondMsg)),
	}

	continueResp, err := sendCommand(conn, authSource, saslContinueDoc)
	if err != nil {
		return fmt.Errorf("saslContinue: %w", err)
	}
	if ok, _ := continueResp["ok"].(float64); ok != 1 {
		errmsg, _ := continueResp["errmsg"].(string)
		return fmt.Errorf("saslContinue rejected: %s", errmsg)
	}

	// Verify server signature
	serverFinal := extractBinaryPayload(continueResp["payload"])
	_, err = conv.Step(string(serverFinal))
	if err != nil {
		return fmt.Errorf("scram step3 (verify): %w", err)
	}

	// If done is false, send one more empty saslContinue
	if done, _ := continueResp["done"].(bool); !done {
		finalDoc := map[string]any{
			"saslContinue":  1,
			"conversationId": conversationId,
			"payload":       bsonBinary([]byte{}),
		}
		finalResp, err := sendCommand(conn, authSource, finalDoc)
		if err != nil {
			return fmt.Errorf("saslContinue final: %w", err)
		}
		if ok, _ := finalResp["ok"].(float64); ok != 1 {
			errmsg, _ := finalResp["errmsg"].(string)
			return fmt.Errorf("saslContinue final rejected: %s", errmsg)
		}
	}

	return nil
}

// sendCommand sends an OP_MSG command and reads the response.
// This is a minimal MongoDB wire protocol implementation.
func sendCommand(conn net.Conn, db string, doc map[string]any) (map[string]any, error) {
	doc["$db"] = db

	body, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}

	// Build BSON document from JSON (minimal approach — use raw JSON as extended JSON)
	// Actually, we need proper BSON. Let's use a minimal BSON encoder.
	bsonDoc := jsonToBSON(doc)

	// OP_MSG format:
	// Header: [length(4), requestID(4), responseTo(4), opcode=2013(4)]
	// flagBits(4) + sections: kind=0(1) + BSON document
	sectionLen := 1 + len(bsonDoc) // kind byte + BSON
	msgLen := 16 + 4 + sectionLen  // header + flags + section

	buf := make([]byte, msgLen)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(msgLen))
	binary.LittleEndian.PutUint32(buf[4:8], 1)    // requestID
	binary.LittleEndian.PutUint32(buf[8:12], 0)   // responseTo
	binary.LittleEndian.PutUint32(buf[12:16], 2013) // OP_MSG
	binary.LittleEndian.PutUint32(buf[16:20], 0)   // flagBits
	buf[20] = 0 // section kind 0 (body)
	copy(buf[21:], bsonDoc)

	if _, err := conn.Write(buf); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	_ = body // unused, we use BSON directly

	// Read response
	header := make([]byte, 4)
	if _, err := readFull(conn, header); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	respLen := binary.LittleEndian.Uint32(header)
	if respLen < 16 || respLen > 1024*1024 {
		return nil, fmt.Errorf("invalid response length: %d", respLen)
	}

	resp := make([]byte, respLen)
	copy(resp[:4], header)
	if _, err := readFull(conn, resp[4:]); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Parse OP_MSG response: skip header(16) + flags(4) + kind(1) → BSON document
	if len(resp) < 21 {
		return nil, fmt.Errorf("response too short: %d", len(resp))
	}
	bsonBody := resp[21:]
	return bsonToJSON(bsonBody)
}

func readFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// Minimal BSON helpers — just enough for SCRAM commands
// For a proper implementation, use go.mongodb.org/mongo-driver/bson

type bsonBinaryType struct {
	data []byte
}

func bsonBinary(data []byte) bsonBinaryType {
	return bsonBinaryType{data: data}
}

// jsonToBSON converts a Go map to minimal BSON encoding
func jsonToBSON(doc map[string]any) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0, 0, 0, 0}) // placeholder for length

	for key, val := range doc {
		switch v := val.(type) {
		case int:
			buf.WriteByte(0x10) // int32
			buf.WriteString(key)
			buf.WriteByte(0)
			var ib [4]byte
			binary.LittleEndian.PutUint32(ib[:], uint32(v))
			buf.Write(ib[:])
		case float64:
			buf.WriteByte(0x01) // double
			buf.WriteString(key)
			buf.WriteByte(0)
			var fb [8]byte
			binary.LittleEndian.PutUint64(fb[:], uint64(v))
			buf.Write(fb[:])
		case string:
			buf.WriteByte(0x02) // string
			buf.WriteString(key)
			buf.WriteByte(0)
			bstr := []byte(v)
			var sl [4]byte
			binary.LittleEndian.PutUint32(sl[:], uint32(len(bstr)+1))
			buf.Write(sl[:])
			buf.Write(bstr)
			buf.WriteByte(0)
		case bool:
			buf.WriteByte(0x08) // boolean
			buf.WriteString(key)
			buf.WriteByte(0)
			if v {
				buf.WriteByte(1)
			} else {
				buf.WriteByte(0)
			}
		case map[string]any:
			buf.WriteByte(0x03) // document
			buf.WriteString(key)
			buf.WriteByte(0)
			sub := jsonToBSON(v)
			buf.Write(sub)
		case map[string]string:
			m := make(map[string]any, len(v))
			for kk, vv := range v {
				m[kk] = vv
			}
			buf.WriteByte(0x03)
			buf.WriteString(key)
			buf.WriteByte(0)
			sub := jsonToBSON(m)
			buf.Write(sub)
		case bsonBinaryType:
			buf.WriteByte(0x05) // binary
			buf.WriteString(key)
			buf.WriteByte(0)
			var bl [4]byte
			binary.LittleEndian.PutUint32(bl[:], uint32(len(v.data)))
			buf.Write(bl[:])
			buf.WriteByte(0x00) // subtype generic
			buf.Write(v.data)
		}
	}
	buf.WriteByte(0) // terminator

	// Write length
	b := buf.Bytes()
	binary.LittleEndian.PutUint32(b[:4], uint32(len(b)))
	return b
}

// bsonToJSON parses minimal BSON into a Go map
func bsonToJSON(data []byte) (map[string]any, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("bson too short: %d", len(data))
	}
	docLen := binary.LittleEndian.Uint32(data[:4])
	if int(docLen) > len(data) {
		return nil, fmt.Errorf("bson length mismatch: %d > %d", docLen, len(data))
	}
	data = data[:docLen]
	pos := 4
	result := make(map[string]any)

	for pos < len(data)-1 {
		if data[pos] == 0 {
			break
		}
		elemType := data[pos]
		pos++
		// Read key (cstring)
		keyEnd := bytes.IndexByte(data[pos:], 0)
		if keyEnd < 0 {
			break
		}
		key := string(data[pos : pos+keyEnd])
		pos += keyEnd + 1

		switch elemType {
		case 0x01: // double
			if pos+8 > len(data) {
				break
			}
			bits := binary.LittleEndian.Uint64(data[pos : pos+8])
			result[key] = float64(bits)
			pos += 8
		case 0x02: // string
			if pos+4 > len(data) {
				break
			}
			sl := binary.LittleEndian.Uint32(data[pos : pos+4])
			pos += 4
			if pos+int(sl) > len(data) {
				break
			}
			result[key] = string(data[pos : pos+int(sl)-1])
			pos += int(sl)
		case 0x03: // document
			sub, err := bsonToJSON(data[pos:])
			if err != nil {
				return nil, err
			}
			subLen := binary.LittleEndian.Uint32(data[pos : pos+4])
			pos += int(subLen)
			result[key] = sub
		case 0x05: // binary
			if pos+5 > len(data) {
				break
			}
			bl := binary.LittleEndian.Uint32(data[pos : pos+4])
			pos += 4
			_ = data[pos] // subtype
			pos++
			if pos+int(bl) > len(data) {
				break
			}
			result[key] = data[pos : pos+int(bl)]
			pos += int(bl)
		case 0x08: // boolean
			result[key] = data[pos] != 0
			pos++
		case 0x09: // datetime (int64)
			if pos+8 > len(data) {
				break
			}
			pos += 8
		case 0x10: // int32
			if pos+4 > len(data) {
				break
			}
			result[key] = float64(int32(binary.LittleEndian.Uint32(data[pos : pos+4])))
			pos += 4
		case 0x12: // int64
			if pos+8 > len(data) {
				break
			}
			result[key] = float64(int64(binary.LittleEndian.Uint64(data[pos : pos+8])))
			pos += 8
		default:
			return result, nil // stop at unknown type
		}
	}
	return result, nil
}

func extractBinaryPayload(v any) []byte {
	switch p := v.(type) {
	case []byte:
		return p
	case string:
		return []byte(p)
	default:
		return nil
	}
}
