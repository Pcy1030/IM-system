package jwt

import (
	"errors"
	"fmt"
	"time"

	"im-system/config"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// JWTService 提供 JWT 生成与校验能力
// 使用对称密钥 HS256
// 仅存放不可逆的用户标识（例如用户ID）在 Subject
// 其他非敏感信息可放入 Data

type JWTService struct {
	secretKey   []byte        // 对称密钥
	issuer      string        // 签发者
	expireAfter time.Duration // 过期时间
}

// CustomClaims 自定义声明载荷
// Data 用于扩展非敏感业务字段

type CustomClaims struct {
	Data map[string]interface{} `json:"data,omitempty"`
	jwtv5.RegisteredClaims
}

// NewJWTService 创建 JWT 服务
func NewJWTService(cfg config.JWTConfig) *JWTService {
	return &JWTService{
		secretKey:   []byte(cfg.Secret),
		issuer:      cfg.Issuer,
		expireAfter: cfg.ExpireTime,
	}
}

// GenerateToken 生成访问令牌
// userID 作为 Subject 存入标准声明
// extraData 将写入 Data 字段（仅存放非敏感信息）
func (s *JWTService) GenerateToken(userID string, extraData map[string]interface{}) (string, error) {
	if userID == "" {
		return "", errors.New("userID is required")
	}

	now := time.Now()
	expiresAt := now.Add(s.expireAfter)

	claims := &CustomClaims{
		Data: extraData,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			IssuedAt:  jwtv5.NewNumericDate(now),
			NotBefore: jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(expiresAt),
		},
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("sign token failed: %w", err)
	}
	return signed, nil
}

// ValidateToken 校验并解析令牌
// 返回解析出的自定义声明（包含 Subject 和 Data）
func (s *JWTService) ValidateToken(tokenString string) (*CustomClaims, error) {
	if tokenString == "" {
		return nil, errors.New("token is empty")
	}
	// 解析令牌
	claims := &CustomClaims{}
	parsedToken, err := jwtv5.ParseWithClaims(
		tokenString, // 令牌字符串
		claims,      // 自定义声明
		// 验证签名方法
		func(token *jwtv5.Token) (interface{}, error) {
			// 验证签名方法
			if token.Method != jwtv5.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.secretKey, nil
		},
		// 验证签发者
		jwtv5.WithIssuer(s.issuer),
	)
	if err != nil {
		return nil, fmt.Errorf("parse token failed: %w", err)
	}
	if !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
