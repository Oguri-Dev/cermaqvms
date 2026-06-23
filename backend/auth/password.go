package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword devuelve el hash bcrypt de una contraseña.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword compara una contraseña en claro contra su hash.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
