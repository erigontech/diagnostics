package sessions

import (
	"crypto/rand"
	"math/big"
)

func generatePIN() (uint64, error) {
	//if insecure { TODO fix this
	//	return uint64(weakrand.Int63n(100_000_000)), nil
	//}
	_max := big.NewInt(100_000_000) // For an 8-digit PIN
	randNum, err := rand.Int(rand.Reader, _max)
	if err != nil {
		return 0, err
	}
	return randNum.Uint64(), nil
}
