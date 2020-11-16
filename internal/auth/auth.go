package auth

import (
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/go-chi/jwtauth"
)

func JWTSecret(secret []byte, alg string) *jwtauth.JWTAuth {
	return jwtauth.New(alg, secret, nil)
}

type jwtPayload struct {
	Allow []string
}

func getJWTAlgorithm(alg string, secret []byte) *jwt.HMACSHA {
	var algFunc func([]byte) *jwt.HMACSHA
	switch alg {
	case "HS256":
		algFunc = jwt.NewHS256
	case "HS512":
		algFunc = jwt.NewHS512
	default:
		algFunc = jwt.NewHS256
	}
	return algFunc(secret)
}

func NewJWT(secret []byte, alg string, perms []string) ([]byte, error) {
	p := jwtPayload{
		Allow: perms,
	}
	return jwt.Sign(&p, getJWTAlgorithm(alg, secret))
}
