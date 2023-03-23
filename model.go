package hub_research

import (
	"github.com/mitchellh/mapstructure"
	"math/big"
)

// struct Validator {
//    uint256[4] blsKey;
//    uint256 stake;
//    uint256 commission;
//    uint256 withdrawableRewards;
//    bool active;
//}

type Validator struct {
	BlsKey              [4]*big.Int
	Stake               *big.Int
	Commission          *big.Int
	WithdrawableRewards *big.Int
	Active              bool
}

func ValidatorFromMap(m map[string]interface{}) (v Validator, err error) {
	err = mapstructure.Decode(m, &v)
	return
}
