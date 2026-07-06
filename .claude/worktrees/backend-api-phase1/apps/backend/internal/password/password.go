package password

import "golang.org/x/crypto/bcrypt"

const cost = 10

func Hash(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func Verify(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
