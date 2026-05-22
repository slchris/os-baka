package api

import (
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/os-baka/backend/internal/model"
	"gorm.io/gorm"
)

const (
	pxeTokenByteLen = 32 // 256 bits of entropy

	// pxeTokenTTLMultiplier sets the default token TTL as a multiple of
	// the install timeout when PXE_TOKEN_TTL_MINUTES isn't explicitly
	// set. 2x means: if a node legitimately uses every minute of its
	// install window, the postinstall callback at the very end still
	// has the full install window again as headroom. Keeps these two
	// knobs in lockstep so operators only have to tune one.
	pxeTokenTTLMultiplier = 2
)

// pxeTokenTTL returns the configured token TTL.
//
// Resolution order:
//   1. PXE_TOKEN_TTL_MINUTES env (explicit operator override)
//   2. installTimeoutFromEnv() * pxeTokenTTLMultiplier (derived default)
//
// The derivation matters: token TTL must comfortably exceed the entire
// install duration, because PostInstall consumes the token at the very
// end of d-i's late_command. A node that hits install timeout right at
// the wire would otherwise have its token expire just as the callback
// fires — install succeeds on the box but never reflects active.
func pxeTokenTTL() time.Duration {
	if v := os.Getenv("PXE_TOKEN_TTL_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Minute
		}
	}
	return time.Duration(installTimeoutFromEnv()*pxeTokenTTLMultiplier) * time.Minute
}

// hashPXEToken returns the hex SHA-256 of the token.
func hashPXEToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// IssuePXEToken mints a fresh provisioning token for a node and invalidates
// any previous active tokens for the same node. Returns the plaintext token —
// the caller is responsible for delivering it exactly once.
func IssuePXEToken(db *gorm.DB, nodeID uint, clientIP string) (string, error) {
	if db == nil {
		return "", errors.New("nil db")
	}

	raw := make([]byte, pxeTokenByteLen)
	if _, err := cryptoRand.Read(raw); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	// URL-safe so it round-trips through iPXE/imgargs without quoting headaches.
	token := base64.RawURLEncoding.EncodeToString(raw)

	now := time.Now()

	// Invalidate prior unconsumed tokens for this node by marking them consumed.
	// We don't delete — keep the audit trail.
	if err := db.Model(&model.PXEProvisioningToken{}).
		Where("node_id = ? AND consumed_at IS NULL", nodeID).
		Update("consumed_at", now).Error; err != nil {
		return "", fmt.Errorf("invalidate prior tokens: %w", err)
	}

	rec := model.PXEProvisioningToken{
		NodeID:    nodeID,
		TokenHash: hashPXEToken(token),
		ClientIP:  clientIP,
		ExpiresAt: now.Add(pxeTokenTTL()),
	}
	if err := db.Create(&rec).Error; err != nil {
		return "", fmt.Errorf("persist token: %w", err)
	}

	slog.Info("PXE token issued",
		"nodeID", nodeID,
		"clientIP", clientIP,
		"expiresAt", rec.ExpiresAt.Format(time.RFC3339))
	return token, nil
}

// ValidatePXEToken checks the token against the node and (optionally) the
// requesting client IP. Returns nil on success.
//
// nodeID may be 0 if the caller doesn't know it yet — in that case we resolve
// the node via the token alone and return the resolved nodeID via the second
// return value. When nodeID is non-zero the token must belong to that node.
//
// requireIP controls whether the request IP must match the IP that minted
// the token. Set false for preseed/postinstall when the installer environment
// (e.g. anaconda using a different egress NIC) may legitimately differ.
func ValidatePXEToken(db *gorm.DB, token string, expectedNodeID uint, clientIP string, requireIP bool) (uint, error) {
	if token == "" {
		return 0, errors.New("missing token")
	}
	if db == nil {
		return 0, errors.New("nil db")
	}

	var rec model.PXEProvisioningToken
	err := db.Where("token_hash = ?", hashPXEToken(token)).First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("unknown token")
		}
		return 0, fmt.Errorf("lookup token: %w", err)
	}

	if rec.ConsumedAt != nil {
		return 0, errors.New("token already consumed")
	}
	if time.Now().After(rec.ExpiresAt) {
		return 0, errors.New("token expired")
	}
	if expectedNodeID != 0 && rec.NodeID != expectedNodeID {
		return 0, errors.New("token does not match node")
	}
	if requireIP && clientIP != "" && rec.ClientIP != "" && rec.ClientIP != clientIP {
		return 0, fmt.Errorf("token bound to %s, request from %s", rec.ClientIP, clientIP)
	}

	return rec.NodeID, nil
}

// ConsumePXEToken atomically marks a token consumed. Idempotent: a second
// consumption call on the same token returns an error so callers can detect
// replay.
func ConsumePXEToken(db *gorm.DB, token string) error {
	if token == "" {
		return errors.New("missing token")
	}
	now := time.Now()
	res := db.Model(&model.PXEProvisioningToken{}).
		Where("token_hash = ? AND consumed_at IS NULL", hashPXEToken(token)).
		Update("consumed_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("token already consumed or unknown")
	}
	return nil
}

// pxeTokenRequired reports whether PXE token gating is enabled. Default: true.
// Set PXE_REQUIRE_TOKEN=false to disable (legacy behavior — not recommended).
func pxeTokenRequired() bool {
	if v := os.Getenv("PXE_REQUIRE_TOKEN"); v != "" {
		switch v {
		case "0", "false", "False", "FALSE", "no", "No":
			return false
		}
	}
	return true
}
