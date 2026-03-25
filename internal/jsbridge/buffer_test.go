package jsbridge

import "testing"

func TestBuffer_FromString(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.from("hello");
		JSON.stringify({
			isBuffer: Buffer.isBuffer(buf),
			length: buf.length,
			utf8: buf.toString("utf8"),
			hex: buf.toString("hex"),
			base64: buf.toString("base64"),
		});
	`)
	expected := `{"isBuffer":true,"length":5,"utf8":"hello","hex":"68656c6c6f","base64":"aGVsbG8="}`
	if result != expected {
		t.Errorf("got %s\nwant %s", result, expected)
	}
}

func TestBuffer_FromArray(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.from([0x48, 0x69]);
		JSON.stringify({ str: buf.toString("utf8"), len: buf.length, isBuffer: Buffer.isBuffer(buf) });
	`)
	expected := `{"str":"Hi","len":2,"isBuffer":true}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}

func TestBuffer_FromBase64(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.from("aGVsbG8=", "base64");
		buf.toString("utf8");
	`)
	if result != "hello" {
		t.Errorf("got %s", result)
	}
}

func TestBuffer_FromHex(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.from("68656c6c6f", "hex");
		buf.toString("utf8");
	`)
	if result != "hello" {
		t.Errorf("got %s", result)
	}
}

func TestBuffer_Alloc(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.alloc(4);
		JSON.stringify({ len: buf.length, zero: buf[0] === 0 && buf[3] === 0, isBuffer: Buffer.isBuffer(buf) });
	`)
	expected := `{"len":4,"zero":true,"isBuffer":true}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}

func TestBuffer_Concat(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var a = Buffer.from("hel");
		var b = Buffer.from("lo");
		var c = Buffer.concat([a, b]);
		JSON.stringify({ str: c.toString("utf8"), len: c.length, isBuffer: Buffer.isBuffer(c) });
	`)
	expected := `{"str":"hello","len":5,"isBuffer":true}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}

func TestBuffer_Int32BE(t *testing.T) {
	// pg wire protocol uses big-endian
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.alloc(4);
		buf.writeInt32BE(0x01020304, 0);
		buf.readInt32BE(0).toString();
	`)
	if result != "16909060" {
		t.Errorf("got %s, want 16909060", result)
	}
}

func TestBuffer_Int32LE(t *testing.T) {
	// MongoDB BSON uses little-endian
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.alloc(4);
		buf.writeInt32LE(0x01020304, 0);
		buf.readInt32LE(0).toString();
	`)
	if result != "16909060" {
		t.Errorf("got %s, want 16909060", result)
	}
}

func TestBuffer_Int16(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.alloc(4);
		buf.writeInt16BE(0x0102, 0);
		buf.writeInt16LE(0x0304, 2);
		JSON.stringify({
			be: buf.readInt16BE(0),
			le: buf.readInt16LE(2),
			ube: buf.readUInt16BE(0),
			ule: buf.readUInt16LE(2),
		});
	`)
	expected := `{"be":258,"le":772,"ube":258,"ule":772}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}

func TestBuffer_Float(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.alloc(12);
		buf.writeFloatLE(3.14, 0);
		buf.writeDoubleLE(2.718, 4);
		JSON.stringify({
			f: Math.abs(buf.readFloatLE(0) - 3.14) < 0.001,
			d: Math.abs(buf.readDoubleLE(4) - 2.718) < 0.001,
		});
	`)
	expected := `{"f":true,"d":true}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}

func TestBuffer_SliceAndSubarray(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.from("hello world");
		var sliced = buf.slice(0, 5);
		JSON.stringify({
			str: sliced.toString("utf8"),
			isBuffer: Buffer.isBuffer(sliced),
			len: sliced.length,
		});
	`)
	expected := `{"str":"hello","isBuffer":true,"len":5}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}

func TestBuffer_Copy(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var src = Buffer.from("hello");
		var dst = Buffer.alloc(3);
		src.copy(dst, 0, 1, 4);
		dst.toString("utf8");
	`)
	if result != "ell" {
		t.Errorf("got %s", result)
	}
}

func TestBuffer_Equals(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var a = Buffer.from("abc");
		var b = Buffer.from("abc");
		var c = Buffer.from("xyz");
		JSON.stringify({ eq: a.equals(b), neq: a.equals(c) });
	`)
	expected := `{"eq":true,"neq":false}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}

func TestBuffer_Base64_WithNullBytes(t *testing.T) {
	// Crypto keys contain null bytes — base64 must not truncate
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.from([0, 1, 0, 2, 0, 3]);
		var b64 = buf.toString("base64");
		var roundtrip = Buffer.from(b64, "base64");
		JSON.stringify({ b64: b64, len: roundtrip.length, match: buf.equals(roundtrip) });
	`)
	expected := `{"b64":"AAEAAgAD","len":6,"match":true}`
	if result != expected {
		t.Errorf("got %s\nwant %s", result, expected)
	}
}

func TestBuffer_Fill(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.alloc(4);
		buf.fill(0xff);
		JSON.stringify([buf[0], buf[1], buf[2], buf[3]]);
	`)
	if result != "[255,255,255,255]" {
		t.Errorf("got %s", result)
	}
}

func TestBuffer_Write(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.alloc(10);
		buf.write("hello", 0);
		buf.toString("utf8", 0, 5);
	`)
	if result != "hello" {
		t.Errorf("got %s", result)
	}
}

func TestBuffer_ToJSON(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer())
	result := evalString(t, b, `
		var buf = Buffer.from([1, 2, 3]);
		var j = buf.toJSON();
		JSON.stringify({ type: j.type, data: j.data });
	`)
	expected := `{"type":"Buffer","data":[1,2,3]}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}
