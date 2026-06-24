package delivery

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type Signature struct {
	Timestamp int64
	V1        string
}

func (s Signature) String() string {
	return fmt.Sprintf("t=%d,v1=%s", s.Timestamp, s.V1)
}

func Sign(payload []byte, secret string, timestamp int64) Signature {
	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	hash := mac.Sum(nil)

	return Signature{
		Timestamp: timestamp,
		V1:        hex.EncodeToString(hash),
	}
}

func Verify(payload []byte, secret string, sig Signature) bool {
	expected := Sign(payload, secret, sig.Timestamp)
	return hmac.Equal([]byte(expected.V1), []byte(sig.V1))
}

func ParseSignature(header string) (Signature, error) {
	parts := strings.Split(header, ",")
	sig := Signature{}

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		switch kv[0] {
		case "t":
			ts, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return Signature{}, fmt.Errorf("parse timestamp: %w", err)
			}
			sig.Timestamp = ts
		case "v1":
			sig.V1 = kv[1]
		}
	}

	if sig.Timestamp == 0 || sig.V1 == "" {
		return Signature{}, fmt.Errorf("invalid signature format")
	}

	return sig, nil
}
