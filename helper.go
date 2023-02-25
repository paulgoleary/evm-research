package hub_research

import (
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/contract"
)

func TxnDoWait(txn contract.Txn, errIn error) error {
	if errIn != nil {
		return errIn
	}
	if err := txn.Do(); err != nil {
		return err
	} else {
		var rcpt *ethgo.Receipt
		if rcpt, err = txn.Wait(); err != nil {
			return err
		}
		_ = rcpt
	}
	return nil
}
