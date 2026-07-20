package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
)

// KeyPair holds the Ed25519 key pair for node identity and signing
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// GenerateKeyPair generates a new Ed25519 key pair
func GenerateKeyPair() (*KeyPair, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}, nil
}

// GenerateAESKey generates a new 256-bit AES key
func GenerateAESKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptAES encrypts plaintext using AES-256-GCM
func EncryptAES(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptAES decrypts ciphertext using AES-256-GCM
func DecryptAES(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// SignMessage signs a message with the private key using Ed25519
func SignMessage(privateKey ed25519.PrivateKey, message []byte) []byte {
	return ed25519.Sign(privateKey, message)
}

// VerifySignature verifies a message signature using the public key
func VerifySignature(publicKey ed25519.PublicKey, message []byte, sig []byte) bool {
	return ed25519.Verify(publicKey, message, sig)
}

// HashContent returns SHA-256 hash of content
func HashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// DeriveSessionKey derives a shared AES key from two key pairs using SHA-256
// Both parties derive the same key by sorting public keys before combining
func DeriveSessionKey(localPrivKey ed25519.PrivateKey, remotePubKey ed25519.PublicKey) ([]byte, error) {
	localPub := localPrivKey.Public().(ed25519.PublicKey)

	// Sort the two public keys lexicographically so both parties derive the same key
	var combined []byte
	if compareBytes(localPub, remotePubKey) <= 0 {
		combined = append(localPub, remotePubKey...)
	} else {
		combined = append(remotePubKey, localPub...)
	}

	hash := sha256.Sum256(combined)
	return hash[:], nil
}

// compareBytes compares two byte slices lexicographically
func compareBytes(a, b []byte) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}
