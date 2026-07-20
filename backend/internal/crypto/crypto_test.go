package crypto

import (
	"bytes"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	if len(kp.PublicKey) != 32 {
		t.Errorf("expected public key length 32, got %d", len(kp.PublicKey))
	}
	if len(kp.PrivateKey) != 64 {
		t.Errorf("expected private key length 64, got %d", len(kp.PrivateKey))
	}
}

func TestSignAndVerify(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	message := []byte("test message for signing")
	sig := SignMessage(kp.PrivateKey, message)
	if len(sig) == 0 {
		t.Fatal("signature is empty")
	}

	if !VerifySignature(kp.PublicKey, message, sig) {
		t.Fatal("signature verification failed")
	}

	// Verify with wrong key
	kp2, _ := GenerateKeyPair()
	if VerifySignature(kp2.PublicKey, message, sig) {
		t.Fatal("signature should not verify with wrong key")
	}

	// Verify with tampered message
	if VerifySignature(kp.PublicKey, []byte("tampered message"), sig) {
		t.Fatal("signature should not verify with tampered message")
	}
}

func TestAESEncryptDecrypt(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey failed: %v", err)
	}

	plaintext := []byte("sensitive data for encryption test")

	ciphertext, err := EncryptAES(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAES failed: %v", err)
	}

	if bytes.Equal(plaintext, ciphertext) {
		t.Fatal("ciphertext should not equal plaintext")
	}

	decrypted, err := DecryptAES(ciphertext, key)
	if err != nil {
		t.Fatalf("DecryptAES failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("decrypted text does not match original")
	}

	// Decrypt with wrong key
	wrongKey, _ := GenerateAESKey()
	_, err = DecryptAES(ciphertext, wrongKey)
	if err == nil {
		t.Fatal("decryption should fail with wrong key")
	}
}

func TestHashContent(t *testing.T) {
	content := []byte("hello world")
	hash1 := HashContent(content)
	hash2 := HashContent(content)
	hash3 := HashContent([]byte("different content"))

	if hash1 != hash2 {
		t.Fatal("same content should produce same hash")
	}

	if hash1 == hash3 {
		t.Fatal("different content should produce different hash")
	}

	if len(hash1) != 64 {
		t.Errorf("expected hash length 64, got %d", len(hash1))
	}
}

func TestDeriveSessionKey(t *testing.T) {
	kp1, _ := GenerateKeyPair()
	kp2, _ := GenerateKeyPair()

	key1, err := DeriveSessionKey(kp1.PrivateKey, kp2.PublicKey)
	if err != nil {
		t.Fatalf("DeriveSessionKey failed: %v", err)
	}

	key2, err := DeriveSessionKey(kp2.PrivateKey, kp1.PublicKey)
	if err != nil {
		t.Fatalf("DeriveSessionKey failed: %v", err)
	}

	if !bytes.Equal(key1, key2) {
		t.Fatal("both parties should derive the same session key")
	}

	if len(key1) != 32 {
		t.Errorf("expected session key length 32, got %d", len(key1))
	}
}

func TestKeyPairRoundTrip(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	message := []byte("round trip test")
	sig := SignMessage(kp.PrivateKey, message)

	// Encrypt message
	aesKey, _ := GenerateAESKey()
	ciphertext, _ := EncryptAES(message, aesKey)

	// Decrypt
	decrypted, _ := DecryptAES(ciphertext, aesKey)

	if !bytes.Equal(message, decrypted) {
		t.Fatal("AES round trip failed")
	}

	if !VerifySignature(kp.PublicKey, message, sig) {
		t.Fatal("signature verification round trip failed")
	}
}