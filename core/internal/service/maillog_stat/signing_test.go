package maillog_stat

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignToken_RoundTrip(t *testing.T) {
	token := SignToken(Encrypt(map[string]interface{}{"type": "click", "url": "https://example.com"}))

	payload, signed, err := VerifyToken(token)
	require.NoError(t, err)
	assert.True(t, signed)

	var data map[string]interface{}
	require.NoError(t, Decrypt(payload, &data))
	assert.Equal(t, "https://example.com", data["url"])
}

// Anyone can build a token that decrypts, because key and IV travel inside it.
// The signature is what stops a forged link from redirecting through our domain (#390).
func TestVerifyToken_RejectsForgery(t *testing.T) {
	genuine := SignToken(Encrypt(map[string]interface{}{"type": "click", "url": "https://example.com"}))
	forgedPayload := Encrypt(map[string]interface{}{"type": "click", "url": "https://evil.example"})
	_, sig, _ := strings.Cut(genuine, ".")

	_, _, err := VerifyToken(forgedPayload + "." + sig)
	assert.ErrorIs(t, err, ErrBadTrackingSignature)

	_, _, err = VerifyToken(forgedPayload + ".AAAAAAAAAAAAAAAAAAAAAA")
	assert.ErrorIs(t, err, ErrBadTrackingSignature)
}

func TestVerifyToken_LegacyUnsigned(t *testing.T) {
	legacy := Encrypt(map[string]interface{}{"type": "open"})

	payload, signed, err := VerifyToken(legacy)
	require.NoError(t, err)
	assert.False(t, signed)
	assert.Equal(t, legacy, payload)
}

func TestLeavingPage_EscapesTarget(t *testing.T) {
	page := leavingPage(`https://example.com/"><script>alert(1)</script>`)
	assert.NotContains(t, page, "<script>")
}
