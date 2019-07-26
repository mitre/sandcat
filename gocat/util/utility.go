package util

import (
	"encoding/base64"
	"encoding/json"
	"os"
)

// Encode base64 encodes bytes
func Encode(b []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(b))
}

// Decode base64 decodes a string
func Decode(s string) []byte {
	raw, _ := base64.StdEncoding.DecodeString(s)
	return raw
}

// Unpack converts bytes into JSON
func Unpack(b []byte) (out map[string]interface{}) {
	_ = json.Unmarshal(b, &out)
	return
}

// Exists checks for a file 
func Exists(path string) bool {
    _, err := os.Stat(path)
    if err == nil { return true }
    if os.IsNotExist(err) { return false }
    return true
}
