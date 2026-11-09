package platform

import "golang.org/x/crypto/bcrypt"

func testPassword(p string) (string, error) {
	b, e := bcrypt.GenerateFromPassword([]byte(p), bcrypt.MinCost)
	return string(b), e
}
