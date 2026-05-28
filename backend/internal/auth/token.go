package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

func Sign(secret string, claims Claims) (string, error) {
	if claims.Exp == 0 {
		claims.Exp = time.Now().Add(30 * 24 * time.Hour).Unix()
	}
	header, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	headerPart := base64.RawURLEncoding.EncodeToString(header)
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := headerPart + "." + payloadPart
	return signingInput + "." + sign(secret, signingInput), nil
}

func Parse(secret string, token string) (Claims, error) {
	var claims Claims
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		if !hmac.Equal([]byte(parts[2]), []byte(sign(secret, parts[0]+"."+parts[1]))) {
			return claims, errors.New("invalid signature")
		}
		headerPayload, err := base64.RawURLEncoding.DecodeString(parts[0])
		if err != nil {
			return claims, err
		}
		var header jwtHeader
		if err := json.Unmarshal(headerPayload, &header); err != nil {
			return claims, err
		}
		if header.Alg != "HS256" || header.Typ != "JWT" {
			return claims, errors.New("unsupported jwt header")
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return claims, err
		}
		if err := json.Unmarshal(payload, &claims); err != nil {
			return claims, err
		}
		return validateClaims(claims)
	}
	// Backward compatibility for tokens issued before the switch to standard JWT.
	if len(parts) == 2 {
		if !hmac.Equal([]byte(parts[1]), []byte(sign(secret, parts[0]))) {
			return claims, errors.New("invalid signature")
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[0])
		if err != nil {
			return claims, err
		}
		if err := json.Unmarshal(payload, &claims); err != nil {
			return claims, err
		}
		return validateClaims(claims)
	}
	return claims, errors.New("invalid token")
}

func validateClaims(claims Claims) (Claims, error) {
	if claims.Exp < time.Now().Unix() {
		return claims, errors.New("token expired")
	}
	return claims, nil
}

func sign(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func FromAuthorizationHeader(header string) (string, error) {
	if header == "" {
		return "", errors.New("missing authorization")
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("authorization must be Bearer token")
	}
	return parts[1], nil
}
