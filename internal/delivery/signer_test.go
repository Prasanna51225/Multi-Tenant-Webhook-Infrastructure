package delivery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSign(t *testing.T) {
	payload := []byte(`{"payment_id":"pay_123","amount":9999}`)
	secret := "whsec_testsecret123"
	timestamp := int64(1492774577)

	sig := Sign(payload, secret, timestamp)

	assert.Equal(t, timestamp, sig.Timestamp)
	assert.NotEmpty(t, sig.V1)
	assert.Len(t, sig.V1, 64)
}

func TestSign_String(t *testing.T) {
	sig := Signature{Timestamp: 1492774577, V1: "abc123"}
	assert.Equal(t, "t=1492774577,v1=abc123", sig.String())
}

func TestSign_Deterministic(t *testing.T) {
	payload := []byte(`{"test":true}`)
	secret := "whsec_deterministic"
	timestamp := int64(1000000)

	sig1 := Sign(payload, secret, timestamp)
	sig2 := Sign(payload, secret, timestamp)

	assert.Equal(t, sig1.V1, sig2.V1)
}

func TestSign_DifferentPayloads(t *testing.T) {
	secret := "whsec_different"
	timestamp := int64(1000000)

	sig1 := Sign([]byte(`{"a":1}`), secret, timestamp)
	sig2 := Sign([]byte(`{"a":2}`), secret, timestamp)

	assert.NotEqual(t, sig1.V1, sig2.V1)
}

func TestSign_DifferentSecrets(t *testing.T) {
	payload := []byte(`{"test":true}`)
	timestamp := int64(1000000)

	sig1 := Sign(payload, "whsec_secret1", timestamp)
	sig2 := Sign(payload, "whsec_secret2", timestamp)

	assert.NotEqual(t, sig1.V1, sig2.V1)
}

func TestVerify_Valid(t *testing.T) {
	payload := []byte(`{"payment_id":"pay_123"}`)
	secret := "whsec_verify_test"
	timestamp := int64(1492774577)

	sig := Sign(payload, secret, timestamp)
	assert.True(t, Verify(payload, secret, sig))
}

func TestVerify_InvalidPayload(t *testing.T) {
	payload := []byte(`{"payment_id":"pay_123"}`)
	secret := "whsec_verify_test"
	timestamp := int64(1492774577)

	sig := Sign(payload, secret, timestamp)
	assert.False(t, Verify([]byte(`{"payment_id":"pay_456"}`), secret, sig))
}

func TestVerify_InvalidSecret(t *testing.T) {
	payload := []byte(`{"payment_id":"pay_123"}`)
	timestamp := int64(1492774577)

	sig := Sign(payload, "whsec_correct", timestamp)
	assert.False(t, Verify(payload, "whsec_wrong", sig))
}

func TestParseSignature(t *testing.T) {
	header := "t=1492774577,v1=5257a869e7ecebeda32affa62cdca3fa51cad7e77a0e56ff536d0ce8e108d8bd"

	sig, err := ParseSignature(header)
	require.NoError(t, err)
	assert.Equal(t, int64(1492774577), sig.Timestamp)
	assert.Equal(t, "5257a869e7ecebeda32affa62cdca3fa51cad7e77a0e56ff536d0ce8e108d8bd", sig.V1)
}

func TestParseSignature_Invalid(t *testing.T) {
	_, err := ParseSignature("invalid")
	assert.Error(t, err)
}

func TestParseSignature_Empty(t *testing.T) {
	_, err := ParseSignature("")
	assert.Error(t, err)
}

func TestSignAndVerify_RoundTrip(t *testing.T) {
	payload := []byte(`{"event":"test","data":{"id":1}}`)
	secret := "whsec_roundtrip_secret"
	timestamp := int64(1700000000)

	sig := Sign(payload, secret, timestamp)

	parsed, err := ParseSignature(sig.String())
	require.NoError(t, err)

	assert.True(t, Verify(payload, secret, parsed))
}
