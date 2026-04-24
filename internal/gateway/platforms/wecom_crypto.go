// Package platforms provides platform adapter implementations for the gateway.
//
// wecom_crypto.go implements WeCom BizMsgCrypt-compatible AES-CBC encryption
// for callback mode. Uses the same wire format as Tencent's official SDK.
package platforms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// WeComCryptoError is the base error type for WeCom crypto operations.
var (
	ErrWeComSignature = errors.New("wecom: signature mismatch")
	ErrWeComDecrypt   = errors.New("wecom: decrypt failed")
	ErrWeComEncrypt   = errors.New("wecom: encrypt failed")
)

// WXBizMsgCrypt is a minimal WeCom callback crypto helper compatible with
// BizMsgCrypt semantics.
type WXBizMsgCrypt struct {
	token     string
	receiveID string
	key       []byte
	iv        []byte
}

// NewWXBizMsgCrypt creates a new WeCom crypto instance.
func NewWXBizMsgCrypt(token, encodingAESKey, receiveID string) (*WXBizMsgCrypt, error) {
	if token == "" {
		return nil, errors.New("wecom crypto: token is required")
	}
	if encodingAESKey == "" {
		return nil, errors.New("wecom crypto: encoding_aes_key is required")
	}
	if len(encodingAESKey) != 43 {
		return nil, errors.New("wecom crypto: encoding_aes_key must be 43 chars")
	}
	if receiveID == "" {
		return nil, errors.New("wecom crypto: receive_id is required")
	}

	key, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("wecom crypto: invalid encoding_aes_key: %w", err)
	}

	return &WXBizMsgCrypt{
		token:     token,
		receiveID: receiveID,
		key:       key,
		iv:        key[:16],
	}, nil
}

// VerifyURL decrypts the echostr for URL verification.
func (c *WXBizMsgCrypt) VerifyURL(msgSignature, timestamp, nonce, echoStr string) (string, error) {
	plain, err := c.Decrypt(msgSignature, timestamp, nonce, echoStr)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// Decrypt verifies the signature and decrypts the encrypted payload.
func (c *WXBizMsgCrypt) Decrypt(msgSignature, timestamp, nonce, encrypted string) ([]byte, error) {
	expected := sha1Signature(c.token, timestamp, nonce, encrypted)
	if expected != msgSignature {
		return nil, ErrWeComSignature
	}

	cipherText, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64 payload: %v", ErrWeComDecrypt, err)
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWeComDecrypt, err)
	}

	mode := cipher.NewCBCDecrypter(block, c.iv)
	padded := make([]byte, len(cipherText))
	mode.CryptBlocks(padded, cipherText)

	plain, err := pkcs7Decode(padded)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWeComDecrypt, err)
	}

	// Skip 16-byte random prefix.
	content := plain[16:]
	// Read 4-byte network-order length.
	xmlLength := binary.BigEndian.Uint32(content[:4])
	xmlContent := content[4 : 4+xmlLength]
	receivedID := string(content[4+xmlLength:])

	if receivedID != c.receiveID {
		return nil, fmt.Errorf("%w: receive_id mismatch", ErrWeComDecrypt)
	}

	return xmlContent, nil
}

// Encrypt encrypts a plaintext message for WeCom.
func (c *WXBizMsgCrypt) Encrypt(plaintext string) (string, error) {
	nonce := randomNonce(10)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	encrypted, err := c.encryptBytes([]byte(plaintext))
	if err != nil {
		return "", err
	}

	signature := sha1Signature(c.token, timestamp, nonce, encrypted)

	xml := fmt.Sprintf(`<xml><Encrypt><![CDATA[%s]]></Encrypt><MsgSignature><![CDATA[%s]]></MsgSignature><TimeStamp>%s</TimeStamp><Nonce><![CDATA[%s]]></Nonce></xml>`,
		encrypted, signature, timestamp, nonce)
	return xml, nil
}

func (c *WXBizMsgCrypt) encryptBytes(raw []byte) (string, error) {
	randomPrefix := make([]byte, 16)
	if _, err := rand.Read(randomPrefix); err != nil {
		return "", fmt.Errorf("%w: random prefix: %v", ErrWeComEncrypt, err)
	}

	msgLen := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLen, uint32(len(raw)))

	payload := make([]byte, 0, 16+4+len(raw)+len(c.receiveID))
	payload = append(payload, randomPrefix...)
	payload = append(payload, msgLen...)
	payload = append(payload, raw...)
	payload = append(payload, []byte(c.receiveID)...)

	padded := pkcs7Encode(payload, 32)

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrWeComEncrypt, err)
	}

	mode := cipher.NewCBCEncrypter(block, c.iv)
	cipherText := make([]byte, len(padded))
	mode.CryptBlocks(cipherText, padded)

	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func sha1Signature(token, timestamp, nonce, encrypt string) string {
	parts := []string{token, timestamp, nonce, encrypt}
	sort.Strings(parts)
	h := sha1.New()
	h.Write([]byte(strings.Join(parts, "")))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func pkcs7Encode(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	if padding == 0 {
		padding = blockSize
	}
	pad := make([]byte, padding)
	for i := range pad {
		pad[i] = byte(padding)
	}
	return append(data, pad...)
}

func pkcs7Decode(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty decrypted payload")
	}
	pad := int(data[len(data)-1])
	if pad < 1 || pad > 32 {
		return nil, errors.New("invalid PKCS7 padding")
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return nil, errors.New("malformed PKCS7 padding")
		}
	}
	return data[:len(data)-pad], nil
}

func randomNonce(length int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}
