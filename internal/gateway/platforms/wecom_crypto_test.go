package platforms

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewWXBizMsgCrypt ---

func TestNewWXBizMsgCrypt_Valid(t *testing.T) {
	// A valid 43-char base64 key (32 bytes when decoded with trailing =).
	key := "keB3Oa4aRbgdy1TUYH49LtcQJGp0BbcD7k6SP8cqBCQ"
	crypt, err := NewWXBizMsgCrypt("token", key, "corp-id")
	require.NoError(t, err)
	require.NotNil(t, crypt)
	assert.Equal(t, "token", crypt.token)
	assert.Equal(t, "corp-id", crypt.receiveID)
	assert.Len(t, crypt.key, 32)
	assert.Len(t, crypt.iv, 16)
}

func TestNewWXBizMsgCrypt_EmptyToken(t *testing.T) {
	_, err := NewWXBizMsgCrypt("", "keB3Oa4aRbgdy1TUYH49LtcQJGp0BbcD7k6SP8cqBCQ", "corp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is required")
}

func TestNewWXBizMsgCrypt_EmptyKey(t *testing.T) {
	_, err := NewWXBizMsgCrypt("token", "", "corp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encoding_aes_key is required")
}

func TestNewWXBizMsgCrypt_WrongKeyLength(t *testing.T) {
	_, err := NewWXBizMsgCrypt("token", "short", "corp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "43 chars")
}

func TestNewWXBizMsgCrypt_EmptyReceiveID(t *testing.T) {
	_, err := NewWXBizMsgCrypt("token", "keB3Oa4aRbgdy1TUYH49LtcQJGp0BbcD7k6SP8cqBCQ", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "receive_id is required")
}

// --- pkcs7Encode/Decode round-trip ---

func TestPKCS7_RoundTrip(t *testing.T) {
	data := []byte("hello world test data")
	encoded := pkcs7Encode(data, 32)
	assert.Greater(t, len(encoded), len(data))
	assert.Equal(t, 0, len(encoded)%32)

	decoded, err := pkcs7Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

func TestPKCS7Decode_Empty(t *testing.T) {
	_, err := pkcs7Decode(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestPKCS7Encode_ExactMultiple(t *testing.T) {
	// When data is exact multiple of block size, a full block of padding is added.
	data := make([]byte, 32)
	encoded := pkcs7Encode(data, 32)
	assert.Len(t, encoded, 64)
}

// --- sha1Signature ---

func TestSha1Signature_Deterministic(t *testing.T) {
	sig1 := sha1Signature("token", "123", "nonce", "data")
	sig2 := sha1Signature("token", "123", "nonce", "data")
	assert.Equal(t, sig1, sig2)
}

func TestSha1Signature_OrderIndependent(t *testing.T) {
	// sha1Signature sorts parts before hashing.
	sig1 := sha1Signature("a", "b", "c", "d")
	sig2 := sha1Signature("d", "c", "b", "a")
	assert.Equal(t, sig1, sig2)
}

func TestSha1Signature_DifferentInputs(t *testing.T) {
	sig1 := sha1Signature("token1", "123", "nonce", "data")
	sig2 := sha1Signature("token2", "123", "nonce", "data")
	assert.NotEqual(t, sig1, sig2)
}

// --- randomNonce ---

func TestRandomNonce_Length(t *testing.T) {
	nonce := randomNonce(16)
	assert.Len(t, nonce, 16)
}

func TestRandomNonce_Uniqueness(t *testing.T) {
	nonces := make(map[string]bool)
	for i := 0; i < 50; i++ {
		n := randomNonce(16)
		assert.False(t, nonces[n])
		nonces[n] = true
	}
}

// --- Encrypt/Decrypt round-trip ---

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := "keB3Oa4aRbgdy1TUYH49LtcQJGp0BbcD7k6SP8cqBCQ"
	crypt, err := NewWXBizMsgCrypt("mytoken", key, "corp123")
	require.NoError(t, err)

	plaintext := "Hello, WeCom!"
	encrypted, err := crypt.encryptBytes([]byte(plaintext))
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	// Build a signature for decryption.
	timestamp := "1234567890"
	nonce := "testnonce"
	signature := sha1Signature("mytoken", timestamp, nonce, encrypted)

	decrypted, err := crypt.Decrypt(signature, timestamp, nonce, encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, string(decrypted))
}

func TestDecrypt_SignatureMismatch(t *testing.T) {
	key := "keB3Oa4aRbgdy1TUYH49LtcQJGp0BbcD7k6SP8cqBCQ"
	crypt, err := NewWXBizMsgCrypt("mytoken", key, "corp123")
	require.NoError(t, err)

	encrypted, err := crypt.encryptBytes([]byte("test"))
	require.NoError(t, err)

	_, err = crypt.Decrypt("wrong-signature", "123", "nonce", encrypted)
	assert.ErrorIs(t, err, ErrWeComSignature)
}
